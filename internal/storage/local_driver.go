package storage

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalDriver implements the Driver interface for local filesystem storage
type LocalDriver struct {
	name        string
	config      Config
	rootPath    string
	permissions Permissions
	baseURL     string
}

// NewLocalDriver creates a new local storage driver
func NewLocalDriver(name string, config Config) (*LocalDriver, error) {
	rootPath := config.LocalPath
	if rootPath == "" {
		rootPath = "storage/app"
	}
	
	// Ensure root directory exists
	if err := os.MkdirAll(rootPath, config.Permissions.DirMode); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	return &LocalDriver{
		name:        name,
		config:      config,
		rootPath:    rootPath,
		permissions: config.Permissions,
		baseURL:     config.BaseURL,
	}, nil
}

// Basic file operations

// Put stores content at the specified path
func (ld *LocalDriver) Put(ctx context.Context, path string, contents []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	fullPath := ld.fullPath(path)
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(fullPath), ld.permissions.DirMode); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Write file
	if err := os.WriteFile(fullPath, contents, ld.permissions.FileMode); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// PutFile stores content from a reader at the specified path
func (ld *LocalDriver) PutFile(ctx context.Context, path string, file io.Reader) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	fullPath := ld.fullPath(path)
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(fullPath), ld.permissions.DirMode); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Create and write file
	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, ld.permissions.FileMode)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()
	
	_, err = io.Copy(f, file)
	if err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}
	
	return nil
}

// PutFileWithMetadata stores content with metadata (metadata stored as extended attributes if supported)
func (ld *LocalDriver) PutFileWithMetadata(ctx context.Context, path string, file io.Reader, metadata map[string]string) error {
	if err := ld.PutFile(ctx, path, file); err != nil {
		return err
	}
	
	// For local storage, we can store metadata in a companion file
	if len(metadata) > 0 {
		return ld.SetMetadata(ctx, path, metadata)
	}
	
	return nil
}

// Get retrieves content from the specified path
func (ld *LocalDriver) Get(ctx context.Context, path string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	content, err := os.ReadFile(ld.fullPath(path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	return content, nil
}

// GetStream returns a reader for the file content
func (ld *LocalDriver) GetStream(ctx context.Context, path string) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	file, err := os.Open(ld.fullPath(path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	
	return file, nil
}

// Exists checks if a file exists at the specified path
func (ld *LocalDriver) Exists(ctx context.Context, path string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}
	
	_, err := os.Stat(ld.fullPath(path))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	
	return true, nil
}

// Delete removes the file at the specified path
func (ld *LocalDriver) Delete(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	fullPath := ld.fullPath(path)
	
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", path)
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}
	
	// Also remove metadata file if it exists
	metadataPath := ld.metadataPath(path)
	os.Remove(metadataPath) // Ignore error - metadata file may not exist
	
	return nil
}

// File operations

// Copy copies a file from source to destination
func (ld *LocalDriver) Copy(ctx context.Context, from, to string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	fromPath := ld.fullPath(from)
	toPath := ld.fullPath(to)
	
	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(toPath), ld.permissions.DirMode); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	
	// Open source file
	sourceFile, err := os.Open(fromPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()
	
	// Create destination file
	destFile, err := os.OpenFile(toPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, ld.permissions.FileMode)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()
	
	// Copy content
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}
	
	// Copy metadata if it exists
	if metadata, err := ld.GetMetadata(ctx, from); err == nil && len(metadata) > 0 {
		ld.SetMetadata(ctx, to, metadata)
	}
	
	return nil
}

// Move moves a file from source to destination
func (ld *LocalDriver) Move(ctx context.Context, from, to string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	fromPath := ld.fullPath(from)
	toPath := ld.fullPath(to)
	
	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(toPath), ld.permissions.DirMode); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	
	// Attempt atomic rename first
	if err := os.Rename(fromPath, toPath); err != nil {
		// If rename fails (e.g., cross-device), fall back to copy + delete
		if err := ld.Copy(ctx, from, to); err != nil {
			return err
		}
		return ld.Delete(ctx, from)
	}
	
	// Move metadata file if it exists
	fromMetadata := ld.metadataPath(from)
	toMetadata := ld.metadataPath(to)
	if _, err := os.Stat(fromMetadata); err == nil {
		os.Rename(fromMetadata, toMetadata)
	}
	
	return nil
}

// Size returns the size of the file
func (ld *LocalDriver) Size(ctx context.Context, path string) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	
	info, err := os.Stat(ld.fullPath(path))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("file not found: %s", path)
		}
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}
	
	return info.Size(), nil
}

