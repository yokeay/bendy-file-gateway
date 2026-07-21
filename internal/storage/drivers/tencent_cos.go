package drivers

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bendy/file-gateway/internal/storage"
	"github.com/bendy/file-gateway/internal/wasm"
)

// Tencent COS storage driver.
type tencentCOSDriver struct {
	region    string
	bucket    string
	secretID  string
	secretKey string
}

func init() {
	storage.Register("tencent_cos", func(cfg map[string]string) (storage.Driver, error) {
		required := []string{"region", "bucket", "secret_id", "secret_key"}
		for _, k := range required {
			if cfg[k] == "" {
				return nil, fmt.Errorf("tencent_cos driver: missing required config key %q", k)
			}
		}
		return &tencentCOSDriver{
			region:    cfg["region"],
			bucket:    cfg["bucket"],
			secretID:  cfg["secret_id"],
			secretKey: cfg["secret_key"],
		}, nil
	})
}

func (d *tencentCOSDriver) Name() string { return "tencent_cos" }

func (d *tencentCOSDriver) host() string {
	return d.bucket + ".cos." + d.region + ".myqcloud.com"
}

func (d *tencentCOSDriver) baseURL() string {
	return "https://" + d.host()
}

func (d *tencentCOSDriver) Put(ctx context.Context, key string, body io.Reader, opts storage.UploadOptions) (storage.FileInfo, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return storage.FileInfo{}, err
	}
	ct := opts.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	url := d.baseURL() + "/" + key
	headers := d.sign("PUT", "/"+key, "", ct, bodyBytes, 0)
	resp, err := wasm.Fetch("PUT", url, headers, string(bodyBytes))
	if err != nil {
		return storage.FileInfo{}, err
	}
	if resp.StatusCode >= 300 {
		return storage.FileInfo{}, fmt.Errorf("tencent cos put error: %d", resp.StatusCode)
	}
	return storage.FileInfo{
		Key: key, Size: int64(len(bodyBytes)), ContentType: ct,
		ETag: resp.Headers["etag"], LastModified: time.Now(),
	}, nil
}

func (d *tencentCOSDriver) Get(ctx context.Context, key string, opts storage.DownloadOptions) (io.ReadCloser, storage.FileInfo, error) {
	url := d.baseURL() + "/" + key
	headers := d.sign("GET", "/"+key, "", "", nil, 0)
	if opts.RangeStart > 0 || opts.RangeEnd > 0 {
		headers["Range"] = fmt.Sprintf("bytes=%d-%d", opts.RangeStart, opts.RangeEnd)
	}
	resp, err := wasm.Fetch("GET", url, headers, "")
	if err != nil {
		return nil, storage.FileInfo{}, err
	}
	if resp.StatusCode >= 300 {
		return nil, storage.FileInfo{}, fmt.Errorf("tencent cos get error: %d", resp.StatusCode)
	}
	size, _ := strconv.ParseInt(resp.Headers["content-length"], 10, 64)
	return io.NopCloser(strings.NewReader(resp.Body)), storage.FileInfo{
		Key: key, Size: size, ContentType: resp.Headers["content-type"],
		ETag: resp.Headers["etag"], LastModified: time.Now(),
	}, nil
}

func (d *tencentCOSDriver) Head(ctx context.Context, key string) (storage.FileInfo, error) {
	url := d.baseURL() + "/" + key
	headers := d.sign("HEAD", "/"+key, "", "", nil, 0)
	resp, err := wasm.Fetch("HEAD", url, headers, "")
	if err != nil {
		return storage.FileInfo{}, err
	}
	if resp.StatusCode >= 300 {
		return storage.FileInfo{}, fmt.Errorf("tencent cos head error: %d", resp.StatusCode)
	}
	size, _ := strconv.ParseInt(resp.Headers["content-length"], 10, 64)
	return storage.FileInfo{
		Key: key, Size: size, ContentType: resp.Headers["content-type"],
		ETag: resp.Headers["etag"], LastModified: time.Now(),
	}, nil
}

func (d *tencentCOSDriver) Delete(ctx context.Context, key string) error {
	url := d.baseURL() + "/" + key
	headers := d.sign("DELETE", "/"+key, "", "", nil, 0)
	resp, err := wasm.Fetch("DELETE", url, headers, "")
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		return fmt.Errorf("tencent cos delete error: %d", resp.StatusCode)
	}
	return nil
}

func (d *tencentCOSDriver) List(ctx context.Context, prefix string, limit int, ct string) ([]storage.FileInfo, string, error) {
	path := "/?prefix=" + url.QueryEscape(prefix) + "&max-keys=" + strconv.Itoa(limit)
	if ct != "" {
		path += "&marker=" + url.QueryEscape(ct)
	}
	url := d.baseURL() + path
	headers := d.sign("GET", path, "", "", nil, 0)
	_, err := wasm.Fetch("GET", url, headers, "")
	if err != nil {
		return nil, "", err
	}
	return []storage.FileInfo{}, "", nil
}

func (d *tencentCOSDriver) Ping(ctx context.Context) error {
	url := d.baseURL() + "/"
	headers := d.sign("HEAD", "/", "", "", nil, 0)
	resp, err := wasm.Fetch("HEAD", url, headers, "")
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("tencent cos ping error: %d", resp.StatusCode)
	}
	return nil
}

// sign generates the Tencent COS Authorization header using the COS signing algorithm.
func (d *tencentCOSDriver) sign(method, path, query, contentType string, body []byte, signExpires time.Duration) map[string]string {
	now := time.Now()
	start := now.Unix()
	end := now.Add(3600).Unix()
	if signExpires > 0 {
		end = now.Add(signExpires).Unix()
	}
	keyTime := strconv.FormatInt(start, 10) + ";" + strconv.FormatInt(end, 10)
	signKey := d.cosHMAC(keyTime)

	// HttpParameters (empty for most operations, included in path for list)
	httpParams := query
	httpHeaders := ""
	if contentType != "" {
		httpHeaders += "content-type=" + url.QueryEscape(strings.ToLower(contentType)) + "&"
	}
	httpHeaders += "host=" + url.QueryEscape(d.host())
	httpString := strings.ToLower(method) + "\n" + path + "\n" + httpParams + "\n" + httpHeaders + "\n"

	sha1Hash := sha1.Sum([]byte(httpString))
	sha1Hex := hex.EncodeToString(sha1Hash[:])

	stringToSign := "sha1\n" + keyTime + "\n" + sha1Hex + "\n"
	mac := hmac.New(sha1.New, []byte(signKey))
	mac.Write([]byte(stringToSign))
	signature := hex.EncodeToString(mac.Sum(nil))

	return map[string]string{
		"Host":         d.host(),
		"Content-Type": contentType,
		"Authorization": fmt.Sprintf(
			"q-sign-algorithm=sha1&q-ak=%s&q-sign-time=%s&q-key-time=%s&q-header-list=content-type;host&q-url-param-list=&q-signature=%s",
			d.secretID, keyTime, keyTime, signature),
	}
}

func (d *tencentCOSDriver) cosHMAC(keyTime string) string {
	mac := hmac.New(sha1.New, []byte(d.secretKey))
	mac.Write([]byte(keyTime))
	return hex.EncodeToString(mac.Sum(nil))
}
