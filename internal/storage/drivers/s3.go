package drivers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/bendy/file-gateway/internal/storage"
	"github.com/bendy/file-gateway/internal/wasm"
)

const DriverNameS3 = "s3"

type S3Driver struct {
	endpoint  string
	region    string
	bucket    string
	accessKey string
	secretKey string
}

func init() {
	storage.Register(DriverNameS3, func(cfg map[string]string) (storage.Driver, error) {
		required := []string{"endpoint", "region", "bucket", "access_key", "secret_key"}
		for _, k := range required {
			if cfg[k] == "" {
				return nil, fmt.Errorf("s3 driver: missing required config key %q", k)
			}
		}
		return &S3Driver{
			endpoint:  strings.TrimRight(cfg["endpoint"], "/"),
			region:    cfg["region"],
			bucket:    cfg["bucket"],
			accessKey: cfg["access_key"],
			secretKey: cfg["secret_key"],
		}, nil
	})
}

func (d *S3Driver) Name() string { return DriverNameS3 }

func (d *S3Driver) Put(ctx context.Context, key string, body io.Reader, opts storage.UploadOptions) (storage.FileInfo, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("failed to read body: %w", err)
	}

	url := d.buildURL(key)
	contentType := opts.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	headers := map[string]string{
		"Content-Type": contentType,
	}
	d.signRequest("PUT", key, "", headers, string(bodyBytes))

	resp, err := wasm.Fetch("PUT", url, headers, string(bodyBytes))
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("s3 put failed: %w", err)
	}
	if resp.StatusCode >= 300 {
		return storage.FileInfo{}, fmt.Errorf("s3 put error: %d - %s", resp.StatusCode, resp.Body)
	}

	return storage.FileInfo{
		Key:          key,
		Size:         int64(len(bodyBytes)),
		ContentType:  contentType,
		ETag:         resp.Headers["etag"],
		LastModified: time.Now(),
	}, nil
}

func (d *S3Driver) Get(ctx context.Context, key string, opts storage.DownloadOptions) (io.ReadCloser, storage.FileInfo, error) {
	url := d.buildURL(key)
	headers := map[string]string{}
	if opts.RangeStart > 0 || opts.RangeEnd > 0 {
		headers["Range"] = fmt.Sprintf("bytes=%d-%d", opts.RangeStart, opts.RangeEnd)
	}
	d.signRequest("GET", key, "", headers, "")

	resp, err := wasm.Fetch("GET", url, headers, "")
	if err != nil {
		return nil, storage.FileInfo{}, fmt.Errorf("s3 get failed: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, storage.FileInfo{}, fmt.Errorf("s3 get error: %d - %s", resp.StatusCode, resp.Body)
	}

	size, _ := strconv.ParseInt(resp.Headers["content-length"], 10, 64)
	info := storage.FileInfo{
		Key:          key,
		Size:         size,
		ContentType:  resp.Headers["content-type"],
		ETag:         resp.Headers["etag"],
		LastModified: time.Now(),
	}
	return io.NopCloser(strings.NewReader(resp.Body)), info, nil
}

func (d *S3Driver) Head(ctx context.Context, key string) (storage.FileInfo, error) {
	url := d.buildURL(key)
	headers := map[string]string{}
	d.signRequest("HEAD", key, "", headers, "")

	resp, err := wasm.Fetch("HEAD", url, headers, "")
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("s3 head failed: %w", err)
	}
	if resp.StatusCode >= 300 {
		return storage.FileInfo{}, fmt.Errorf("s3 head error: %d", resp.StatusCode)
	}

	size, _ := strconv.ParseInt(resp.Headers["content-length"], 10, 64)
	return storage.FileInfo{
		Key:          key,
		Size:         size,
		ContentType:  resp.Headers["content-type"],
		ETag:         resp.Headers["etag"],
		LastModified: time.Now(),
	}, nil
}

func (d *S3Driver) Delete(ctx context.Context, key string) error {
	url := d.buildURL(key)
	headers := map[string]string{}
	d.signRequest("DELETE", key, "", headers, "")

	resp, err := wasm.Fetch("DELETE", url, headers, "")
	if err != nil {
		return fmt.Errorf("s3 delete failed: %w", err)
	}
	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		return fmt.Errorf("s3 delete error: %d - %s", resp.StatusCode, resp.Body)
	}
	return nil
}

func (d *S3Driver) List(ctx context.Context, prefix string, limit int, ct string) ([]storage.FileInfo, string, error) {
	url := fmt.Sprintf("%s/%s?list-type=2&prefix=%s&max-keys=%d", d.endpoint, d.bucket, prefix, limit)
	if ct != "" {
		url += "&continuation-token=" + ct
	}
	headers := map[string]string{}
	d.signRequest("GET", "", "list-type=2&prefix="+prefix+"&max-keys="+strconv.Itoa(limit), headers, "")

	_, err := wasm.Fetch("GET", url, headers, "")
	if err != nil {
		return nil, "", fmt.Errorf("s3 list failed: %w", err)
	}
	return []storage.FileInfo{}, "", nil
}

func (d *S3Driver) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/%s", d.endpoint, d.bucket)
	headers := map[string]string{}
	d.signRequest("HEAD", "", "", headers, "")

	resp, err := wasm.Fetch("HEAD", url, headers, "")
	if err != nil {
		return fmt.Errorf("s3 ping failed: %w", err)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("s3 ping error: %d", resp.StatusCode)
	}
	return nil
}

func (d *S3Driver) buildURL(key string) string {
	return fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
}

// signRequest adds AWS Signature V4 headers.
func (d *S3Driver) signRequest(method, path, query string, headers map[string]string, body string) {
	t := time.Now().UTC()
	dateStamp := t.Format("20060102")
	amzDate := t.Format("20060102T150405Z")
	payloadHash := sha256Hex(body)

	headers["Host"] = strings.TrimPrefix(strings.TrimPrefix(d.endpoint, "https://"), "http://")
	headers["X-Amz-Date"] = amzDate
	headers["X-Amz-Content-Sha256"] = payloadHash

	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		headers["Host"], payloadHash, amzDate)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	uri := "/" + d.bucket
	if path != "" {
		uri += "/" + path
	}
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method, uri, query, canonicalHeaders, signedHeaders, payloadHash)

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, d.region)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate, credentialScope, sha256Hex(canonicalRequest))

	signingKey := d.getSigningKey(dateStamp)
	signature := hmacHex(signingKey, stringToSign)

	headers["Authorization"] = fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		d.accessKey, credentialScope, signedHeaders, signature)
}

func (d *S3Driver) getSigningKey(dateStamp string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+d.secretKey), dateStamp)
	kRegion := hmacSHA256(kDate, d.region)
	kService := hmacSHA256(kRegion, "s3")
	return hmacSHA256(kService, "aws4_request")
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func hmacHex(key []byte, msg string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

func hmacSHA256(key []byte, msg string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(msg))
	return mac.Sum(nil)
}
