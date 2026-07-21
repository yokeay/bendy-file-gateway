package drivers

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/bendy/file-gateway/internal/storage"
	"github.com/bendy/file-gateway/internal/wasm"
)

// Huawei OBS uses an S3-compatible API with Huawei-specific SigV4 signing.
type huaweiOBSDriver struct {
	endpoint  string
	bucket    string
	accessKey string
	secretKey string
}

func init() {
	storage.Register("huawei_obs", func(cfg map[string]string) (storage.Driver, error) {
		required := []string{"endpoint", "bucket", "access_key_id", "access_key_secret"}
		for _, k := range required {
			if cfg[k] == "" {
				return nil, fmt.Errorf("huawei_obs driver: missing required config key %q", k)
			}
		}
		return &huaweiOBSDriver{
			endpoint:  strings.TrimRight(cfg["endpoint"], "/"),
			bucket:    cfg["bucket"],
			accessKey: cfg["access_key_id"],
			secretKey: cfg["access_key_secret"],
		}, nil
	})
}

func (d *huaweiOBSDriver) Name() string { return "huawei_obs" }

func (d *huaweiOBSDriver) Put(ctx context.Context, key string, body io.Reader, opts storage.UploadOptions) (storage.FileInfo, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return storage.FileInfo{}, err
	}
	ct := opts.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
	headers := d.sign("PUT", key, ct, string(bodyBytes))
	resp, err := wasm.Fetch("PUT", url, headers, string(bodyBytes))
	if err != nil {
		return storage.FileInfo{}, err
	}
	if resp.StatusCode >= 300 {
		return storage.FileInfo{}, fmt.Errorf("huawei obs put error: %d", resp.StatusCode)
	}
	return storage.FileInfo{
		Key: key, Size: int64(len(bodyBytes)), ContentType: ct,
		ETag: resp.Headers["etag"], LastModified: time.Now(),
	}, nil
}

func (d *huaweiOBSDriver) Get(ctx context.Context, key string, opts storage.DownloadOptions) (io.ReadCloser, storage.FileInfo, error) {
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
	headers := d.sign("GET", key, "", "")
	if opts.RangeStart > 0 || opts.RangeEnd > 0 {
		headers["Range"] = fmt.Sprintf("bytes=%d-%d", opts.RangeStart, opts.RangeEnd)
	}
	resp, err := wasm.Fetch("GET", url, headers, "")
	if err != nil {
		return nil, storage.FileInfo{}, err
	}
	if resp.StatusCode >= 300 {
		return nil, storage.FileInfo{}, fmt.Errorf("huawei obs get error: %d", resp.StatusCode)
	}
	size, _ := strconv.ParseInt(resp.Headers["content-length"], 10, 64)
	return io.NopCloser(strings.NewReader(resp.Body)), storage.FileInfo{
		Key: key, Size: size, ContentType: resp.Headers["content-type"],
		ETag: resp.Headers["etag"], LastModified: time.Now(),
	}, nil
}

func (d *huaweiOBSDriver) Head(ctx context.Context, key string) (storage.FileInfo, error) {
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
	headers := d.sign("HEAD", key, "", "")
	resp, err := wasm.Fetch("HEAD", url, headers, "")
	if err != nil {
		return storage.FileInfo{}, err
	}
	if resp.StatusCode >= 300 {
		return storage.FileInfo{}, fmt.Errorf("huawei obs head error: %d", resp.StatusCode)
	}
	size, _ := strconv.ParseInt(resp.Headers["content-length"], 10, 64)
	return storage.FileInfo{
		Key: key, Size: size, ContentType: resp.Headers["content-type"],
		ETag: resp.Headers["etag"], LastModified: time.Now(),
	}, nil
}

func (d *huaweiOBSDriver) Delete(ctx context.Context, key string) error {
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
	headers := d.sign("DELETE", key, "", "")
	resp, err := wasm.Fetch("DELETE", url, headers, "")
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		return fmt.Errorf("huawei obs delete error: %d", resp.StatusCode)
	}
	return nil
}

func (d *huaweiOBSDriver) List(ctx context.Context, prefix string, limit int, ct string) ([]storage.FileInfo, string, error) {
	url := fmt.Sprintf("%s/%s?prefix=%s&max-keys=%d", d.endpoint, d.bucket, prefix, limit)
	if ct != "" {
		url += "&marker=" + ct
	}
	headers := d.sign("GET", "", "", "")
	_, err := wasm.Fetch("GET", url, headers, "")
	if err != nil {
		return nil, "", err
	}
	return []storage.FileInfo{}, "", nil
}

func (d *huaweiOBSDriver) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/%s", d.endpoint, d.bucket)
	headers := d.sign("HEAD", "", "", "")
	resp, err := wasm.Fetch("HEAD", url, headers, "")
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("huawei obs ping error: %d", resp.StatusCode)
	}
	return nil
}

func (d *huaweiOBSDriver) sign(method, key, contentType, body string) map[string]string {
	t := time.Now().UTC()
	dateStamp := t.Format("20060102")
	amzDate := t.Format("20060102T150405Z")
	payloadHash := sha256Hex(body)

	host := strings.TrimPrefix(strings.TrimPrefix(d.endpoint, "https://"), "http://")
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		host, payloadHash, amzDate)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	resource := "/" + d.bucket
	if key != "" {
		resource += "/" + key
	}
	canonicalRequest := method + "\n" + resource + "\n\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + payloadHash
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + dateStamp + "/obs/aws4_request\n" + sha256Hex(canonicalRequest)

	signingKey := hmacSHA256([]byte("AWS4"+d.secretKey), dateStamp)
	signingKey = hmacSHA256(signingKey, "obs")
	signingKey = hmacSHA256(signingKey, "aws4_request")
	sig := hmacHex(signingKey, stringToSign)

	return map[string]string{
		"Host":                   host,
		"X-Amz-Date":             amzDate,
		"X-Amz-Content-Sha256":   payloadHash,
		"Content-Type":           contentType,
		"Authorization":          "AWS4-HMAC-SHA256 Credential=" + d.accessKey + "/" + dateStamp + "/obs/aws4_request, SignedHeaders=" + signedHeaders + ", Signature=" + sig,
	}
}
