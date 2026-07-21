package drivers

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/bendy/file-gateway/internal/storage"
	"github.com/bendy/file-gateway/internal/wasm"
)

// MySQL storage driver, stores files as binary blobs in a table.
type mysqlDriver struct {
	config map[string]string
}

func init() {
	storage.Register("mysql", func(cfg map[string]string) (storage.Driver, error) {
		required := []string{"host", "port", "database", "username", "password"}
		for _, k := range required {
			if cfg[k] == "" {
				return nil, fmt.Errorf("mysql driver: missing required config key %q", k)
			}
		}
		d := &mysqlDriver{config: cfg}
		if err := d.ensureTable(); err != nil {
			return nil, err
		}
		return d, nil
	})
}

func (d *mysqlDriver) Name() string { return "mysql" }

func (d *mysqlDriver) ensureTable() error {
	_, err := wasm.DBExec(
		`CREATE TABLE IF NOT EXISTS bendy_storage (
			bendy_key VARCHAR(1024) PRIMARY KEY,
			data LONGBLOB NOT NULL,
			content_type VARCHAR(255) NOT NULL DEFAULT 'application/octet-stream',
			size BIGINT NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		[]interface{}{},
	)
	return err
}

func (d *mysqlDriver) Put(ctx context.Context, key string, body io.Reader, opts storage.UploadOptions) (storage.FileInfo, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return storage.FileInfo{}, err
	}
	ct := opts.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	now := time.Now().UTC().Format(time.RFC3339)

	_, err = wasm.DBExec(
		`INSERT INTO bendy_storage (bendy_key, data, content_type, size, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE data = VALUES(data), size = VALUES(size), content_type = VALUES(content_type), updated_at = VALUES(updated_at)`,
		[]interface{}{key, string(bodyBytes), ct, int64(len(bodyBytes)), now, now},
	)
	if err != nil {
		return storage.FileInfo{}, err
	}
	return storage.FileInfo{
		Key: key, Size: int64(len(bodyBytes)), ContentType: ct,
		LastModified: time.Now(),
	}, nil
}

func (d *mysqlDriver) Get(ctx context.Context, key string, opts storage.DownloadOptions) (io.ReadCloser, storage.FileInfo, error) {
	rows, err := wasm.DBQuery(
		"SELECT bendy_key, data, content_type, size FROM bendy_storage WHERE bendy_key = ?",
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

func (d *mysqlDriver) Head(ctx context.Context, key string) (storage.FileInfo, error) {
	rows, err := wasm.DBQuery(
		"SELECT bendy_key, content_type, size FROM bendy_storage WHERE bendy_key = ?",
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

func (d *mysqlDriver) Delete(ctx context.Context, key string) error {
	_, err := wasm.DBExec("DELETE FROM bendy_storage WHERE bendy_key = ?", []interface{}{key})
	return err
}

func (d *mysqlDriver) List(ctx context.Context, prefix string, limit int, ct string) ([]storage.FileInfo, string, error) {
	var rows []map[string]interface{}
	var err error
	if prefix != "" {
		rows, err = wasm.DBQuery(
			"SELECT bendy_key, content_type, size FROM bendy_storage WHERE bendy_key LIKE ? ORDER BY bendy_key LIMIT ?",
			[]interface{}{prefix + "%", limit},
		)
	} else {
		rows, err = wasm.DBQuery(
			"SELECT bendy_key, content_type, size FROM bendy_storage ORDER BY bendy_key LIMIT ?",
			[]interface{}{limit},
		)
	}
	if err != nil {
		return nil, "", err
	}
	infos := make([]storage.FileInfo, len(rows))
	for i, row := range rows {
		infos[i] = storage.FileInfo{
			Key: asStr(row["bendy_key"]), Size: asInt(row["size"]),
			ContentType: asStr(row["content_type"]),
		}
	}
	var nextToken string
	if len(infos) == limit && limit > 0 {
		nextToken = infos[len(infos)-1].Key
	}
	return infos, nextToken, nil
}

func (d *mysqlDriver) Ping(ctx context.Context) error {
	_, err := wasm.DBQuery("SELECT 1", []interface{}{})
	return err
}
