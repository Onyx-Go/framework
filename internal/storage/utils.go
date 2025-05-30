package storage

import (
	"crypto/rand"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// StatsCollector collects and manages storage statistics
type StatsCollector struct {
	stats Stats
	mutex sync.RWMutex
}

// NewStatsCollector creates a new statistics collector
func NewStatsCollector() *StatsCollector {
	return &StatsCollector{
		stats: Stats{
			DriverStats:     make(map[string]DriverStats),
			OperationsCount: make(map[string]int64),
			AverageLatency:  make(map[string]time.Duration),
			ErrorRate:       make(map[string]float64),
			CollectedAt:     time.Now(),
		},
	}
}

// InitializeDriver initializes statistics for a driver
func (sc *StatsCollector) InitializeDriver(name, driverType string) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	
	sc.stats.DriverStats[name] = DriverStats{
		Name:         name,
		Type:         driverType,
		HealthStatus: "unknown",
	}
}

// RecordOperation records a successful operation
func (sc *StatsCollector) RecordOperation(driverName, operation string, duration time.Duration, size int64) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	
	// Update driver stats
	driverStats := sc.stats.DriverStats[driverName]
	driverStats.LastOperation = time.Now()
	driverStats.OperationsCount++
	driverStats.TotalSize += size
	
	// Update average latency
	if driverStats.OperationsCount == 1 {
		driverStats.AverageLatency = duration
	} else {
		// Calculate running average
		total := driverStats.AverageLatency*time.Duration(driverStats.OperationsCount-1) + duration
		driverStats.AverageLatency = total / time.Duration(driverStats.OperationsCount)
	}
	
	sc.stats.DriverStats[driverName] = driverStats
	
	// Update global stats
	operationKey := fmt.Sprintf("%s.%s", driverName, operation)
	sc.stats.OperationsCount[operationKey]++
	
	// Update global average latency
	if latency, exists := sc.stats.AverageLatency[operationKey]; exists {
		count := sc.stats.OperationsCount[operationKey]
		total := latency*time.Duration(count-1) + duration
		sc.stats.AverageLatency[operationKey] = total / time.Duration(count)
	} else {
		sc.stats.AverageLatency[operationKey] = duration
	}
	
	sc.stats.CollectedAt = time.Now()
}

// RecordError records a failed operation
func (sc *StatsCollector) RecordError(driverName, operation string, duration time.Duration) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	
	// Update driver stats
	driverStats := sc.stats.DriverStats[driverName]
	driverStats.ErrorsCount++
	driverStats.LastOperation = time.Now()
	sc.stats.DriverStats[driverName] = driverStats
	
	// Update error rate
	operationKey := fmt.Sprintf("%s.%s", driverName, operation)
	totalOps := sc.stats.OperationsCount[operationKey] + 1 // Include this error
	errorRate := float64(driverStats.ErrorsCount) / float64(totalOps)
	sc.stats.ErrorRate[operationKey] = errorRate
	
	sc.stats.CollectedAt = time.Now()
}

// GetStats returns a copy of current statistics
func (sc *StatsCollector) GetStats() *Stats {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	
	// Create a deep copy
	statsCopy := Stats{
		DriverStats:     make(map[string]DriverStats),
		OperationsCount: make(map[string]int64),
		AverageLatency:  make(map[string]time.Duration),
		ErrorRate:       make(map[string]float64),
		CollectedAt:     sc.stats.CollectedAt,
		Period:          time.Since(sc.stats.CollectedAt),
	}
	
	// Copy driver stats
	for name, stats := range sc.stats.DriverStats {
		statsCopy.DriverStats[name] = stats
		statsCopy.TotalFiles += stats.FilesCount
		statsCopy.TotalSize += stats.TotalSize
	}
	
	// Copy operation stats
	for key, count := range sc.stats.OperationsCount {
		statsCopy.OperationsCount[key] = count
	}
	
	for key, latency := range sc.stats.AverageLatency {
		statsCopy.AverageLatency[key] = latency
	}
	
	for key, rate := range sc.stats.ErrorRate {
		statsCopy.ErrorRate[key] = rate
	}
	
	return &statsCopy
}

