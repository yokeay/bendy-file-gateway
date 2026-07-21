package drivers

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/bendy/file-gateway/internal/storage"
	"github.com/bendy/file-gateway/internal/wasm"
)

type aliyunOSDriver struct {
	endpoint    string
	bucket      string
	accessKeyID string
	accessKey   string
}

func init() {
	storage.Register("aliyun_oss", func(cfg map[string]string) (storage.Driver, error) {
		required := []string{"endpoint", "bucket", "access_key_id", "access_key_secret"}
		for _, k := range required {
			if cfg[k] == "" {
				return nil, fmt.Errorf("aliyun_oss driver: missing required config key %q", k)
			}
		}
		return &aliyunOSDriver{
			endpoint:    strings.TrimRight(cfg["endpoint"], "/"),
			bucket:      cfg["bucket"],
			accessKeyID: cfg["access_key_id"],
			accessKey:   cfg["access_key_secret"],
		}, nil
	})
}

func (d *aliyunOSDriver) Name() string { return "aliyun_oss" }

func (d *aliyunOSDriver) Put(ctx context.Context, key string, body io.Reader, opts storage.UploadOptions) (storage.FileInfo, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return storage.FileInfo{}, err
	}
	ct := opts.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
	headers := d.sign("PUT", "/"+d.bucket+"/"+key, ct, bodyBytes)
	resp, err := wasm.Fetch("PUT", url, headers, string(bodyBytes))
	if err != nil {
		return storage.FileInfo{}, err
	}
	if resp.StatusCode >= 300 {
		return storage.FileInfo{}, fmt.Errorf("aliyun oss put error: %d", resp.StatusCode)
	}
	return storage.FileInfo{
		Key: key, Size: int64(len(bodyBytes)), ContentType: ct,
		ETag: resp.Headers["etag"], LastModified: time.Now(),
	}, nil
}

func (d *aliyunOSDriver) Get(ctx context.Context, key string, opts storage.DownloadOptions) (io.ReadCloser, storage.FileInfo, error) {
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
	headers := d.sign("GET", "/"+d.bucket+"/"+key, "", nil)
	if opts.RangeStart > 0 || opts.RangeEnd > 0 {
		headers["Range"] = fmt.Sprintf("bytes=%d-%d", opts.RangeStart, opts.RangeEnd)
	}
	resp, err := wasm.Fetch("GET", url, headers, "")
	if err != nil {
		return nil, storage.FileInfo{}, err
	}
	if resp.StatusCode >= 300 {
		return nil, storage.FileInfo{}, fmt.Errorf("aliyun oss get error: %d", resp.StatusCode)
	}
	size, _ := strconv.ParseInt(resp.Headers["content-length"], 10, 64)
	return io.NopCloser(strings.NewReader(resp.Body)), storage.FileInfo{
		Key: key, Size: size, ContentType: resp.Headers["content-type"],
		ETag: resp.Headers["etag"], LastModified: time.Now(),
	}, nil
}

func (d *aliyunOSDriver) Head(ctx context.Context, key string) (storage.FileInfo, error) {
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
	headers := d.sign("HEAD", "/"+d.bucket+"/"+key, "", nil)
	resp, err := wasm.Fetch("HEAD", url, headers, "")
	if err != nil {
		return storage.FileInfo{}, err
	}
	if resp.StatusCode >= 300 {
		return storage.FileInfo{}, fmt.Errorf("aliyun oss head error: %d", resp.StatusCode)
	}
	size, _ := strconv.ParseInt(resp.Headers["content-length"], 10, 64)
	return storage.FileInfo{
		Key: key, Size: size, ContentType: resp.Headers["content-type"],
		ETag: resp.Headers["etag"], LastModified: time.Now(),
	}, nil
}

func (d *aliyunOSDriver) Delete(ctx context.Context, key string) error {
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
	headers := d.sign("DELETE", "/"+d.bucket+"/"+key, "", nil)
	resp, err := wasm.Fetch("DELETE", url, headers, "")
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		return fmt.Errorf("aliyun oss delete error: %d", resp.StatusCode)
	}
	return nil
}

func (d *aliyunOSDriver) List(ctx context.Context, prefix string, limit int, ct string) ([]storage.FileInfo, string, error) {
	url := fmt.Sprintf("%s/%s?prefix=%s&max-keys=%d", d.endpoint, d.bucket, prefix, limit)
	if ct != "" {
		url += "&marker=" + ct
	}
	headers := d.sign("GET", "/"+d.bucket+"/", "", nil)
	_, err := wasm.Fetch("GET", url, headers, "")
	if err != nil {
		return nil, "", err
	}
	return []storage.FileInfo{}, "", nil
}

func (d *aliyunOSDriver) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/%s", d.endpoint, d.bucket)
	headers := d.sign("HEAD", "/"+d.bucket, "", nil)
	resp, err := wasm.Fetch("HEAD", url, headers, "")
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("aliyun oss ping error: %d", resp.StatusCode)
	}
	return nil
}

// sign generates Alibaba Cloud OSS Authorization header.
// Signature = base64(hmac-sha1(accessKey, VERB + "\n" + MD5 + "\n" + Type + "\n" + Date + "\n" + OSSHeaders + "\n" + Resource))
func (d *aliyunOSDriver) sign(method, resource, contentType string, body []byte) map[string]string {
	date := time.Now().UTC().Format(httpTimeFormat)
	md5 := ""
	if body != nil {
		h := sha256.Sum256(body)
		md5 = base64.StdEncoding.EncodeToString(h[:])
	}
	stringToSign := method + "\n" + md5 + "\n" + contentType + "\n" + date + "\n" + resource
	mac := hmac.New(sha1.New, []byte(d.accessKey))
	mac.Write([]byte(stringToSign))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return map[string]string{
		"Date":          date,
		"Content-Type":  contentType,
		"Authorization": "OSS " + d.accessKeyID + ":" + sig,
	}
}

const httpTimeFormat = time.RFC1123
