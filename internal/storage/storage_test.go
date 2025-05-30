package storage

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	cleanup := func() {
		os.RemoveAll(tempDir)
	}
	
	return tempDir, cleanup
}

func TestLocalDriver_BasicOperations(t *testing.T) {
	tempDir, cleanup := setupTestDir(t)
	defer cleanup()
	
	config := DefaultConfig()
	config.LocalPath = tempDir
	
	driver, err := NewLocalDriver("test", config)
	if err != nil {
		t.Fatalf("Failed to create local driver: %v", err)
	}
	
	ctx := context.Background()
	testPath := "test/file.txt"
	testContent := []byte("Hello, World!")
	
	// Test Put
	err = driver.Put(ctx, testPath, testContent)
	if err != nil {
		t.Errorf("Put failed: %v", err)
	}
	
	// Test Exists
	exists, err := driver.Exists(ctx, testPath)
	if err != nil {
		t.Errorf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("File should exist after Put")
	}
	
	// Test Get
	content, err := driver.Get(ctx, testPath)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if !bytes.Equal(content, testContent) {
		t.Errorf("Expected %s, got %s", testContent, content)
	}
	
	// Test Size
	size, err := driver.Size(ctx, testPath)
	if err != nil {
		t.Errorf("Size failed: %v", err)
	}
	if size != int64(len(testContent)) {
		t.Errorf("Expected size %d, got %d", len(testContent), size)
	}
	
	// Test Delete
	err = driver.Delete(ctx, testPath)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}
	
	// Verify deletion
	exists, err = driver.Exists(ctx, testPath)
	if err != nil {
		t.Errorf("Exists check after delete failed: %v", err)
	}
	if exists {
		t.Error("File should not exist after Delete")
	}
}

func TestLocalDriver_FileOperations(t *testing.T) {
	tempDir, cleanup := setupTestDir(t)
	defer cleanup()
	
	config := DefaultConfig()
	config.LocalPath = tempDir
	
	driver, err := NewLocalDriver("test", config)
	if err != nil {
		t.Fatalf("Failed to create local driver: %v", err)
	}
	
	ctx := context.Background()
	
	// Setup test files
	sourcePath := "source.txt"
	sourceContent := []byte("Source content")
	err = driver.Put(ctx, sourcePath, sourceContent)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	// Test Copy
	copyPath := "copy.txt"
	err = driver.Copy(ctx, sourcePath, copyPath)
	if err != nil {
		t.Errorf("Copy failed: %v", err)
	}
	
	// Verify copy
	copyContent, err := driver.Get(ctx, copyPath)
	if err != nil {
		t.Errorf("Get copy failed: %v", err)
	}
	if !bytes.Equal(copyContent, sourceContent) {
		t.Error("Copy content doesn't match source")
	}
	
	// Test Move
	movePath := "moved.txt"
	err = driver.Move(ctx, copyPath, movePath)
	if err != nil {
		t.Errorf("Move failed: %v", err)
	}
	
	// Verify move
	exists, _ := driver.Exists(ctx, copyPath)
	if exists {
		t.Error("Source file should not exist after move")
	}
	
	exists, _ = driver.Exists(ctx, movePath)
	if !exists {
		t.Error("Destination file should exist after move")
	}
	
	movedContent, err := driver.Get(ctx, movePath)
	if err != nil {
		t.Errorf("Get moved file failed: %v", err)
	}
	if !bytes.Equal(movedContent, sourceContent) {
		t.Error("Moved content doesn't match source")
	}
}