// UploadedFileFromMultipart creates an UploadedFile from multipart.FileHeader
func UploadedFileFromMultipart(fieldName string, header *multipart.FileHeader) (*UploadedFile, error) {
	file, err := header.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()
	
	// Create temporary file
	tempFile, err := createTempFile(header.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	
	// Copy content to temp file
	if _, err := io.Copy(tempFile, file); err != nil {
		tempFile.Close()
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}
	tempFile.Close()
	
	return &UploadedFile{
		FieldName:    fieldName,
		OriginalName: header.Filename,
		Size:         header.Size,
		MimeType:     header.Header.Get("Content-Type"),
		Extension:    filepath.Ext(header.Filename),
		TempPath:     tempFile.Name(),
		Headers:      header.Header,
		Metadata:     make(map[string]interface{}),
	}, nil
}

// createTempFile creates a temporary file
func createTempFile(originalName string) (*os.File, error) {
	ext := filepath.Ext(originalName)
	pattern := fmt.Sprintf("upload_*%s", ext)
	
	return os.CreateTemp("", pattern)
}

// GenerateUniqueFilename generates a unique filename
func GenerateUniqueFilename(originalName string) string {
	ext := filepath.Ext(originalName)
	name := strings.TrimSuffix(originalName, ext)
	
	// Generate random suffix
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	suffix := fmt.Sprintf("%x", randomBytes)
	
	timestamp := time.Now().Unix()
	
	return fmt.Sprintf("%s_%d_%s%s", name, timestamp, suffix, ext)
}

// ValidateFilename validates and sanitizes a filename
func ValidateFilename(filename string) string {
	// Remove path separators
	filename = filepath.Base(filename)
	
	// Handle empty filename first
	if filename == "" || filename == "." || filename == ".." {
		return "unnamed_file"
	}
	
	// Replace problematic characters
	problematic := []string{" ", "<", ">", ":", "\"", "|", "?", "*", "\x00"}
	for _, char := range problematic {
		filename = strings.ReplaceAll(filename, char, "_")
	}
	
	// Limit filename length
	if len(filename) > 255 {
		ext := filepath.Ext(filename)
		name := strings.TrimSuffix(filename, ext)
		maxNameLen := 255 - len(ext)
		if maxNameLen > 0 {
			filename = name[:maxNameLen] + ext
		} else {
			filename = "file" + ext
		}
	}
	
	return filename
}

// GetMimeTypeFromExtension returns MIME type from file extension
func GetMimeTypeFromExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".txt":  "text/plain",
		".csv":  "text/csv",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".zip":  "application/zip",
		".rar":  "application/vnd.rar",
		".7z":   "application/x-7z-compressed",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
	}
	
	if mimeType, exists := mimeTypes[ext]; exists {
		return mimeType
	}
	
	return "application/octet-stream"
}

// IsAllowedExtension checks if file extension is allowed
func IsAllowedExtension(filename string, allowedExtensions []string) bool {
	if len(allowedExtensions) == 0 {
		return true // No restrictions
	}
	
	ext := strings.ToLower(filepath.Ext(filename))
	
	for _, allowed := range allowedExtensions {
		if strings.ToLower(allowed) == ext {
			return true
		}
	}
	
	return false
}

// IsAllowedMimeType checks if MIME type is allowed
func IsAllowedMimeType(mimeType string, allowedTypes []string) bool {
	if len(allowedTypes) == 0 {
		return true // No restrictions
	}
	
	mimeType = strings.ToLower(mimeType)
	
	for _, allowed := range allowedTypes {
		if strings.ToLower(allowed) == mimeType {
			return true
		}
		
		// Check wildcard patterns (e.g., "image/*")
		if strings.HasSuffix(allowed, "/*") {
			prefix := strings.TrimSuffix(allowed, "/*")
			if strings.HasPrefix(mimeType, prefix+"/") {
				return true
			}
		}
	}
	
	return false
}

// IsBlockedExtension checks if file extension is blocked
func IsBlockedExtension(filename string, blockedExtensions []string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	
	for _, blocked := range blockedExtensions {
		if strings.ToLower(blocked) == ext {
			return true
		}
	}
	
	return false
}

// FilePathBuilder helps build file paths safely
type FilePathBuilder struct {
	basePath string
	segments []string
}

// NewFilePathBuilder creates a new file path builder
func NewFilePathBuilder(basePath string) *FilePathBuilder {
	return &FilePathBuilder{
		basePath: basePath,
		segments: make([]string, 0),
	}
}

// AddSegment adds a path segment
func (fpb *FilePathBuilder) AddSegment(segment string) *FilePathBuilder {
	// Sanitize segment
	segment = strings.TrimSpace(segment)
	segment = strings.ReplaceAll(segment, "..", "_")
	segment = strings.ReplaceAll(segment, "/", "_")
	segment = strings.ReplaceAll(segment, "\\", "_")
	
	if segment != "" {
		fpb.segments = append(fpb.segments, segment)
	}
	
	return fpb
}

// AddDateSegment adds a date-based segment (YYYY/MM/DD)
func (fpb *FilePathBuilder) AddDateSegment(date time.Time) *FilePathBuilder {
	return fpb.AddSegment(date.Format("2006")).
		AddSegment(date.Format("01")).
		AddSegment(date.Format("02"))
}

