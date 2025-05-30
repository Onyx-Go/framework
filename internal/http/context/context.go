package context

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	httpInternal "github.com/onyx-go/framework/internal/http"
)

// Context implements the HTTP context interface
type Context struct {
	request        *http.Request
	responseWriter http.ResponseWriter
	params         map[string]string
	queries        url.Values
	app            httpInternal.Application
	index          int
	middleware     []httpInternal.MiddlewareFunc
	data           map[string]interface{}
	aborted        bool
	statusCode     int
}

// NewContext creates a new HTTP context
func NewContext(w http.ResponseWriter, r *http.Request, app httpInternal.Application) *Context {
	return &Context{
		request:        r,
		responseWriter: w,
		params:         make(map[string]string),
		queries:        r.URL.Query(),
		app:            app,
		index:          -1,
		middleware:     make([]httpInternal.MiddlewareFunc, 0),
		data:           make(map[string]interface{}),
		aborted:        false,
		statusCode:     200,
	}
}

// Request methods implementation

func (c *Context) Method() string {
	return c.request.Method
}

func (c *Context) URL() string {
	return c.request.URL.String()
}

func (c *Context) Path() string {
	return c.request.URL.Path
}

func (c *Context) Query(key string) string {
	return c.queries.Get(key)
}

func (c *Context) QueryDefault(key, defaultValue string) string {
	if value := c.queries.Get(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Context) Param(key string) string {
	return c.params[key]
}

func (c *Context) Header(key string) string {
	return c.request.Header.Get(key)
}

// Response methods implementation

func (c *Context) Status(code int) {
	c.statusCode = code
	c.responseWriter.WriteHeader(code)
}

func (c *Context) SetHeader(key, value string) {
	c.responseWriter.Header().Set(key, value)
}

func (c *Context) String(code int, text string) error {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	_, err := c.responseWriter.Write([]byte(text))
	return err
}

func (c *Context) JSON(code int, data interface{}) error {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.responseWriter)
	return encoder.Encode(data)
}

func (c *Context) HTML(code int, html string) error {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	_, err := c.responseWriter.Write([]byte(html))
	return err
}

// Utility methods implementation

func (c *Context) Next() error {
	if c.aborted {
		return nil
	}
	
	c.index++
	for c.index < len(c.middleware) && !c.aborted {
		if err := c.middleware[c.index](c); err != nil {
			return err
		}
		c.index++
	}
	return nil
}

func (c *Context) Abort() {
	c.aborted = true
}

func (c *Context) IsAborted() bool {
	return c.aborted
}

func (c *Context) Set(key string, value interface{}) {
	c.data[key] = value
}

func (c *Context) Get(key string) (interface{}, bool) {
	value, exists := c.data[key]
	return value, exists
}

// Request body methods implementation

func (c *Context) Bind(obj interface{}) error {
	contentType := c.Header("Content-Type")
	
	if strings.Contains(contentType, "application/json") {
		decoder := json.NewDecoder(c.request.Body)
		return decoder.Decode(obj)
	}
	
	// For form data, we would implement form binding here
	// This is a simplified implementation
	return fmt.Errorf("unsupported content type: %s", contentType)
}

func (c *Context) Body() ([]byte, error) {
	return io.ReadAll(c.request.Body)
}

// Response writer access

func (c *Context) ResponseWriter() http.ResponseWriter {
	return c.responseWriter
}

func (c *Context) Request() *http.Request {
	return c.request
}

// Context-specific methods for router integration

// SetParam sets a route parameter value
func (c *Context) SetParam(key, value string) {
	c.params[key] = value
}

// AddMiddleware adds middleware to the context's middleware chain
func (c *Context) AddMiddleware(middleware ...httpInternal.MiddlewareFunc) {
	c.middleware = append(c.middleware, middleware...)
}

// Additional utility methods

func (c *Context) QueryInt(key string) (int, error) {
	value := c.Query(key)
	if value == "" {
		return 0, fmt.Errorf("query parameter %s not found", key)
	}
	return strconv.Atoi(value)
}

func (c *Context) PostForm(key string) string {
	return c.request.PostFormValue(key)
}

func (c *Context) GetHeader(key string) string {
	return c.request.Header.Get(key)
}

func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.request.Cookie(name)
}

func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.responseWriter, cookie)
}

func (c *Context) UserAgent() string {
	return c.request.UserAgent()
}

func (c *Context) RemoteIP() string {
	forwarded := c.request.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	
	realIP := c.request.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	
	return strings.Split(c.request.RemoteAddr, ":")[0]
}

func (c *Context) Redirect(code int, location string) error {
	c.SetHeader("Location", location)
	c.Status(code)
	return nil
}

// Application returns the application instance
func (c *Context) Application() httpInternal.Application {
	return c.app
}