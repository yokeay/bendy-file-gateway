package drivers

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/bendy/file-gateway/internal/storage"
	"github.com/bendy/file-gateway/internal/wasm"
)

// Qiniu Kodo storage driver.
type qiniuKodoDriver struct {
	accessKey string
	secretKey string
	bucket    string
	domain    string
	useHTTPS  bool
}

func init() {
	storage.Register("qiniu_kodo", func(cfg map[string]string) (storage.Driver, error) {
		required := []string{"access_key", "secret_key", "bucket", "domain"}
		for _, k := range required {
			if cfg[k] == "" {
				return nil, fmt.Errorf("qiniu_kodo driver: missing required config key %q", k)
			}
		}
		return &qiniuKodoDriver{
			accessKey: cfg["access_key"],
			secretKey: cfg["secret_key"],
			bucket:    cfg["bucket"],
			domain:    strings.TrimRight(cfg["domain"], "/"),
			useHTTPS:  cfg["use_https"] == "true" || cfg["use_https"] == "1",
		}, nil
	})
}

func (d *qiniuKodoDriver) Name() string { return "qiniu_kodo" }

func (d *qiniuKodoDriver) baseURL() string {
	scheme := "http"
	if d.useHTTPS {
		scheme = "https"
	}
	return scheme + "://" + d.domain
}

func (d *qiniuKodoDriver) Put(ctx context.Context, key string, body io.Reader, opts storage.UploadOptions) (storage.FileInfo, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return storage.FileInfo{}, err
	}
	ct := opts.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	// Generate upload token for Qiniu
	uploadToken := d.genUploadToken(key)

	url := d.baseURL() + "/" + key
	headers := map[string]string{
		"Content-Type":  ct,
		"Authorization": "UpToken " + uploadToken,
	}
	resp, err := wasm.Fetch("PUT", url, headers, string(bodyBytes))
	if err != nil {
		return storage.FileInfo{}, err
	}
	if resp.StatusCode >= 300 {
		return storage.FileInfo{}, fmt.Errorf("qiniu kodo put error: %d - %s", resp.StatusCode, resp.Body)
	}
	return storage.FileInfo{
		Key: key, Size: int64(len(bodyBytes)), ContentType: ct,
		ETag: resp.Headers["etag"], LastModified: time.Now(),
	}, nil
}

func (d *qiniuKodoDriver) Get(ctx context.Context, key string, opts storage.DownloadOptions) (io.ReadCloser, storage.FileInfo, error) {
	url := d.baseURL() + "/" + key
	headers := d.sign("GET", "/"+key, "")
	if opts.RangeStart > 0 || opts.RangeEnd > 0 {
		headers["Range"] = fmt.Sprintf("bytes=%d-%d", opts.RangeStart, opts.RangeEnd)
	}
	resp, err := wasm.Fetch("GET", url, headers, "")
	if err != nil {
		return nil, storage.FileInfo{}, err
	}
	if resp.StatusCode >= 300 {
		return nil, storage.FileInfo{}, fmt.Errorf("qiniu kodo get error: %d", resp.StatusCode)
	}
	size, _ := strconv.ParseInt(resp.Headers["content-length"], 10, 64)
	return io.NopCloser(strings.NewReader(resp.Body)), storage.FileInfo{
		Key: key, Size: size, ContentType: resp.Headers["content-type"],
		ETag: resp.Headers["etag"], LastModified: time.Now(),
	}, nil
}

func (d *qiniuKodoDriver) Head(ctx context.Context, key string) (storage.FileInfo, error) {
	url := d.baseURL() + "/" + key
	headers := d.sign("GET", "/"+key+"?stat", "")
	statURL := url + "?stat"
	resp, err := wasm.Fetch("GET", statURL, headers, "")
	if err != nil {
		return storage.FileInfo{}, err
	}
	if resp.StatusCode >= 300 {
		return storage.FileInfo{}, fmt.Errorf("qiniu kodo head error: %d", resp.StatusCode)
	}
	size, _ := strconv.ParseInt(resp.Headers["content-length"], 10, 64)
	return storage.FileInfo{
		Key: key, Size: size, ContentType: resp.Headers["content-type"],
		ETag: resp.Headers["etag"], LastModified: time.Now(),
	}, nil
}

func (d *qiniuKodoDriver) Delete(ctx context.Context, key string) error {
	url := d.baseURL() + "/" + key
	headers := d.sign("DELETE", "/"+key, "")
	resp, err := wasm.Fetch("DELETE", url, headers, "")
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		return fmt.Errorf("qiniu kodo delete error: %d", resp.StatusCode)
	}
	return nil
}

func (d *qiniuKodoDriver) List(ctx context.Context, prefix string, limit int, ct string) ([]storage.FileInfo, string, error) {
	host := "rsf.qbox.me"
	path := "/list?bucket=" + d.bucket + "&prefix=" + prefix + "&limit=" + strconv.Itoa(limit)
	if ct != "" {
		path += "&marker=" + ct
	}
	url := "https://" + host + path
	headers := d.sign("GET", path, "")
	_, err := wasm.Fetch("GET", url, headers, "")
	if err != nil {
		return nil, "", err
	}
	return []storage.FileInfo{}, "", nil
}

func (d *qiniuKodoDriver) Ping(ctx context.Context) error {
	host := "rsf.qbox.me"
	path := "/list?bucket=" + d.bucket + "&limit=1"
	url := "https://" + host + path
	headers := d.sign("GET", path, "")
	resp, err := wasm.Fetch("GET", url, headers, "")
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("qiniu kodo ping error: %d", resp.StatusCode)
	}
	return nil
}

// sign generates Qiniu management API signature.
// Sign = base64(hmac_sha1(secretKey, pathAndQuery + "\n"))
func (d *qiniuKodoDriver) sign(method, pathAndQuery, body string) map[string]string {
	data := pathAndQuery + "\n" + body
	mac := hmac.New(sha1.New, []byte(d.secretKey))
	mac.Write([]byte(data))
	sign := base64.URLEncoding.EncodeToString(mac.Sum(nil))
	token := d.accessKey + ":" + sign
	return map[string]string{
		"Authorization": "Qiniu " + token,
	}
}

// genUploadToken generates an upload token for the given key.
func (d *qiniuKodoDriver) genUploadToken(key string) string {
	deadline := time.Now().Unix() + 3600
	policy := map[string]interface{}{
		"scope":    d.bucket + ":" + key,
		"deadline": deadline,
	}
	policyJSON, _ := json.Marshal(policy)
	encodedPolicy := base64.URLEncoding.EncodeToString(policyJSON)
	mac := hmac.New(sha1.New, []byte(d.secretKey))
	mac.Write([]byte(encodedPolicy))
	sign := base64.URLEncoding.EncodeToString(mac.Sum(nil))
	return d.accessKey + ":" + sign + ":" + encodedPolicy
}
