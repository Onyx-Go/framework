package storage

import (
	"context"
	"io"
	"os"
	"time"
)

// Driver defines the interface for storage drivers
type Driver interface {
	// Basic file operations
	Put(ctx context.Context, path string, contents []byte) error
	PutFile(ctx context.Context, path string, file io.Reader) error
	PutFileWithMetadata(ctx context.Context, path string, file io.Reader, metadata map[string]string) error
	Get(ctx context.Context, path string) ([]byte, error)
	GetStream(ctx context.Context, path string) (io.ReadCloser, error)
	Exists(ctx context.Context, path string) (bool, error)
	Delete(ctx context.Context, path string) error
	
	// File operations
	Copy(ctx context.Context, from, to string) error
	Move(ctx context.Context, from, to string) error
	Size(ctx context.Context, path string) (int64, error)
	LastModified(ctx context.Context, path string) (time.Time, error)
	MimeType(ctx context.Context, path string) (string, error)
	
	// Directory operations
	Files(ctx context.Context, directory string) ([]FileInfo, error)
	AllFiles(ctx context.Context, directory string) ([]FileInfo, error)
	Directories(ctx context.Context, directory string) ([]string, error)
	AllDirectories(ctx context.Context, directory string) ([]string, error)
	MakeDirectory(ctx context.Context, path string) error
	DeleteDirectory(ctx context.Context, path string) error
	
	// URL generation
	URL(path string) (string, error)
	TemporaryURL(path string, expiration time.Duration) (string, error)
	SignedURL(path string, expiration time.Duration, permissions map[string]interface{}) (string, error)
	
	// Advanced operations
	SetVisibility(ctx context.Context, path string, visibility Visibility) error
	GetVisibility(ctx context.Context, path string) (Visibility, error)
	
	// Batch operations
	PutMany(ctx context.Context, files map[string][]byte) error
	DeleteMany(ctx context.Context, paths []string) error
	
	// Metadata operations
	GetMetadata(ctx context.Context, path string) (map[string]string, error)
	SetMetadata(ctx context.Context, path string, metadata map[string]string) error
	
	// Driver management
	GetName() string
	GetConfig() Config
	HealthCheck(ctx context.Context) error
	Close(ctx context.Context) error
}

// Manager handles multiple storage drivers and provides a unified interface
type Manager interface {
	// Driver management
	RegisterDriver(name string, driver Driver) error
	GetDriver(name string) (Driver, error)
	SetDefaultDriver(name string)
	GetDefaultDriver() string
	ListDrivers() []string
	
	// Disk operations (uses default or specified driver)
	Disk(name ...string) Driver
	
	// Global configuration
	GetConfig() *ManagerConfig
	UpdateConfig(config *ManagerConfig) error
	
	// Health and monitoring
	HealthCheck(ctx context.Context, driverName ...string) error
	GetStats(ctx context.Context) (*Stats, error)
	
	// Lifecycle
	Close(ctx context.Context) error
}

// FileManager provides high-level file operations with additional features
type FileManager interface {
	// File operations with context
	Store(ctx context.Context, path string, content io.Reader, options ...StoreOption) (*FileInfo, error)
	Retrieve(ctx context.Context, path string) (*File, error)
	Remove(ctx context.Context, path string) error
	
	// Advanced file operations
	Archive(ctx context.Context, paths []string, archivePath string, format ArchiveFormat) error
	Extract(ctx context.Context, archivePath, destination string) error
	
	// File processing
	Transform(ctx context.Context, path string, transformer Transformer) (*FileInfo, error)
	Compress(ctx context.Context, path string, algorithm CompressionAlgorithm) (*FileInfo, error)
	
	// Search and indexing
	Search(ctx context.Context, query SearchQuery) ([]FileInfo, error)
	Index(ctx context.Context, path string) error
	
	// Synchronization
	Sync(ctx context.Context, source, destination Driver, options SyncOptions) error
}

// Uploader handles file uploads with validation and processing
type Uploader interface {
	// Upload operations
	Upload(ctx context.Context, file UploadedFile, destination string, options ...UploadOption) (*FileInfo, error)
	UploadMultiple(ctx context.Context, files []UploadedFile, destination string, options ...UploadOption) ([]FileInfo, error)
	
	// Validation
	Validate(file UploadedFile) error
	ValidateMultiple(files []UploadedFile) error
	
	// Configuration
	SetMaxFileSize(size int64)
	SetAllowedTypes(types []string)
	SetAllowedExtensions(extensions []string)
	AddValidator(validator UploadValidator)
	
	// Processing hooks
	OnBeforeUpload(hook func(ctx context.Context, file UploadedFile) error)
	OnAfterUpload(hook func(ctx context.Context, file UploadedFile, info *FileInfo) error)
}

// Transformer interface for file transformations
type Transformer interface {
	// Transform applies transformation to a file
	Transform(ctx context.Context, input io.Reader, output io.Writer, options map[string]interface{}) error
	
	// GetSupportedFormats returns supported input/output formats
	GetSupportedFormats() ([]string, []string)
	
	// GetName returns the transformer name
	GetName() string
}

