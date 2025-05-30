package onyx

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Storage interface {
	Put(path string, contents []byte) error
	PutFile(path string, file io.Reader) error
	Get(path string) ([]byte, error)
	Exists(path string) bool
	Delete(path string) error
	Copy(from, to string) error
	Move(from, to string) error
	Size(path string) (int64, error)
	LastModified(path string) (time.Time, error)
	Files(directory string) ([]string, error)
	AllFiles(directory string) ([]string, error)
	Directories(directory string) ([]string, error)
	AllDirectories(directory string) ([]string, error)
	MakeDirectory(path string) error
	DeleteDirectory(path string) error
	URL(path string) string
	TemporaryURL(path string, expiration time.Time) string
}

type LocalStorage struct {
	root string
}

func NewLocalStorage(root string) *LocalStorage {
	if err := os.MkdirAll(root, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create storage directory: %v", err))
	}
	
	return &LocalStorage{root: root}
}

func (ls *LocalStorage) Put(path string, contents []byte) error {
	fullPath := ls.fullPath(path)
	
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	
	return os.WriteFile(fullPath, contents, 0644)
}

func (ls *LocalStorage) PutFile(path string, file io.Reader) error {
	fullPath := ls.fullPath(path)
	
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	
	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()
	
	_, err = io.Copy(f, file)
	return err
}

func (ls *LocalStorage) Get(path string) ([]byte, error) {
	return os.ReadFile(ls.fullPath(path))
}

func (ls *LocalStorage) Exists(path string) bool {
	_, err := os.Stat(ls.fullPath(path))
	return err == nil
}

func (ls *LocalStorage) Delete(path string) error {
	return os.Remove(ls.fullPath(path))
}

func (ls *LocalStorage) Copy(from, to string) error {
	fromPath := ls.fullPath(from)
	toPath := ls.fullPath(to)
	
	if err := os.MkdirAll(filepath.Dir(toPath), 0755); err != nil {
		return err
	}
	
	sourceFile, err := os.Open(fromPath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(toPath)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (ls *LocalStorage) Move(from, to string) error {
	fromPath := ls.fullPath(from)
	toPath := ls.fullPath(to)
	
	if err := os.MkdirAll(filepath.Dir(toPath), 0755); err != nil {
		return err
	}
	
	return os.Rename(fromPath, toPath)
}

func (ls *LocalStorage) Size(path string) (int64, error) {
	info, err := os.Stat(ls.fullPath(path))
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (ls *LocalStorage) LastModified(path string) (time.Time, error) {
	info, err := os.Stat(ls.fullPath(path))
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func (ls *LocalStorage) Files(directory string) ([]string, error) {
	fullPath := ls.fullPath(directory)
	
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}
	
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(directory, entry.Name()))
		}
	}
	
	return files, nil
}

func (ls *LocalStorage) AllFiles(directory string) ([]string, error) {
	var files []string
	
	err := filepath.WalkDir(ls.fullPath(directory), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if !d.IsDir() {
			relPath, _ := filepath.Rel(ls.root, path)
			files = append(files, relPath)
		}
		
		return nil
	})
	
	return files, err
}

func (ls *LocalStorage) Directories(directory string) ([]string, error) {
	fullPath := ls.fullPath(directory)
	
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}
	
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(directory, entry.Name()))
		}
	}
	
	return dirs, nil
}

func (ls *LocalStorage) AllDirectories(directory string) ([]string, error) {
	var dirs []string
	
	err := filepath.WalkDir(ls.fullPath(directory), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() && path != ls.fullPath(directory) {
			relPath, _ := filepath.Rel(ls.root, path)
			dirs = append(dirs, relPath)
		}
		
		return nil
	})
	
	return dirs, err
}

func (ls *LocalStorage) MakeDirectory(path string) error {
	return os.MkdirAll(ls.fullPath(path), 0755)
}

func (ls *LocalStorage) DeleteDirectory(path string) error {
	return os.RemoveAll(ls.fullPath(path))
}

func (ls *LocalStorage) URL(path string) string {
	return fmt.Sprintf("/storage/%s", path)
}

func (ls *LocalStorage) TemporaryURL(path string, expiration time.Time) string {
	return ls.URL(path)
}

func (ls *LocalStorage) fullPath(path string) string {
	return filepath.Join(ls.root, path)
}

type StorageManager struct {
	disks   map[string]Storage
	default_ string
}

func NewStorageManager() *StorageManager {
	return &StorageManager{
		disks:   make(map[string]Storage),
		default_: "local",
	}
}

func (sm *StorageManager) Disk(name ...string) Storage {
	diskName := sm.default_
	if len(name) > 0 {
		diskName = name[0]
	}
	
	if disk, exists := sm.disks[diskName]; exists {
		return disk
	}
	
	return sm.createDisk(diskName)
}

func (sm *StorageManager) createDisk(name string) Storage {
	var storage Storage
	
	switch name {
	case "local":
		storage = NewLocalStorage("storage/app")
	case "public":
		storage = NewLocalStorage("storage/app/public")
	default:
		storage = NewLocalStorage("storage/app")
	}
	
	sm.disks[name] = storage
	return storage
}

