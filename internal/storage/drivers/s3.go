package drivers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bendy/file-gateway/internal/storage"
)

const DriverNameS3 = "s3"

// S3Driver implements storage.Driver for S3-compatible storage.
type S3Driver struct {
	endpoint  string
	region    string
	bucket    string
	accessKey string
	secretKey string
	client    *http.Client
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
			client:    &http.Client{Timeout: 30 * time.Second},
		}, nil
	})
}

func (d *S3Driver) Name() string { return DriverNameS3 }

func (d *S3Driver) Put(ctx context.Context, key string, body io.Reader, opts storage.UploadOptions) (storage.FileInfo, error) {
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)

	// Read body for signing
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("failed to read body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return storage.FileInfo{}, err
	}

	contentType := opts.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req.Header.Set("Content-Type", contentType)

	// Sign the request with AWS Signature V4
	d.signRequest(req, string(bodyBytes))

	resp, err := d.client.Do(req)
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("s3 put failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return storage.FileInfo{}, fmt.Errorf("s3 put error: %d - %s", resp.StatusCode, string(respBody))
	}

	return storage.FileInfo{
		Key:          key,
		Size:         int64(len(bodyBytes)),
		ContentType:  contentType,
		ETag:         strings.Trim(resp.Header.Get("ETag"), "\""),
		LastModified: time.Now(),
	}, nil
}

func (d *S3Driver) Get(ctx context.Context, key string, opts storage.DownloadOptions) (io.ReadCloser, storage.FileInfo, error) {
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, storage.FileInfo{}, err
	}

	if opts.RangeStart > 0 || opts.RangeEnd > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", opts.RangeStart, opts.RangeEnd))
	}

	d.signRequest(req, "")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, storage.FileInfo{}, fmt.Errorf("s3 get failed: %w", err)
	}

	if resp.StatusCode >= 300 {
		resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, storage.FileInfo{}, fmt.Errorf("s3 get error: %d - %s", resp.StatusCode, string(respBody))
	}

	info := storage.FileInfo{
		Key:          key,
		Size:         resp.ContentLength,
		ContentType:  resp.Header.Get("Content-Type"),
		ETag:         strings.Trim(resp.Header.Get("ETag"), "\""),
		LastModified: time.Now(),
	}

	return resp.Body, info, nil
}

func (d *S3Driver) Head(ctx context.Context, key string) (storage.FileInfo, error) {
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return storage.FileInfo{}, err
	}

	d.signRequest(req, "")

	resp, err := d.client.Do(req)
	if err != nil {
		return storage.FileInfo{}, fmt.Errorf("s3 head failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return storage.FileInfo{}, fmt.Errorf("s3 head error: %d", resp.StatusCode)
	}

	return storage.FileInfo{
		Key:          key,
		Size:         resp.ContentLength,
		ContentType:  resp.Header.Get("Content-Type"),
		ETag:         strings.Trim(resp.Header.Get("ETag"), "\""),
		LastModified: time.Now(),
	}, nil
}

func (d *S3Driver) Delete(ctx context.Context, key string) error {
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	d.signRequest(req, "")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("s3 delete failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 delete error: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (d *S3Driver) List(ctx context.Context, prefix string, limit int, continuationToken string) ([]storage.FileInfo, string, error) {
	url := fmt.Sprintf("%s/%s?list-type=2&prefix=%s&max-keys=%d", d.endpoint, d.bucket, prefix, limit)
	if continuationToken != "" {
		url += "&continuation-token=" + continuationToken
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}

	d.signRequest(req, "")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("s3 list failed: %w", err)
	}
	defer resp.Body.Close()

	// Basic XML parsing - full implementation will use a proper XML parser
	// For now, return an empty list (will be implemented in Phase 3)
	return []storage.FileInfo{}, "", nil
}

func (d *S3Driver) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/%s", d.endpoint, d.bucket)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return err
	}
	d.signRequest(req, "")
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("s3 ping failed: %w", err)
	}
	resp.Body.Close()
	return nil
}

// signRequest adds AWS Signature V4 authorization headers.
func (d *S3Driver) signRequest(req *http.Request, body string) {
	t := time.Now().UTC()
	dateStamp := t.Format("20060102")
	amzDate := t.Format("20060102T150405Z")

	// Create hashed payload
	payloadHash := sha256Hex(body)

	// Set required headers
	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	// Create canonical request
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		req.URL.Host, payloadHash, amzDate)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method,
		req.URL.EscapedPath(),
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	)

	// Create string to sign
	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, d.region)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate,
		credentialScope,
		sha256Hex(canonicalRequest),
	)

	// Calculate signing key
	signingKey := d.getSigningKey(dateStamp)

	// Calculate signature
	signature := hmacHex(signingKey, stringToSign)

	// Add authorization header
	req.Header.Set("Authorization",
		fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
			d.accessKey, credentialScope, signedHeaders, signature))
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
