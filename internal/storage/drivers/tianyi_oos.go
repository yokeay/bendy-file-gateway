package drivers

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/bendy/file-gateway/internal/storage"
	"github.com/bendy/file-gateway/internal/wasm"
)

// China Telecom Tianyi OOS (e-Surfing Cloud Object Storage) driver.
// Uses S3-compatible API with Signature V2-style auth.
type tianyiOOSDriver struct {
	endpoint  string
	bucket    string
	accessKey string
	secretKey string
}

func init() {
	storage.Register("tianyi_oos", func(cfg map[string]string) (storage.Driver, error) {
		required := []string{"endpoint", "bucket", "access_key", "secret_key"}
		for _, k := range required {
			if cfg[k] == "" {
				return nil, fmt.Errorf("tianyi_oos driver: missing required config key %q", k)
			}
		}
		return &tianyiOOSDriver{
			endpoint:  strings.TrimRight(cfg["endpoint"], "/"),
			bucket:    cfg["bucket"],
			accessKey: cfg["access_key"],
			secretKey: cfg["secret_key"],
		}, nil
	})
}

func (d *tianyiOOSDriver) Name() string { return "tianyi_oos" }

func (d *tianyiOOSDriver) Put(ctx context.Context, key string, body io.Reader, opts storage.UploadOptions) (storage.FileInfo, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return storage.FileInfo{}, err
	}
	ct := opts.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
	headers := d.sign("PUT", "/"+d.bucket+"/"+key, ct)
	resp, err := wasm.Fetch("PUT", url, headers, string(bodyBytes))
	if err != nil {
		return storage.FileInfo{}, err
	}
	if resp.StatusCode >= 300 {
		return storage.FileInfo{}, fmt.Errorf("tianyi oos put error: %d", resp.StatusCode)
	}
	return storage.FileInfo{
		Key: key, Size: int64(len(bodyBytes)), ContentType: ct,
		ETag: resp.Headers["etag"], LastModified: time.Now(),
	}, nil
}

func (d *tianyiOOSDriver) Get(ctx context.Context, key string, opts storage.DownloadOptions) (io.ReadCloser, storage.FileInfo, error) {
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
	headers := d.sign("GET", "/"+d.bucket+"/"+key, "")
	if opts.RangeStart > 0 || opts.RangeEnd > 0 {
		headers["Range"] = fmt.Sprintf("bytes=%d-%d", opts.RangeStart, opts.RangeEnd)
	}
	resp, err := wasm.Fetch("GET", url, headers, "")
	if err != nil {
		return nil, storage.FileInfo{}, err
	}
	if resp.StatusCode >= 300 {
		return nil, storage.FileInfo{}, fmt.Errorf("tianyi oos get error: %d", resp.StatusCode)
	}
	size, _ := strconv.ParseInt(resp.Headers["content-length"], 10, 64)
	return io.NopCloser(strings.NewReader(resp.Body)), storage.FileInfo{
		Key: key, Size: size, ContentType: resp.Headers["content-type"],
		ETag: resp.Headers["etag"], LastModified: time.Now(),
	}, nil
}

func (d *tianyiOOSDriver) Head(ctx context.Context, key string) (storage.FileInfo, error) {
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
	headers := d.sign("HEAD", "/"+d.bucket+"/"+key, "")
	resp, err := wasm.Fetch("HEAD", url, headers, "")
	if err != nil {
		return storage.FileInfo{}, err
	}
	if resp.StatusCode >= 300 {
		return storage.FileInfo{}, fmt.Errorf("tianyi oos head error: %d", resp.StatusCode)
	}
	size, _ := strconv.ParseInt(resp.Headers["content-length"], 10, 64)
	return storage.FileInfo{
		Key: key, Size: size, ContentType: resp.Headers["content-type"],
		ETag: resp.Headers["etag"], LastModified: time.Now(),
	}, nil
}

func (d *tianyiOOSDriver) Delete(ctx context.Context, key string) error {
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
	headers := d.sign("DELETE", "/"+d.bucket+"/"+key, "")
	resp, err := wasm.Fetch("DELETE", url, headers, "")
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		return fmt.Errorf("tianyi oos delete error: %d", resp.StatusCode)
	}
	return nil
}

func (d *tianyiOOSDriver) List(ctx context.Context, prefix string, limit int, ct string) ([]storage.FileInfo, string, error) {
	url := fmt.Sprintf("%s/%s?prefix=%s&max-keys=%d", d.endpoint, d.bucket, prefix, limit)
	if ct != "" {
		url += "&marker=" + ct
	}
	headers := d.sign("GET", "/"+d.bucket+"/", "")
	_, err := wasm.Fetch("GET", url, headers, "")
	if err != nil {
		return nil, "", err
	}
	return []storage.FileInfo{}, "", nil
}

func (d *tianyiOOSDriver) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/%s", d.endpoint, d.bucket)
	headers := d.sign("HEAD", "/"+d.bucket, "")
	resp, err := wasm.Fetch("HEAD", url, headers, "")
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("tianyi oos ping error: %d", resp.StatusCode)
	}
	return nil
}

// sign generates Signature V2-style auth for Tianyi OOS.
func (d *tianyiOOSDriver) sign(method, resource, contentType string) map[string]string {
	date := time.Now().UTC().Format(httpTimeFormat)
	stringToSign := method + "\n\n" + contentType + "\n" + date + "\n" + resource
	mac := hmac.New(sha1.New, []byte(d.secretKey))
	mac.Write([]byte(stringToSign))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return map[string]string{
		"Date":          date,
		"Content-Type":  contentType,
		"Authorization": "AWS " + d.accessKey + ":" + sig,
	}
}