func (sm *StorageManager) RegisterDisk(name string, storage Storage) {
	sm.disks[name] = storage
}

func (sm *StorageManager) SetDefaultDisk(name string) {
	sm.default_ = name
}

type UploadedFile struct {
	Header   *multipart.FileHeader
	Size     int64
	Filename string
	MimeType string
}

func NewUploadedFile(header *multipart.FileHeader) *UploadedFile {
	return &UploadedFile{
		Header:   header,
		Size:     header.Size,
		Filename: header.Filename,
		MimeType: header.Header.Get("Content-Type"),
	}
}

func (uf *UploadedFile) Store(path string, storage Storage) (string, error) {
	file, err := uf.Header.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	filename := uf.generateUniqueFilename(path)
	fullPath := filepath.Join(path, filename)
	
	err = storage.PutFile(fullPath, file)
	return fullPath, err
}

func (uf *UploadedFile) StoreAs(path, name string, storage Storage) (string, error) {
	file, err := uf.Header.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	fullPath := filepath.Join(path, name)
	err = storage.PutFile(fullPath, file)
	return fullPath, err
}

func (uf *UploadedFile) generateUniqueFilename(path string) string {
	ext := filepath.Ext(uf.Filename)
	name := strings.TrimSuffix(uf.Filename, ext)
	timestamp := time.Now().Unix()
	
	return fmt.Sprintf("%s_%d%s", name, timestamp, ext)
}

func (uf *UploadedFile) GetClientOriginalName() string {
	return uf.Filename
}

func (uf *UploadedFile) GetSize() int64 {
	return uf.Size
}

func (uf *UploadedFile) GetMimeType() string {
	return uf.MimeType
}

func (uf *UploadedFile) GetClientOriginalExtension() string {
	return filepath.Ext(uf.Filename)
}

func (uf *UploadedFile) IsValid() bool {
	return uf.Header != nil && uf.Size > 0
}

func (c *Context) File(key string) (*UploadedFile, error) {
	if c.Request.MultipartForm == nil {
		if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
			return nil, err
		}
	}
	
	file, header, err := c.Request.FormFile(key)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	return NewUploadedFile(header), nil
}

func (c *Context) HasFile(key string) bool {
	_, err := c.File(key)
	return err == nil
}

func (c *Context) Storage(disk ...string) Storage {
	storageManager, _ := c.app.Container().Make("storage")
	if sm, ok := storageManager.(*StorageManager); ok {
		return sm.Disk(disk...)
	}
	return NewLocalStorage("storage/app")
}

type FileUploadMiddleware struct {
	MaxFileSize   int64
	AllowedTypes  []string
	RequiredFiles []string
}

func NewFileUploadMiddleware() *FileUploadMiddleware {
	return &FileUploadMiddleware{
		MaxFileSize:  10 << 20, // 10MB
		AllowedTypes: []string{"image/jpeg", "image/png", "image/gif", "application/pdf"},
	}
}

func (fum *FileUploadMiddleware) Handle(c *Context) error {
	if c.Request.Method != "POST" && c.Request.Method != "PUT" && c.Request.Method != "PATCH" {
		return c.Next()
	}
	
	if err := c.Request.ParseMultipartForm(fum.MaxFileSize); err != nil {
		return c.JSON(400, map[string]string{
			"error": "Failed to parse multipart form",
		})
	}
	
	for _, requiredFile := range fum.RequiredFiles {
		if !c.HasFile(requiredFile) {
			return c.JSON(400, map[string]string{
				"error": fmt.Sprintf("Required file '%s' is missing", requiredFile),
			})
		}
	}
	
	if c.Request.MultipartForm != nil {
		for fieldName := range c.Request.MultipartForm.File {
			uploadedFile, err := c.File(fieldName)
			if err != nil {
				continue
			}
			
			if uploadedFile.GetSize() > fum.MaxFileSize {
				return c.JSON(400, map[string]string{
					"error": fmt.Sprintf("File '%s' exceeds maximum size limit", fieldName),
				})
			}
			
			if len(fum.AllowedTypes) > 0 && !fum.isAllowedType(uploadedFile.GetMimeType()) {
				return c.JSON(400, map[string]string{
					"error": fmt.Sprintf("File type '%s' is not allowed", uploadedFile.GetMimeType()),
				})
			}
		}
	}
	
	return c.Next()
}

func (fum *FileUploadMiddleware) isAllowedType(mimeType string) bool {
	for _, allowedType := range fum.AllowedTypes {
		if allowedType == mimeType {
			return true
		}
	}
	return false
}

func (fum *FileUploadMiddleware) WithMaxFileSize(size int64) *FileUploadMiddleware {
	fum.MaxFileSize = size
	return fum
}

func (fum *FileUploadMiddleware) WithAllowedTypes(types []string) *FileUploadMiddleware {
	fum.AllowedTypes = types
	return fum
}

func (fum *FileUploadMiddleware) WithRequiredFiles(files []string) *FileUploadMiddleware {
	fum.RequiredFiles = files
	return fum
}

func FileUploadMiddlewareFunc() MiddlewareFunc {
	middleware := NewFileUploadMiddleware()
	return middleware.Handle
}