// LastModified returns the last modification time
func (ld *LocalDriver) LastModified(ctx context.Context, path string) (time.Time, error) {
	select {
	case <-ctx.Done():
		return time.Time{}, ctx.Err()
	default:
	}
	
	info, err := os.Stat(ld.fullPath(path))
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, fmt.Errorf("file not found: %s", path)
		}
		return time.Time{}, fmt.Errorf("failed to get file info: %w", err)
	}
	
	return info.ModTime(), nil
}

// MimeType returns the MIME type of the file
func (ld *LocalDriver) MimeType(ctx context.Context, path string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	
	// Get MIME type from file extension
	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		// Try to detect from file content
		file, err := os.Open(ld.fullPath(path))
		if err != nil {
			return "", fmt.Errorf("failed to open file for MIME detection: %w", err)
		}
		defer file.Close()
		
		// Read first 512 bytes for content detection
		buffer := make([]byte, 512)
		n, _ := file.Read(buffer)
		mimeType = http.DetectContentType(buffer[:n])
	}
	
	return mimeType, nil
}

// Directory operations

// Files returns files in the specified directory
func (ld *LocalDriver) Files(ctx context.Context, directory string) ([]FileInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	fullPath := ld.fullPath(directory)
	
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	
	var files []FileInfo
	for _, entry := range entries {
		if !entry.IsDir() && !ld.isMetadataFile(entry.Name()) {
			info := ld.createFileInfo(filepath.Join(directory, entry.Name()), entry)
			files = append(files, info)
		}
	}
	
	return files, nil
}

// AllFiles returns all files recursively in the specified directory
func (ld *LocalDriver) AllFiles(ctx context.Context, directory string) ([]FileInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	var files []FileInfo
	
	err := filepath.WalkDir(ld.fullPath(directory), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if !d.IsDir() && !ld.isMetadataFile(d.Name()) {
			relPath, _ := filepath.Rel(ld.rootPath, path)
			info := ld.createFileInfo(relPath, d)
			files = append(files, info)
		}
		
		return nil
	})
	
	return files, err
}

// Directories returns directories in the specified directory
func (ld *LocalDriver) Directories(ctx context.Context, directory string) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	fullPath := ld.fullPath(directory)
	
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(directory, entry.Name()))
		}
	}
	
	return dirs, nil
}

// AllDirectories returns all directories recursively
func (ld *LocalDriver) AllDirectories(ctx context.Context, directory string) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	var dirs []string
	
	err := filepath.WalkDir(ld.fullPath(directory), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() && path != ld.fullPath(directory) {
			relPath, _ := filepath.Rel(ld.rootPath, path)
			dirs = append(dirs, relPath)
		}
		
		return nil
	})
	
	return dirs, err
}

// MakeDirectory creates a directory
func (ld *LocalDriver) MakeDirectory(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	return os.MkdirAll(ld.fullPath(path), ld.permissions.DirMode)
}

// DeleteDirectory removes a directory and all its contents
func (ld *LocalDriver) DeleteDirectory(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	return os.RemoveAll(ld.fullPath(path))
}

// URL generation

// URL returns the public URL for the file
func (ld *LocalDriver) URL(path string) (string, error) {
	if ld.baseURL == "" {
		return "", fmt.Errorf("no base URL configured for local driver")
	}
	
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(ld.baseURL, "/"), strings.TrimPrefix(path, "/")), nil
}

// TemporaryURL returns a temporary URL (same as URL for local storage)
func (ld *LocalDriver) TemporaryURL(path string, expiration time.Duration) (string, error) {
	return ld.URL(path)
}

// SignedURL returns a signed URL (same as URL for local storage)
func (ld *LocalDriver) SignedURL(path string, expiration time.Duration, permissions map[string]interface{}) (string, error) {
	return ld.URL(path)
}

// Advanced operations

// SetVisibility sets file visibility (for local storage, this affects file permissions)
func (ld *LocalDriver) SetVisibility(ctx context.Context, path string, visibility Visibility) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	fullPath := ld.fullPath(path)
	
	var mode os.FileMode
	if visibility == VisibilityPublic {
		mode = 0644
	} else {
		mode = 0600
	}
	
	return os.Chmod(fullPath, mode)
}