func TestLocalDriver_DirectoryOperations(t *testing.T) {
	tempDir, cleanup := setupTestDir(t)
	defer cleanup()
	
	config := DefaultConfig()
	config.LocalPath = tempDir
	
	driver, err := NewLocalDriver("test", config)
	if err != nil {
		t.Fatalf("Failed to create local driver: %v", err)
	}
	
	ctx := context.Background()
	
	// Test MakeDirectory
	dirPath := "test/subdir"
	err = driver.MakeDirectory(ctx, dirPath)
	if err != nil {
		t.Errorf("MakeDirectory failed: %v", err)
	}
	
	// Create test files
	testFiles := []string{
		"test/file1.txt",
		"test/file2.txt",
		"test/subdir/file3.txt",
	}
	
	for _, file := range testFiles {
		err = driver.Put(ctx, file, []byte("content"))
		if err != nil {
			t.Errorf("Failed to create test file %s: %v", file, err)
		}
	}
	
	// Test Files (non-recursive)
	files, err := driver.Files(ctx, "test")
	if err != nil {
		t.Errorf("Files failed: %v", err)
	}
	
	expectedFiles := 2 // file1.txt and file2.txt
	if len(files) != expectedFiles {
		t.Errorf("Expected %d files, got %d", expectedFiles, len(files))
	}
	
	// Test AllFiles (recursive)
	allFiles, err := driver.AllFiles(ctx, "test")
	if err != nil {
		t.Errorf("AllFiles failed: %v", err)
	}
	
	expectedAllFiles := 3 // all three files
	if len(allFiles) != expectedAllFiles {
		t.Errorf("Expected %d files recursively, got %d", expectedAllFiles, len(allFiles))
	}
	
	// Test Directories
	dirs, err := driver.Directories(ctx, "test")
	if err != nil {
		t.Errorf("Directories failed: %v", err)
	}
	
	if len(dirs) != 1 {
		t.Errorf("Expected 1 directory, got %d", len(dirs))
	}
	
	// Test DeleteDirectory
	err = driver.DeleteDirectory(ctx, dirPath)
	if err != nil {
		t.Errorf("DeleteDirectory failed: %v", err)
	}
	
	// Verify deletion
	exists, _ := driver.Exists(ctx, "test/subdir/file3.txt")
	if exists {
		t.Error("File in deleted directory should not exist")
	}
}

func TestLocalDriver_Metadata(t *testing.T) {
	tempDir, cleanup := setupTestDir(t)
	defer cleanup()
	
	config := DefaultConfig()
	config.LocalPath = tempDir
	
	driver, err := NewLocalDriver("test", config)
	if err != nil {
		t.Fatalf("Failed to create local driver: %v", err)
	}
	
	ctx := context.Background()
	testPath := "test/file.txt"
	
	// Create file
	err = driver.Put(ctx, testPath, []byte("content"))
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Set metadata
	metadata := map[string]string{
		"author":      "test",
		"description": "test file",
		"version":     "1.0",
	}
	
	err = driver.SetMetadata(ctx, testPath, metadata)
	if err != nil {
		t.Errorf("SetMetadata failed: %v", err)
	}
	
	// Get metadata
	retrievedMetadata, err := driver.GetMetadata(ctx, testPath)
	if err != nil {
		t.Errorf("GetMetadata failed: %v", err)
	}
	
	for key, value := range metadata {
		if retrievedMetadata[key] != value {
			t.Errorf("Metadata mismatch for key %s: expected %s, got %s", key, value, retrievedMetadata[key])
		}
	}
}

func TestLocalDriver_HealthCheck(t *testing.T) {
	tempDir, cleanup := setupTestDir(t)
	defer cleanup()
	
	config := DefaultConfig()
	config.LocalPath = tempDir
	
	driver, err := NewLocalDriver("test", config)
	if err != nil {
		t.Fatalf("Failed to create local driver: %v", err)
	}
	
	ctx := context.Background()
	
	// Test health check
	err = driver.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
	
	// Test health check with inaccessible directory
	badConfig := DefaultConfig()
	badConfig.LocalPath = "/nonexistent/directory"
	
	badDriver, err := NewLocalDriver("bad", badConfig)
	if err == nil {
		err = badDriver.HealthCheck(ctx)
		if err == nil {
			t.Error("HealthCheck should fail for inaccessible directory")
		}
	}
}

func TestStorageManager(t *testing.T) {
	tempDir, cleanup := setupTestDir(t)
	defer cleanup()
	
	// Create test directory with proper permissions
	testDir := filepath.Join(tempDir, "test")
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	
	// Create manager config
	config := DefaultManagerConfig()
	config.Drivers["test"] = Config{
		Type:        "local",
		LocalPath:   testDir,
		Permissions: DefaultPermissions(),
	}
	
	manager, err := SetupManager(config)
	if err != nil {
		t.Fatalf("Failed to setup manager: %v", err)
	}
	defer manager.Close(context.Background())
	
	// Test driver registration
	drivers := manager.ListDrivers()
	if len(drivers) == 0 {
		t.Error("Expected at least one driver")
	}
	
	// Test default driver
	defaultDriver := manager.GetDefaultDriver()
	if defaultDriver == "" {
		t.Error("Default driver should be set")
	}
	
	// Test disk access
	disk := manager.Disk("test")
	if disk == nil {
		t.Error("Disk should not be nil")
	}
	
	// Test operations through manager
	ctx := context.Background()
	testPath := "manager_test.txt"
	testContent := []byte("Manager test content")
	
	err = disk.Put(ctx, testPath, testContent)
	if err != nil {
		t.Errorf("Put through manager failed: %v", err)
	}
	
	content, err := disk.Get(ctx, testPath)
	if err != nil {
		t.Errorf("Get through manager failed: %v", err)
	}
	
	if !bytes.Equal(content, testContent) {
		t.Error("Content mismatch through manager")
	}
	
	// Test health check
	err = manager.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Manager health check failed: %v", err)
	}
	
	// Test stats
	stats, err := manager.GetStats(ctx)
	if err != nil {
		t.Errorf("GetStats failed: %v", err)
	}
	if stats == nil {
		t.Error("Stats should not be nil")
	}
}

