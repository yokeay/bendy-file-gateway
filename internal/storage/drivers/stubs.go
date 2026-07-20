package drivers

import (
	"context"
	"fmt"
	"io"

	"github.com/bendy/file-gateway/internal/storage"
)

type stubDriver struct {
	name   string
	config map[string]string
}

func (d *stubDriver) Name() string { return d.name }

func (d *stubDriver) Put(ctx context.Context, key string, body io.Reader, opts storage.UploadOptions) (storage.FileInfo, error) {
	return storage.FileInfo{}, fmt.Errorf("%s: not implemented", d.name)
}

func (d *stubDriver) Get(ctx context.Context, key string, opts storage.DownloadOptions) (io.ReadCloser, storage.FileInfo, error) {
	return nil, storage.FileInfo{}, fmt.Errorf("%s: not implemented", d.name)
}

func (d *stubDriver) Head(ctx context.Context, key string) (storage.FileInfo, error) {
	return storage.FileInfo{}, fmt.Errorf("%s: not implemented", d.name)
}

func (d *stubDriver) Delete(ctx context.Context, key string) error {
	return fmt.Errorf("%s: not implemented", d.name)
}

func (d *stubDriver) List(ctx context.Context, prefix string, limit int, ct string) ([]storage.FileInfo, string, error) {
	return nil, "", fmt.Errorf("%s: not implemented", d.name)
}

func (d *stubDriver) Ping(ctx context.Context) error {
	return fmt.Errorf("%s: not implemented", d.name)
}

func stubFactory(name string, required []string) {
	storage.Register(name, func(cfg map[string]string) (storage.Driver, error) {
		for _, k := range required {
			if cfg[k] == "" {
				return nil, fmt.Errorf("%s driver: missing required config key %q", name, k)
			}
		}
		return &stubDriver{name: name, config: cfg}, nil
	})
}

func init() {
	stubFactory("redis", []string{"addr", "db"})
	stubFactory("postgres", []string{"host", "port", "database", "username", "password"})
	stubFactory("mysql", []string{"host", "port", "database", "username", "password"})
	stubFactory("aliyun_oss", []string{"endpoint", "bucket", "access_key_id", "access_key_secret"})
	stubFactory("huawei_obs", []string{"endpoint", "bucket", "access_key_id", "access_key_secret"})
	stubFactory("qiniu_kodo", []string{"access_key", "secret_key", "bucket", "domain"})
	stubFactory("tencent_cos", []string{"region", "bucket", "secret_id", "secret_key"})
	stubFactory("tianyi_oos", []string{"endpoint", "bucket", "access_key", "secret_key"})
	stubFactory("unicom_oss", []string{"endpoint", "bucket", "access_key", "secret_key"})
}