// GetVisibility returns file visibility
func (ld *LocalDriver) GetVisibility(ctx context.Context, path string) (Visibility, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	
	info, err := os.Stat(ld.fullPath(path))
	if err != nil {
		return "", err
	}
	
	mode := info.Mode()
	if mode&0044 != 0 { // Others can read
		return VisibilityPublic, nil
	}
	
	return VisibilityPrivate, nil
}

// Batch operations

// PutMany stores multiple files
func (ld *LocalDriver) PutMany(ctx context.Context, files map[string][]byte) error {
	for path, content := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		if err := ld.Put(ctx, path, content); err != nil {
			return fmt.Errorf("failed to put file %s: %w", path, err)
		}
	}
	
	return nil
}

// DeleteMany deletes multiple files
func (ld *LocalDriver) DeleteMany(ctx context.Context, paths []string) error {
	for _, path := range paths {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		if err := ld.Delete(ctx, path); err != nil {
			return fmt.Errorf("failed to delete file %s: %w", path, err)
		}
	}
	
	return nil
}

// Metadata operations

// GetMetadata retrieves file metadata
func (ld *LocalDriver) GetMetadata(ctx context.Context, path string) (map[string]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	metadataPath := ld.metadataPath(path)
	
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return make(map[string]string), nil
	}
	
	content, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}
	
	metadata := make(map[string]string)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			metadata[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	
	return metadata, nil
}

// SetMetadata sets file metadata
func (ld *LocalDriver) SetMetadata(ctx context.Context, path string, metadata map[string]string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	if len(metadata) == 0 {
		// Remove metadata file if no metadata
		return os.Remove(ld.metadataPath(path))
	}
	
	metadataPath := ld.metadataPath(path)
	
	// Create metadata directory
	if err := os.MkdirAll(filepath.Dir(metadataPath), ld.permissions.DirMode); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}
	
	// Write metadata as key=value pairs
	var content strings.Builder
	for key, value := range metadata {
		content.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}
	
	return os.WriteFile(metadataPath, []byte(content.String()), ld.permissions.FileMode)
}

// Driver management

// GetName returns the driver name
func (ld *LocalDriver) GetName() string {
	return ld.name
}

// GetConfig returns the driver configuration
func (ld *LocalDriver) GetConfig() Config {
	return ld.config
}

// HealthCheck checks if the driver is healthy
func (ld *LocalDriver) HealthCheck(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	// Check if root directory is accessible
	_, err := os.Stat(ld.rootPath)
	if err != nil {
		return fmt.Errorf("root directory not accessible: %w", err)
	}
	
	// Try to create a test file
	testPath := filepath.Join(ld.rootPath, ".health_check")
	if err := os.WriteFile(testPath, []byte("test"), 0644); err != nil {
		return fmt.Errorf("cannot write to storage directory: %w", err)
	}
	
	// Clean up test file
	os.Remove(testPath)
	
	return nil
}

// Close closes the driver and releases resources
func (ld *LocalDriver) Close(ctx context.Context) error {
	// Local driver doesn't need explicit closing
	return nil
}

// Helper methods

// fullPath returns the full filesystem path
func (ld *LocalDriver) fullPath(path string) string {
	return filepath.Join(ld.rootPath, filepath.Clean(path))
}

// metadataPath returns the path for metadata file
func (ld *LocalDriver) metadataPath(path string) string {
	return ld.fullPath(path) + ".meta"
}

// isMetadataFile checks if a file is a metadata file
func (ld *LocalDriver) isMetadataFile(name string) bool {
	return strings.HasSuffix(name, ".meta")
}

// createFileInfo creates a FileInfo from directory entry
func (ld *LocalDriver) createFileInfo(path string, entry os.DirEntry) FileInfo {
	info, _ := entry.Info()
	
	// Calculate checksum for small files
	var checksum string
	if info.Size() < 1024*1024 { // 1MB
		if content, err := os.ReadFile(ld.fullPath(path)); err == nil {
			checksum = fmt.Sprintf("%x", md5.Sum(content))
		}
	}
	
	// Get metadata
	metadata, _ := ld.GetMetadata(context.Background(), path)
	
	// Get URL
	url, _ := ld.URL(path)
	
	return FileInfo{
		Path:      path,
		Name:      entry.Name(),
		Size:      info.Size(),
		MimeType:  mime.TypeByExtension(filepath.Ext(path)),
		Extension: filepath.Ext(path),
		ModTime:   info.ModTime(),
		IsDir:     entry.IsDir(),
		Visibility: VisibilityPrivate, // Default for local storage
		Checksum:  checksum,
		Metadata:  metadata,
		URL:       url,
	}
}