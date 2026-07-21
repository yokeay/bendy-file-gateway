package drivers

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/bendy/file-gateway/internal/storage"
	"github.com/bendy/file-gateway/internal/wasm"
)

// PostgreSQL storage driver, stores files as binary blobs in a table.
type postgresDriver struct {
	config map[string]string
}

func init() {
	storage.Register("postgres", func(cfg map[string]string) (storage.Driver, error) {
		required := []string{"host", "port", "database", "username", "password"}
		for _, k := range required {
			if cfg[k] == "" {
				return nil, fmt.Errorf("postgres driver: missing required config key %q", k)
			}
		}
		d := &postgresDriver{config: cfg}
		if err := d.ensureTable(); err != nil {
			return nil, err
		}
		return d, nil
	})
}

func (d *postgresDriver) Name() string { return "postgres" }

func (d *postgresDriver) ensureTable() error {
	_, err := wasm.DBExec(
		`CREATE TABLE IF NOT EXISTS bendy_storage (
			key TEXT PRIMARY KEY,
			data BYTEA NOT NULL,
			content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
			size BIGINT NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		[]interface{}{},
	)
	return err
}

func (d *postgresDriver) Put(ctx context.Context, key string, body io.Reader, opts storage.UploadOptions) (storage.FileInfo, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return storage.FileInfo{}, err
	}
	ct := opts.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	b64Data := base64.StdEncoding.EncodeToString(bodyBytes)

	_, err = wasm.DBExec(
		`INSERT INTO bendy_storage (key, data, content_type, size, created_at, updated_at)
		 VALUES (?, decode(?, 'base64'), ?, ?, ?, ?)
		 ON CONFLICT (key) DO UPDATE SET data = EXCLUDED.data, size = EXCLUDED.size, content_type = EXCLUDED.content_type, updated_at = EXCLUDED.updated_at`,
		[]interface{}{key, b64Data, ct, int64(len(bodyBytes)), now, now},
	)
	if err != nil {
		return storage.FileInfo{}, err
	}
	return storage.FileInfo{
		Key: key, Size: int64(len(bodyBytes)), ContentType: ct,
		LastModified: time.Now(),
	}, nil
}

func (d *postgresDriver) Get(ctx context.Context, key string, opts storage.DownloadOptions) (io.ReadCloser, storage.FileInfo, error) {
	rows, err := wasm.DBQuery(
		"SELECT key, data, content_type, size FROM bendy_storage WHERE key = ?",
		[]interface{}{key},
	)
	if err != nil || len(rows) == 0 {
		return nil, storage.FileInfo{}, fmt.Errorf("file not found: %s", key)
	}
	row := rows[0]
	data := []byte(asStr(row["data"]))
	size := asInt(row["size"])
	ct := asStr(row["content_type"])

	if opts.RangeStart > 0 || opts.RangeEnd > 0 {
		start := opts.RangeStart
		end := int64(len(data))
		if opts.RangeEnd > 0 && opts.RangeEnd < end {
			end = opts.RangeEnd
		}
		data = data[start:end]
	}

	return io.NopCloser(&byteReader{data: data}), storage.FileInfo{
		Key: key, Size: size, ContentType: ct, LastModified: time.Now(),
	}, nil
}

func (d *postgresDriver) Head(ctx context.Context, key string) (storage.FileInfo, error) {
	rows, err := wasm.DBQuery(
		"SELECT key, content_type, size FROM bendy_storage WHERE key = ?",
		[]interface{}{key},
	)
	if err != nil || len(rows) == 0 {
		return storage.FileInfo{}, fmt.Errorf("file not found: %s", key)
	}
	return storage.FileInfo{
		Key: key, Size: asInt(rows[0]["size"]), ContentType: asStr(rows[0]["content_type"]),
		LastModified: time.Now(),
	}, nil
}

func (d *postgresDriver) Delete(ctx context.Context, key string) error {
	_, err := wasm.DBExec("DELETE FROM bendy_storage WHERE key = ?", []interface{}{key})
	return err
}

func (d *postgresDriver) List(ctx context.Context, prefix string, limit int, ct string) ([]storage.FileInfo, string, error) {
	var rows []map[string]interface{}
	var err error
	if prefix != "" {
		rows, err = wasm.DBQuery(
			"SELECT key, content_type, size FROM bendy_storage WHERE key LIKE ? ORDER BY key LIMIT ?",
			[]interface{}{prefix + "%", limit},
		)
	} else {
		rows, err = wasm.DBQuery(
			"SELECT key, content_type, size FROM bendy_storage ORDER BY key LIMIT ?",
			[]interface{}{limit},
		)
	}
	if err != nil {
		return nil, "", err
	}
	infos := make([]storage.FileInfo, len(rows))
	for i, row := range rows {
		infos[i] = storage.FileInfo{
			Key: asStr(row["key"]), Size: asInt(row["size"]),
			ContentType: asStr(row["content_type"]),
		}
	}
	var nextToken string
	if len(infos) == limit && limit > 0 {
		nextToken = infos[len(infos)-1].Key
	}
	return infos, nextToken, nil
}

func (d *postgresDriver) Ping(ctx context.Context) error {
	_, err := wasm.DBQuery("SELECT 1", []interface{}{})
	return err
}

type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *byteReader) Close() error { return nil }

func asStr(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func asInt(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case string:
		n, _ := strconv.ParseInt(val, 10, 64)
		return n
	case []byte:
		n, _ := strconv.ParseInt(string(val), 10, 64)
		return n
	default:
		return 0
	}
}