// UploadValidator validates uploaded files
type UploadValidator interface {
	// Validate checks if the uploaded file is valid
	Validate(ctx context.Context, file UploadedFile) error
	
	// GetName returns the validator name
	GetName() string
}

// Config holds storage driver configuration
type Config struct {
	// Driver type (local, s3, gcs, azure, etc.)
	Type string `json:"type"`
	
	// Connection settings
	Endpoint   string `json:"endpoint,omitempty"`
	Region     string `json:"region,omitempty"`
	Bucket     string `json:"bucket,omitempty"`
	RootPath   string `json:"root_path,omitempty"`
	
	// Authentication
	AccessKey    string `json:"access_key,omitempty"`
	SecretKey    string `json:"secret_key,omitempty"`
	Token        string `json:"token,omitempty"`
	Credentials  string `json:"credentials,omitempty"`
	
	// Local storage settings
	LocalPath   string      `json:"local_path,omitempty"`
	Permissions Permissions `json:"permissions,omitempty"`
	
	// URL settings
	BaseURL        string `json:"base_url,omitempty"`
	CDNUrl         string `json:"cdn_url,omitempty"`
	UsePathStyle   bool   `json:"use_path_style,omitempty"`
	
	// Performance settings
	Timeout         time.Duration `json:"timeout,omitempty"`
	RetryAttempts   int           `json:"retry_attempts,omitempty"`
	ChunkSize       int64         `json:"chunk_size,omitempty"`
	ConcurrentOps   int           `json:"concurrent_ops,omitempty"`
	
	// Feature flags
	UseSSL          bool `json:"use_ssl,omitempty"`
	VerifySSL       bool `json:"verify_ssl,omitempty"`
	UsePresignedURL bool `json:"use_presigned_url,omitempty"`
	
	// Additional options
	Options map[string]interface{} `json:"options,omitempty"`
}

// ManagerConfig holds storage manager configuration
type ManagerConfig struct {
	// Default driver name
	DefaultDriver string `json:"default_driver"`
	
	// Driver configurations
	Drivers map[string]Config `json:"drivers"`
	
	// Global settings
	TempDirectory string        `json:"temp_directory"`
	CacheEnabled  bool          `json:"cache_enabled"`
	CacheTTL      time.Duration `json:"cache_ttl"`
	
	// Upload settings
	MaxFileSize       int64    `json:"max_file_size"`
	AllowedTypes      []string `json:"allowed_types"`
	AllowedExtensions []string `json:"allowed_extensions"`
	
	// Security settings
	ScanUploads      bool     `json:"scan_uploads"`
	QuarantinePath   string   `json:"quarantine_path"`
	BlockedExtensions []string `json:"blocked_extensions"`
	
	// Performance settings
	EnableCompression bool  `json:"enable_compression"`
	CompressionLevel  int   `json:"compression_level"`
	EnableDeduplication bool `json:"enable_deduplication"`
}

