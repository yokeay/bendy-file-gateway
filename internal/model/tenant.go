package model

import "time"

// Tenant represents a multi-tenant account.
type Tenant struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	AccessKey     string    `json:"access_key"`
	SecretKeyHash string    `json:"-"`
	Status        string    `json:"status"` // active, suspended, deleted
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TenantQuota tracks usage limits for a tenant.
type TenantQuota struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	TrafficLimit  int64      `json:"traffic_limit"`
	TrafficUsed   int64      `json:"traffic_used"`
	APICallsLimit int64      `json:"api_calls_limit"`
	APICallsUsed  int64      `json:"api_calls_used"`
	ExpiryAt      *time.Time `json:"expiry_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Backend represents a storage backend configuration for a tenant.
type Backend struct {
	ID        string            `json:"id"`
	TenantID  string            `json:"tenant_id"`
	Name      string            `json:"name"`
	Driver    string            `json:"driver"`
	Config    map[string]string `json:"config"`
	IsDefault bool              `json:"is_default"`
	Status    string            `json:"status"` // active, disabled
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Directory is a virtual directory node.
type Directory struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	ParentID  *string   `json:"parent_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// File represents a file stored in a backend.
type File struct {
	ID           string            `json:"id"`
	TenantID     string            `json:"tenant_id"`
	DirectoryID  *string           `json:"directory_id"`
	BackendID    string            `json:"backend_id"`
	VirtualName  string            `json:"virtual_name"`
	StorageKey   string            `json:"storage_key"`
	ContentType  string            `json:"content_type"`
	Size         int64             `json:"size"`
	Checksum     string            `json:"checksum"`
	Metadata     map[string]string `json:"metadata"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// Admin represents an admin user (GitHub-authenticated).
type Admin struct {
	ID             string     `json:"id"`
	GitHubUsername string     `json:"github_username"`
	GitHubID       int64      `json:"github_id"`
	Name           string     `json:"name"`
	AvatarURL      string     `json:"avatar_url"`
	Role           string     `json:"role"` // admin, superadmin
	LastLoginAt    *time.Time `json:"last_login_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// AdminSession is an active admin session.
type AdminSession struct {
	ID        string    `json:"id"`
	AdminID   string    `json:"admin_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// APILog records each API request for billing/auditing.
type APILog struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Method        string    `json:"method"`
	Path          string    `json:"path"`
	StatusCode    int       `json:"status_code"`
	BytesSent     int64     `json:"bytes_sent"`
	BytesReceived int64     `json:"bytes_received"`
	DurationMs    int64     `json:"duration_ms"`
	CreatedAt     time.Time `json:"created_at"`
}
