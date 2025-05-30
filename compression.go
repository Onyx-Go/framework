package onyx

import (
	"compress/gzip"
	"net/http"
	"strings"
	
	httpInternal "github.com/onyx-go/framework/internal/http"
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
	gzipWriter     *gzip.Writer
	config         CompressionConfig
	wroteHeader    bool
	statusCode     int
	buffer         []byte
	compressionSet bool
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
	
	// Don't compress error responses or redirects - write immediately
	if statusCode < 200 || statusCode >= 300 {
		crw.wroteHeader = true
		crw.compressionSet = true
		crw.ResponseWriter.WriteHeader(statusCode)
		return
	}
	
	// For success responses, delay writing headers until we have content
	// This will be handled in the Write method
}

// Write writes the data to the connection as part of an HTTP reply
func (crw *CompressedResponseWriter) Write(data []byte) (int, error) {
	// Buffer the data until we can make compression decision
	if !crw.compressionSet {
		crw.buffer = append(crw.buffer, data...)
		
		// Get content type and length to decide compression
		contentType := crw.Header().Get("Content-Type")
		if contentType == "" {
			// Try to detect content type from data
			contentType = http.DetectContentType(crw.buffer)
			crw.Header().Set("Content-Type", contentType)
		}
		
		// Decide compression based on content type and length
		shouldCompress := len(crw.buffer) >= crw.config.MinLength && crw.shouldCompress(contentType)
		
		if shouldCompress && crw.statusCode >= 200 && crw.statusCode < 300 {
			// Set compression headers
			crw.Header().Set("Content-Encoding", "gzip")
			crw.Header().Set("Vary", "Accept-Encoding")
			crw.Header().Del("Content-Length") // Gzip will handle length
			
			// Create gzip writer
			var err error
			crw.gzipWriter, err = gzip.NewWriterLevel(crw.ResponseWriter, crw.config.Level)
			if err != nil {
				// Fall back to no compression
				crw.compressionSet = true
				return crw.writeBuffered()
			}
		}
		
		crw.compressionSet = true
		
		// Write headers if not already written
		if !crw.wroteHeader {
			crw.ResponseWriter.WriteHeader(crw.statusCode)
			crw.wroteHeader = true
		}
		
		// Write buffered data
		return crw.writeBuffered()
	}
	
	// If we have a gzip writer, use it
	if crw.gzipWriter != nil {
		return crw.gzipWriter.Write(data)
	}
	
	// Otherwise write directly
	return crw.ResponseWriter.Write(data)
}