func TestStatsCollector(t *testing.T) {
	collector := NewStatsCollector()
	
	driverName := "test"
	driverType := "local"
	
	// Initialize driver
	collector.InitializeDriver(driverName, driverType)
	
	// Record operations
	collector.RecordOperation(driverName, "Put", 100*time.Millisecond, 1024)
	collector.RecordOperation(driverName, "Get", 50*time.Millisecond, 1024)
	collector.RecordError(driverName, "Delete", 200*time.Millisecond)
	
	// Get stats
	stats := collector.GetStats()
	if stats == nil {
		t.Fatal("Stats should not be nil")
	}
	
	if len(stats.DriverStats) != 1 {
		t.Errorf("Expected 1 driver stat, got %d", len(stats.DriverStats))
	}
	
	driverStats, exists := stats.DriverStats[driverName]
	if !exists {
		t.Fatal("Driver stats should exist")
	}
	
	if driverStats.Name != driverName {
		t.Errorf("Expected driver name %s, got %s", driverName, driverStats.Name)
	}
	
	if driverStats.Type != driverType {
		t.Errorf("Expected driver type %s, got %s", driverType, driverStats.Type)
	}
	
	if driverStats.OperationsCount != 2 {
		t.Errorf("Expected 2 operations, got %d", driverStats.OperationsCount)
	}
	
	if driverStats.ErrorsCount != 1 {
		t.Errorf("Expected 1 error, got %d", driverStats.ErrorsCount)
	}
}

func TestValidationHelper(t *testing.T) {
	helper := NewValidationHelper()
	helper.SetMaxFileSize(1024). // 1KB
		SetAllowedExtensions([]string{".txt", ".jpg"}).
		SetAllowedTypes([]string{"text/plain", "image/jpeg"}).
		SetBlockedExtensions([]string{".exe"})
	
	// Test valid file
	validFile := UploadedFile{
		OriginalName: "test.txt",
		Size:         512,
		MimeType:     "text/plain",
	}
	
	err := helper.ValidateFile(validFile)
	if err != nil {
		t.Errorf("Valid file should pass validation: %v", err)
	}
	
	// Test file too large
	largeFile := UploadedFile{
		OriginalName: "large.txt",
		Size:         2048, // 2KB > 1KB limit
		MimeType:     "text/plain",
	}
	
	err = helper.ValidateFile(largeFile)
	if err == nil {
		t.Error("Large file should fail validation")
	}
	
	// Test blocked extension
	blockedFile := UploadedFile{
		OriginalName: "virus.exe",
		Size:         512,
		MimeType:     "application/octet-stream",
	}
	
	err = helper.ValidateFile(blockedFile)
	if err == nil {
		t.Error("Blocked file should fail validation")
	}
	
	// Test disallowed extension
	disallowedFile := UploadedFile{
		OriginalName: "doc.pdf",
		Size:         512,
		MimeType:     "application/pdf",
	}
	
	err = helper.ValidateFile(disallowedFile)
	if err == nil {
		t.Error("Disallowed extension should fail validation")
	}
	
	// Test disallowed MIME type
	disallowedMimeFile := UploadedFile{
		OriginalName: "test.txt",
		Size:         512,
		MimeType:     "application/pdf",
	}
	
	err = helper.ValidateFile(disallowedMimeFile)
	if err == nil {
		t.Error("Disallowed MIME type should fail validation")
	}
}