// Build returns the final path
func (fpb *FilePathBuilder) Build() string {
	if len(fpb.segments) == 0 {
		return fpb.basePath
	}
	
	return filepath.Join(append([]string{fpb.basePath}, fpb.segments...)...)
}

// SizeFormatter formats file sizes in human-readable format
type SizeFormatter struct{}

// Format formats a size in bytes to human-readable format
func (sf *SizeFormatter) Format(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}
	
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// Parse parses a human-readable size to bytes
func (sf *SizeFormatter) Parse(size string) (int64, error) {
	size = strings.TrimSpace(strings.ToUpper(size))
	
	// Check units in order of length (longest first) to avoid conflicts
	units := []struct {
		suffix     string
		multiplier int64
	}{
		{"PB", 1024 * 1024 * 1024 * 1024 * 1024},
		{"TB", 1024 * 1024 * 1024 * 1024},
		{"GB", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}
	
	for _, unit := range units {
		if strings.HasSuffix(size, unit.suffix) {
			numberPart := strings.TrimSuffix(size, unit.suffix)
			numberPart = strings.TrimSpace(numberPart)
			
			var number float64
			if _, err := fmt.Sscanf(numberPart, "%f", &number); err != nil {
				return 0, fmt.Errorf("invalid size format: %s", size)
			}
			
			return int64(number * float64(unit.multiplier)), nil
		}
	}
	
	// Try to parse as plain number (assume bytes)
	var number int64
	if _, err := fmt.Sscanf(size, "%d", &number); err != nil {
		return 0, fmt.Errorf("invalid size format: %s", size)
	}
	
	return number, nil
}

// ValidationHelper provides file validation utilities
type ValidationHelper struct {
	maxFileSize       int64
	allowedTypes      []string
	allowedExtensions []string
	blockedExtensions []string
}

// NewValidationHelper creates a new validation helper
func NewValidationHelper() *ValidationHelper {
	return &ValidationHelper{
		maxFileSize:       10 << 20, // 10MB default
		allowedTypes:      []string{},
		allowedExtensions: []string{},
		blockedExtensions: []string{".exe", ".bat", ".cmd", ".scr"},
	}
}

// SetMaxFileSize sets the maximum file size
func (vh *ValidationHelper) SetMaxFileSize(size int64) *ValidationHelper {
	vh.maxFileSize = size
	return vh
}

// SetAllowedTypes sets allowed MIME types
func (vh *ValidationHelper) SetAllowedTypes(types []string) *ValidationHelper {
	vh.allowedTypes = types
	return vh
}

// SetAllowedExtensions sets allowed file extensions
func (vh *ValidationHelper) SetAllowedExtensions(extensions []string) *ValidationHelper {
	vh.allowedExtensions = extensions
	return vh
}

// SetBlockedExtensions sets blocked file extensions
func (vh *ValidationHelper) SetBlockedExtensions(extensions []string) *ValidationHelper {
	vh.blockedExtensions = extensions
	return vh
}

// ValidateFile validates an uploaded file
func (vh *ValidationHelper) ValidateFile(file UploadedFile) error {
	// Check file size
	if file.Size > vh.maxFileSize {
		return fmt.Errorf("file size (%s) exceeds maximum allowed size (%s)",
			new(SizeFormatter).Format(file.Size),
			new(SizeFormatter).Format(vh.maxFileSize))
	}
	
	// Check blocked extensions
	if IsBlockedExtension(file.OriginalName, vh.blockedExtensions) {
		return fmt.Errorf("file extension %s is not allowed", filepath.Ext(file.OriginalName))
	}
	
	// Check allowed extensions
	if !IsAllowedExtension(file.OriginalName, vh.allowedExtensions) {
		return fmt.Errorf("file extension %s is not allowed", filepath.Ext(file.OriginalName))
	}
	
	// Check allowed MIME types
	if !IsAllowedMimeType(file.MimeType, vh.allowedTypes) {
		return fmt.Errorf("file type %s is not allowed", file.MimeType)
	}
	
	return nil
}

// Global instances for convenience
var (
	defaultSizeFormatter    = &SizeFormatter{}
	defaultValidationHelper = NewValidationHelper()
)

// Package-level convenience functions

// FormatSize formats a file size in human-readable format
func FormatSize(bytes int64) string {
	return defaultSizeFormatter.Format(bytes)
}

// ParseSize parses a human-readable size to bytes
func ParseSize(size string) (int64, error) {
	return defaultSizeFormatter.Parse(size)
}

// ValidateUploadedFile validates an uploaded file with default settings
func ValidateUploadedFile(file UploadedFile) error {
	return defaultValidationHelper.ValidateFile(file)
}

// SanitizeFilename sanitizes a filename
func SanitizeFilename(filename string) string {
	return ValidateFilename(filename)
}