package drivers

import (
	"context"
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

// Redis storage driver using Upstash Redis HTTP API.
type redisDriver struct {
	addr  string
	token string
}

func init() {
	storage.Register("redis", func(cfg map[string]string) (storage.Driver, error) {
		required := []string{"addr", "token"}
		for _, k := range required {
			if cfg[k] == "" {
				return nil, fmt.Errorf("redis driver: missing required config key %q", k)
			}
		}
		return &redisDriver{
			addr:  strings.TrimRight(cfg["addr"], "/"),
			token: cfg["token"],
		}, nil
	})
}

func (d *redisDriver) Name() string { return "redis" }

func (d *redisDriver) authHeaders() map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + d.token,
		"Content-Type":  "application/json",
	}
}

func (d *redisDriver) redisCmd(cmd []interface{}) (*wasm.FetchResponse, error) {
	body, _ := json.Marshal(cmd)
	return wasm.Fetch("POST", d.addr, d.authHeaders(), string(body))
}

func (d *redisDriver) Put(ctx context.Context, key string, body io.Reader, opts storage.UploadOptions) (storage.FileInfo, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return storage.FileInfo{}, err
	}
	b64Data := base64.StdEncoding.EncodeToString(bodyBytes)
	ct := opts.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}

	// Store file data and metadata
	metaJSON, _ := json.Marshal(map[string]string{
		"content-type": ct,
		"size":         strconv.FormatInt(int64(len(bodyBytes)), 10),
	})
	storageKey := "bendy:file:" + key
	metaKey := "bendy:meta:" + key

	// SET file data
	_, err = d.redisCmd([]interface{}{"SET", storageKey, b64Data})
	if err != nil {
		return storage.FileInfo{}, err
	}
	// SET metadata
	_, err = d.redisCmd([]interface{}{"SET", metaKey, string(metaJSON)})
	if err != nil {
		return storage.FileInfo{}, err
	}

	return storage.FileInfo{
		Key: key, Size: int64(len(bodyBytes)), ContentType: ct,
		LastModified: time.Now(),
	}, nil
}

func (d *redisDriver) Get(ctx context.Context, key string, opts storage.DownloadOptions) (io.ReadCloser, storage.FileInfo, error) {
	storageKey := "bendy:file:" + key
	resp, err := d.redisCmd([]interface{}{"GET", storageKey})
	if err != nil {
		return nil, storage.FileInfo{}, err
	}
	if resp.StatusCode != 200 {
		return nil, storage.FileInfo{}, fmt.Errorf("redis get error: %d", resp.StatusCode)
	}

	var result struct{ Result *string `json:"result"` }
	json.Unmarshal([]byte(resp.Body), &result)
	if result.Result == nil {
		return nil, storage.FileInfo{}, fmt.Errorf("file not found: %s", key)
	}
	b64Data := *result.Result

	// Get metadata for content type
	metaKey := "bendy:meta:" + key
	metaResp, _ := d.redisCmd([]interface{}{"GET", metaKey})
	var metaResult struct{ Result *string `json:"result"` }
	var meta map[string]string
	if metaResp != nil {
		json.Unmarshal([]byte(metaResp.Body), &metaResult)
		if metaResult.Result != nil {
			json.Unmarshal([]byte(*metaResult.Result), &meta)
		}
	}

	data, _ := base64.StdEncoding.DecodeString(b64Data)
	ct := ""
	size := int64(len(data))
	if meta != nil {
		ct = meta["content-type"]
	}
	return io.NopCloser(strings.NewReader(string(data))), storage.FileInfo{
		Key: key, Size: size, ContentType: ct, LastModified: time.Now(),
	}, nil
}

func (d *redisDriver) Head(ctx context.Context, key string) (storage.FileInfo, error) {
	metaKey := "bendy:meta:" + key
	resp, err := d.redisCmd([]interface{}{"GET", metaKey})
	if err != nil {
		return storage.FileInfo{}, err
	}
	if resp.StatusCode != 200 {
		return storage.FileInfo{}, fmt.Errorf("redis head error: %d", resp.StatusCode)
	}
	var result struct{ Result *string `json:"result"` }
	json.Unmarshal([]byte(resp.Body), &result)
	if result.Result == nil {
		return storage.FileInfo{}, fmt.Errorf("file not found: %s", key)
	}
	var meta map[string]string
	json.Unmarshal([]byte(*result.Result), &meta)
	size, _ := strconv.ParseInt(meta["size"], 10, 64)
	return storage.FileInfo{
		Key: key, Size: size, ContentType: meta["content-type"], LastModified: time.Now(),
	}, nil
}

func (d *redisDriver) Delete(ctx context.Context, key string) error {
	storageKey := "bendy:file:" + key
	metaKey := "bendy:meta:" + key
	d.redisCmd([]interface{}{"DEL", storageKey})
	d.redisCmd([]interface{}{"DEL", metaKey})
	return nil
}

func (d *redisDriver) List(ctx context.Context, prefix string, limit int, ct string) ([]storage.FileInfo, string, error) {
	pattern := "bendy:meta:" + prefix + "*"
	if prefix == "" {
		pattern = "bendy:meta:*"
	}
	resp, err := d.redisCmd([]interface{}{"KEYS", pattern})
	if err != nil {
		return nil, "", err
	}
	var result struct{ Result []string `json:"result"` }
	json.Unmarshal([]byte(resp.Body), &result)

	var infos []storage.FileInfo
	for _, metaKey := range result.Result {
		if limit > 0 && len(infos) >= limit {
			break
		}
		metaResp, _ := d.redisCmd([]interface{}{"GET", metaKey})
		if metaResp != nil {
			var mr struct{ Result *string `json:"result"` }
			json.Unmarshal([]byte(metaResp.Body), &mr)
			if mr.Result != nil {
				var meta map[string]string
				json.Unmarshal([]byte(*mr.Result), &meta)
				size, _ := strconv.ParseInt(meta["size"], 10, 64)
				key := strings.TrimPrefix(metaKey, "bendy:meta:")
				infos = append(infos, storage.FileInfo{
					Key: key, Size: size, ContentType: meta["content-type"],
				})
			}
		}
	}
	return infos, "", nil
}

func (d *redisDriver) Ping(ctx context.Context) error {
	resp, err := d.redisCmd([]interface{}{"PING"})
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("redis ping error: %d", resp.StatusCode)
	}
	return nil
}
