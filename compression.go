package onyx

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// CompressionConfig configures compression middleware
type CompressionConfig struct {
	Level     int      // Compression level (1-9, where 1 is fastest and 9 is best compression)
	MinLength int      // Minimum response size to compress (bytes)
	Types     []string // MIME types to compress
	Exclude   []string // Path patterns to exclude from compression
}

// DefaultCompressionConfig returns sensible defaults for compression
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		Level:     gzip.DefaultCompression, // Usually 6
		MinLength: 1024,                    // Only compress responses >= 1KB
		Types: []string{
			"text/plain",
			"text/html",
			"text/css",
			"text/javascript",
			"application/javascript",
			"application/json",
			"application/xml",
			"text/xml",
			"application/xhtml+xml",
			"image/svg+xml",
		},
		Exclude: []string{
			"/health",
			"/metrics",
		},
	}
}

// CompressedResponseWriter wraps http.ResponseWriter to provide gzip compression
type CompressedResponseWriter struct {
	http.ResponseWriter
	gzipWriter  *gzip.Writer
	config      CompressionConfig
	wroteHeader bool
	statusCode  int
}

// NewCompressedResponseWriter creates a new compressed response writer
func NewCompressedResponseWriter(w http.ResponseWriter, config CompressionConfig) *CompressedResponseWriter {
	return &CompressedResponseWriter{
		ResponseWriter: w,
		config:         config,
		statusCode:     200,
	}
}

// Header returns the header map of the underlying ResponseWriter
func (crw *CompressedResponseWriter) Header() http.Header {
	return crw.ResponseWriter.Header()
}

// WriteHeader sends an HTTP response header with the provided status code
func (crw *CompressedResponseWriter) WriteHeader(statusCode int) {
	if crw.wroteHeader {
		return
	}
	
	crw.statusCode = statusCode
	crw.wroteHeader = true
	
	// Don't compress error responses or redirects
	if statusCode < 200 || statusCode >= 300 {
		crw.ResponseWriter.WriteHeader(statusCode)
		return
	}
	
	// Check if content type should be compressed
	contentType := crw.Header().Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}
	
	if crw.shouldCompress(contentType) {
		// Set compression headers
		crw.Header().Set("Content-Encoding", "gzip")
		crw.Header().Set("Vary", "Accept-Encoding")
		crw.Header().Del("Content-Length") // Gzip will handle length
		
		// Write status header
		crw.ResponseWriter.WriteHeader(statusCode)
		
		// Create gzip writer
		var err error
		crw.gzipWriter, err = gzip.NewWriterLevel(crw.ResponseWriter, crw.config.Level)
		if err != nil {
			// If we can't create gzip writer, this is a problem because we already set headers
			// Best we can do is continue without the gzip writer
			return
		}
	} else {
		// Not compressing, write headers normally
		crw.ResponseWriter.WriteHeader(statusCode)
	}
}

// Write writes the data to the connection as part of an HTTP reply
func (crw *CompressedResponseWriter) Write(data []byte) (int, error) {
	// Make sure headers are written first
	if !crw.wroteHeader {
		crw.WriteHeader(200)
	}
	
	// If we have a gzip writer, use it
	if crw.gzipWriter != nil {
		return crw.gzipWriter.Write(data)
	}
	
	// Otherwise write directly
	return crw.ResponseWriter.Write(data)
}

// shouldCompress determines if the response should be compressed based on content type
func (crw *CompressedResponseWriter) shouldCompress(contentType string) bool {
	contentType = strings.ToLower(contentType)
	for _, ct := range crw.config.Types {
		if strings.Contains(contentType, ct) {
			return true
		}
	}
	return false
}

// Close finalizes compression if active
func (crw *CompressedResponseWriter) Close() error {
	if crw.gzipWriter != nil {
		return crw.gzipWriter.Close()
	}
	return nil
}

// CompressionMiddleware returns a middleware that compresses HTTP responses
func CompressionMiddleware(config ...CompressionConfig) MiddlewareFunc {
	var cfg CompressionConfig
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultCompressionConfig()
	}
	
	return func(c *Context) error {
		// Check if client accepts gzip
		acceptEncoding := c.Request.Header.Get("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "gzip") {
			return c.Next()
		}
		
		// Check if path should be excluded
		path := c.Request.URL.Path
		for _, exclude := range cfg.Exclude {
			if strings.HasPrefix(path, exclude) {
				return c.Next()
			}
		}
		
		// Create compressed response writer
		crw := NewCompressedResponseWriter(c.ResponseWriter, cfg)
		
		// Replace the response writer
		originalWriter := c.ResponseWriter
		c.ResponseWriter = crw
		
		// Process the request
		err := c.Next()
		
		// Ensure compression is finalized
		crw.Close()
		
		// Restore original writer
		c.ResponseWriter = originalWriter
		
		return err
	}
}

// GzipMiddleware is a convenience function for default gzip compression
func GzipMiddleware() MiddlewareFunc {
	return CompressionMiddleware()
}

// CustomCompressionMiddleware allows for custom compression configuration
func CustomCompressionMiddleware(level int, minLength int, types []string) MiddlewareFunc {
	config := CompressionConfig{
		Level:     level,
		MinLength: minLength,
		Types:     types,
		Exclude:   []string{},
	}
	return CompressionMiddleware(config)
}