func TestFilenameValidation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal.txt", "normal.txt"},
		{"file with spaces.txt", "file_with_spaces.txt"},
		{"file<>:\"|?*.txt", "file_______.txt"},
		{"", "unnamed_file"},
		{strings.Repeat("a", 300) + ".txt", strings.Repeat("a", 251) + ".txt"},
	}
	
	for _, test := range tests {
		result := ValidateFilename(test.input)
		if result != test.expected {
			t.Errorf("ValidateFilename(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestSizeFormatter(t *testing.T) {
	formatter := &SizeFormatter{}
	
	tests := []struct {
		bytes    int64
		expected string
	}{
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}
	
	for _, test := range tests {
		result := formatter.Format(test.bytes)
		if result != test.expected {
			t.Errorf("Format(%d) = %q, expected %q", test.bytes, result, test.expected)
		}
	}
	
	// Test parsing
	parseTests := []struct {
		input    string
		expected int64
	}{
		{"512 B", 512},
		{"1 KB", 1024},
		{"1.5 KB", 1536},
		{"1 MB", 1048576},
		{"1 GB", 1073741824},
	}
	
	for _, test := range parseTests {
		result, err := formatter.Parse(test.input)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", test.input, err)
			continue
		}
		if result != test.expected {
			t.Errorf("Parse(%q) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

func TestFilePathBuilder(t *testing.T) {
	builder := NewFilePathBuilder("/base")
	
	path := builder.
		AddSegment("uploads").
		AddSegment("2023").
		AddSegment("12").
		AddSegment("file.txt").
		Build()
	
	expected := filepath.Join("/base", "uploads", "2023", "12", "file.txt")
	if path != expected {
		t.Errorf("Expected path %s, got %s", expected, path)
	}
	
	// Test date segment
	date := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)
	datePath := NewFilePathBuilder("/base").
		AddDateSegment(date).
		AddSegment("file.txt").
		Build()
	
	expectedDatePath := filepath.Join("/base", "2023", "12", "25", "file.txt")
	if datePath != expectedDatePath {
		t.Errorf("Expected date path %s, got %s", expectedDatePath, datePath)
	}
	
	// Test segment sanitization
	sanitizedPath := NewFilePathBuilder("/base").
		AddSegment("../dangerous").
		AddSegment("path/with/slashes").
		Build()
	
	expectedSanitized := filepath.Join("/base", "__dangerous", "path_with_slashes")
	if sanitizedPath != expectedSanitized {
		t.Errorf("Expected sanitized path %s, got %s", expectedSanitized, sanitizedPath)
	}
}

func TestMimeTypeDetection(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"image.jpg", "image/jpeg"},
		{"image.jpeg", "image/jpeg"},
		{"image.png", "image/png"},
		{"document.pdf", "application/pdf"},
		{"script.js", "application/javascript"},
		{"data.json", "application/json"},
		{"unknown.xyz", "application/octet-stream"},
	}
	
	for _, test := range tests {
		result := GetMimeTypeFromExtension(test.filename)
		if result != test.expected {
			t.Errorf("GetMimeTypeFromExtension(%q) = %q, expected %q", test.filename, result, test.expected)
		}
	}
}

func TestAllowedExtensions(t *testing.T) {
	allowed := []string{".jpg", ".png", ".txt"}
	
	tests := []struct {
		filename string
		expected bool
	}{
		{"image.jpg", true},
		{"image.JPG", true}, // Case insensitive
		{"image.png", true},
		{"document.txt", true},
		{"script.js", false},
		{"binary.exe", false},
	}
	
	for _, test := range tests {
		result := IsAllowedExtension(test.filename, allowed)
		if result != test.expected {
			t.Errorf("IsAllowedExtension(%q) = %v, expected %v", test.filename, result, test.expected)
		}
	}
	
	// Test empty allowed list (should allow all)
	result := IsAllowedExtension("anything.xyz", []string{})
	if !result {
		t.Error("Empty allowed list should allow all extensions")
	}
}

func TestAllowedMimeTypes(t *testing.T) {
	allowed := []string{"image/*", "text/plain", "application/json"}
	
	tests := []struct {
		mimeType string
		expected bool
	}{
		{"image/jpeg", true},
		{"image/png", true},
		{"image/gif", true},
		{"text/plain", true},
		{"text/html", false},
		{"application/json", true},
		{"application/pdf", false},
		{"video/mp4", false},
	}
	
	for _, test := range tests {
		result := IsAllowedMimeType(test.mimeType, allowed)
		if result != test.expected {
			t.Errorf("IsAllowedMimeType(%q) = %v, expected %v", test.mimeType, result, test.expected)
		}
	}
	
	// Test empty allowed list (should allow all)
	result := IsAllowedMimeType("anything/anything", []string{})
	if !result {
		t.Error("Empty allowed list should allow all MIME types")
	}
}