// FileInfo contains information about a file
type FileInfo struct {
	Path         string            `json:"path"`
	Name         string            `json:"name"`
	Size         int64             `json:"size"`
	MimeType     string            `json:"mime_type"`
	Extension    string            `json:"extension"`
	ModTime      time.Time         `json:"mod_time"`
	IsDir        bool              `json:"is_dir"`
	Visibility   Visibility        `json:"visibility"`
	Checksum     string            `json:"checksum,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	URL          string            `json:"url,omitempty"`
	TemporaryURL string            `json:"temporary_url,omitempty"`
}

// File represents a file with its content
type File struct {
	Info    FileInfo          `json:"info"`
	Content io.ReadCloser     `json:"-"`
	Reader  func() (io.ReadCloser, error) `json:"-"`
}

// UploadedFile represents an uploaded file
type UploadedFile struct {
	FieldName    string                 `json:"field_name"`
	OriginalName string                 `json:"original_name"`
	Size         int64                  `json:"size"`
	MimeType     string                 `json:"mime_type"`
	Extension    string                 `json:"extension"`
	TempPath     string                 `json:"temp_path"`
	Content      io.Reader              `json:"-"`
	Headers      map[string][]string    `json:"headers,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Stats holds storage statistics
type Stats struct {
	// Driver statistics
	DriverStats map[string]DriverStats `json:"driver_stats"`
	
	// Global statistics
	TotalFiles      int64 `json:"total_files"`
	TotalSize       int64 `json:"total_size"`
	TotalDirectories int64 `json:"total_directories"`
	
	// Operation statistics
	OperationsCount map[string]int64  `json:"operations_count"`
	AverageLatency  map[string]time.Duration `json:"average_latency"`
	ErrorRate       map[string]float64 `json:"error_rate"`
	
	// Collection info
	CollectedAt time.Time     `json:"collected_at"`
	Period      time.Duration `json:"period"`
}

// DriverStats holds statistics for a specific driver
type DriverStats struct {
	Name            string        `json:"name"`
	Type            string        `json:"type"`
	FilesCount      int64         `json:"files_count"`
	TotalSize       int64         `json:"total_size"`
	LastOperation   time.Time     `json:"last_operation"`
	OperationsCount int64         `json:"operations_count"`
	ErrorsCount     int64         `json:"errors_count"`
	AverageLatency  time.Duration `json:"average_latency"`
	HealthStatus    string        `json:"health_status"`
}

// Permissions for file operations
type Permissions struct {
	FileMode os.FileMode `json:"file_mode"`
	DirMode  os.FileMode `json:"dir_mode"`
	Owner    string      `json:"owner,omitempty"`
	Group    string      `json:"group,omitempty"`
}

// Visibility levels
type Visibility string

const (
	VisibilityPrivate Visibility = "private"
	VisibilityPublic  Visibility = "public"
)

// Archive formats
type ArchiveFormat string

const (
	ArchiveFormatZip    ArchiveFormat = "zip"
	ArchiveFormatTarGz  ArchiveFormat = "tar.gz"
	ArchiveFormatTar    ArchiveFormat = "tar"
)

// Compression algorithms
type CompressionAlgorithm string

const (
	CompressionGzip   CompressionAlgorithm = "gzip"
	CompressionBzip2  CompressionAlgorithm = "bzip2"
	CompressionLzma   CompressionAlgorithm = "lzma"
	CompressionSnappy CompressionAlgorithm = "snappy"
)

// Store options
type StoreOption func(*StoreOptions)

type StoreOptions struct {
	Visibility   Visibility
	Metadata     map[string]string
	ContentType  string
	CacheControl string
	Overwrite    bool
}

// Upload options
type UploadOption func(*UploadOptions)

type UploadOptions struct {
	StoreOptions
	ProcessImage    bool
	GenerateThumbnail bool
	ValidateContent bool
	ScanVirus       bool
}

// Search query
type SearchQuery struct {
	Path       string            `json:"path,omitempty"`
	Name       string            `json:"name,omitempty"`
	Extension  string            `json:"extension,omitempty"`
	MimeType   string            `json:"mime_type,omitempty"`
	MinSize    int64             `json:"min_size,omitempty"`
	MaxSize    int64             `json:"max_size,omitempty"`
	ModifiedAfter  time.Time     `json:"modified_after,omitempty"`
	ModifiedBefore time.Time     `json:"modified_before,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Limit      int               `json:"limit,omitempty"`
	Offset     int               `json:"offset,omitempty"`
}

// Sync options
type SyncOptions struct {
	DeleteExtra   bool
	PreserveTime  bool
	DryRun        bool
	ExcludePatterns []string
	IncludePatterns []string
	ChunkSize     int64
	Concurrent    int
}

// Default configurations
func DefaultConfig() Config {
	return Config{
		Type:            "local",
		LocalPath:       "storage/app",
		Permissions:     DefaultPermissions(),
		Timeout:         30 * time.Second,
		RetryAttempts:   3,
		ChunkSize:       1024 * 1024, // 1MB
		ConcurrentOps:   5,
		UseSSL:          true,
		VerifySSL:       true,
		Options:         make(map[string]interface{}),
	}
}

func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		DefaultDriver: "local",
		Drivers: map[string]Config{
			"local": DefaultConfig(),
			"public": {
				Type:        "local",
				LocalPath:   "storage/app/public",
				Permissions: DefaultPermissions(),
				BaseURL:     "/storage",
			},
		},
		TempDirectory:     "storage/tmp",
		CacheEnabled:      true,
		CacheTTL:          1 * time.Hour,
		MaxFileSize:       10 << 20, // 10MB
		AllowedTypes:      []string{"image/jpeg", "image/png", "image/gif", "application/pdf", "text/plain"},
		AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".txt"},
		ScanUploads:       true,
		BlockedExtensions: []string{".exe", ".bat", ".cmd", ".scr"},
		EnableCompression: true,
		CompressionLevel:  6,
	}
}

func DefaultPermissions() Permissions {
	return Permissions{
		FileMode: 0644,
		DirMode:  0755,
	}
}

// Helper functions for options
func WithVisibility(visibility Visibility) StoreOption {
	return func(opts *StoreOptions) {
		opts.Visibility = visibility
	}
}

func WithMetadata(metadata map[string]string) StoreOption {
	return func(opts *StoreOptions) {
		opts.Metadata = metadata
	}
}

func WithContentType(contentType string) StoreOption {
	return func(opts *StoreOptions) {
		opts.ContentType = contentType
	}
}

func WithOverwrite(overwrite bool) StoreOption {
	return func(opts *StoreOptions) {
		opts.Overwrite = overwrite
	}
}

func WithImageProcessing(process bool) UploadOption {
	return func(opts *UploadOptions) {
		opts.ProcessImage = process
	}
}

func WithThumbnailGeneration(generate bool) UploadOption {
	return func(opts *UploadOptions) {
		opts.GenerateThumbnail = generate
	}
}

func WithContentValidation(validate bool) UploadOption {
	return func(opts *UploadOptions) {
		opts.ValidateContent = validate
	}
}