// writeBuffered writes all buffered data
func (crw *CompressedResponseWriter) writeBuffered() (int, error) {
	if len(crw.buffer) == 0 {
		return 0, nil
	}
	
	var written int
	var err error
	
	if crw.gzipWriter != nil {
		written, err = crw.gzipWriter.Write(crw.buffer)
	} else {
		written, err = crw.ResponseWriter.Write(crw.buffer)
	}
	
	crw.buffer = nil // Clear buffer
	return written, err
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
	
	return func(c Context) error {
		// Check if client accepts gzip
		acceptEncoding := c.Request().Header.Get("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "gzip") {
			return c.Next()
		}
		
		// Check if path should be excluded
		path := c.Request().URL.Path
		for _, exclude := range cfg.Exclude {
			if strings.HasPrefix(path, exclude) {
				return c.Next()
			}
		}
		
		// Create a buffer response writer to capture the response
		bufferWriter := &BufferResponseWriter{
			ResponseWriter: c.ResponseWriter(),
		}
		
		// For now, just call Next() without buffering
		// The middleware detection should handle using new-style compression
		err := c.Next()
		
		if err != nil {
			return err
		}
		
		// Now check if we should compress the buffered content
		contentType := bufferWriter.Header().Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(bufferWriter.body)
		}
		
		println("DEBUG: Buffer size:", len(bufferWriter.body))
		println("DEBUG: Status code:", bufferWriter.statusCode)
		println("DEBUG: Content type:", contentType)
		println("DEBUG: Min length:", cfg.MinLength)
		
		shouldCompress := len(bufferWriter.body) >= cfg.MinLength && 
						 bufferWriter.statusCode >= 200 && 
						 bufferWriter.statusCode < 300 &&
						 shouldCompressContentType(contentType, cfg)
		
		println("DEBUG: Should compress:", shouldCompress)
		
		if shouldCompress {
			// Set compression headers
			c.ResponseWriter().Header().Set("Content-Encoding", "gzip")
			c.ResponseWriter().Header().Set("Vary", "Accept-Encoding")
			c.ResponseWriter().Header().Del("Content-Length")
			
			// Copy other headers
			for key, values := range bufferWriter.Header() {
				if key != "Content-Length" {
					for _, value := range values {
						c.ResponseWriter().Header().Add(key, value)
					}
				}
			}
			
			// Write status code
			c.ResponseWriter().WriteHeader(bufferWriter.statusCode)
			
			// Compress and write the content
			gzipWriter, err := gzip.NewWriterLevel(c.ResponseWriter(), cfg.Level)
			if err != nil {
				return err
			}
			defer gzipWriter.Close()
			
			_, err = gzipWriter.Write(bufferWriter.body)
			return err
		} else {
			// Write uncompressed
			for key, values := range bufferWriter.Header() {
				for _, value := range values {
					c.ResponseWriter().Header().Add(key, value)
				}
			}
			c.ResponseWriter().WriteHeader(bufferWriter.statusCode)
			_, err := c.ResponseWriter().Write(bufferWriter.body)
			return err
		}
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

// NewStyleCompressionMiddleware returns a new-style middleware that compresses HTTP responses
func NewStyleCompressionMiddleware(config ...CompressionConfig) httpInternal.MiddlewareFunc {
	var cfg CompressionConfig
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultCompressionConfig()
	}
	
	return func(c httpInternal.Context) error {
		// Check if client accepts gzip
		acceptEncoding := c.Request().Header.Get("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "gzip") {
			return c.Next()
		}
		
		// Check if path should be excluded
		path := c.Request().URL.Path
		for _, exclude := range cfg.Exclude {
			if strings.HasPrefix(path, exclude) {
				return c.Next()
			}
		}
		
		// Create a buffer to capture the response
		buffer := &BufferResponseWriter{
			ResponseWriter: c.ResponseWriter(),
		}
		
		// Replace the response writer temporarily using reflection/unsafe
		originalWriter := c.ResponseWriter()
		
		// We need to modify the context to use our buffer writer
		// Since the internal context uses ResponseWriter() method calls, 
		// we can create a temporary wrapper
		wrapper := &responseWriterReplacer{
			Context: c,
			writer:  buffer,
		}
		
		// Call next - this should write to our buffer
		err := wrapper.Next()
		if err != nil {
			return err
		}
		
		println("DEBUG NEW: Buffer size:", len(buffer.body))
		println("DEBUG NEW: Status code:", buffer.statusCode)
		
		// Now process the buffered response
		contentType := buffer.Header().Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(buffer.body)
		}
		
		shouldCompress := len(buffer.body) >= cfg.MinLength && 
						 buffer.statusCode >= 200 && 
						 buffer.statusCode < 300 &&
						 shouldCompressContentType(contentType, cfg)
		
		println("DEBUG NEW: Should compress:", shouldCompress)
		
		if shouldCompress {
			// Set compression headers on original writer
			originalWriter.Header().Set("Content-Encoding", "gzip")
			originalWriter.Header().Set("Vary", "Accept-Encoding")
			originalWriter.Header().Del("Content-Length")
			
			// Copy other headers from buffer to original
			for key, values := range buffer.Header() {
				if key != "Content-Length" {
					for _, value := range values {
						originalWriter.Header().Set(key, value)
					}
				}
			}
			
			// Write status
			originalWriter.WriteHeader(buffer.statusCode)
			
			// Compress and write content
			gzipWriter, err := gzip.NewWriterLevel(originalWriter, cfg.Level)
			if err != nil {
				return err
			}
			defer gzipWriter.Close()
			
			_, err = gzipWriter.Write(buffer.body)
			return err
		} else {
			// Write uncompressed
			for key, values := range buffer.Header() {
				for _, value := range values {
					originalWriter.Header().Set(key, value)
				}
			}
			originalWriter.WriteHeader(buffer.statusCode)
			_, err := originalWriter.Write(buffer.body)
			return err
		}
	}
}

// compressedContextWrapper wraps a Context to provide a different ResponseWriter
type compressedContextWrapper struct {
	Context
	writer http.ResponseWriter
}

// ResponseWriter returns the wrapped response writer
func (c *compressedContextWrapper) ResponseWriter() http.ResponseWriter {
	return c.writer
}

// newStyleCompressedContextWrapper wraps a new-style Context
type newStyleCompressedContextWrapper struct {
	httpInternal.Context
	writer *CompressedResponseWriter
}

// ResponseWriter returns the compressed response writer
func (c *newStyleCompressedContextWrapper) ResponseWriter() http.ResponseWriter {
	return c.writer
}

// BufferResponseWriter buffers the response to allow post-processing
type BufferResponseWriter struct {
	http.ResponseWriter
	body       []byte
	statusCode int
}

func (b *BufferResponseWriter) WriteHeader(statusCode int) {
	if b.statusCode == 0 {
		b.statusCode = statusCode
	}
}

func (b *BufferResponseWriter) Write(data []byte) (int, error) {
	if b.statusCode == 0 {
		b.statusCode = 200
	}
	b.body = append(b.body, data...)
	return len(data), nil
}

// shouldCompressContentType checks if content type should be compressed
func shouldCompressContentType(contentType string, config CompressionConfig) bool {
	contentType = strings.ToLower(contentType)
	for _, ct := range config.Types {
		if strings.Contains(contentType, ct) {
			return true
		}
	}
	return false
}

// responseWriterReplacer wraps a context to replace the ResponseWriter
type responseWriterReplacer struct {
	httpInternal.Context
	writer http.ResponseWriter
}

func (r *responseWriterReplacer) ResponseWriter() http.ResponseWriter {
	return r.writer
}