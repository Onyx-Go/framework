package onyx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Context struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	params         map[string]string
	queries        url.Values
	app            *Application
	index          int
	middleware     []MiddlewareFunc
	data           map[string]interface{}
}

func NewContext(w http.ResponseWriter, r *http.Request, app *Application) *Context {
	return &Context{
		Request:        r,
		ResponseWriter: w,
		params:         make(map[string]string),
		queries:        r.URL.Query(),
		app:            app,
		index:          -1,
		data:           make(map[string]interface{}),
	}
}

func (c *Context) Param(key string) string {
	return c.params[key]
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

func (c *Context) QueryInt(key string) (int, error) {
	value := c.queries.Get(key)
	return strconv.Atoi(value)
}

func (c *Context) PostForm(key string) string {
	return c.Request.PostFormValue(key)
}

func (c *Context) JSON(code int, obj interface{}) error {
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.ResponseWriter.WriteHeader(code)
	encoder := json.NewEncoder(c.ResponseWriter)
	return encoder.Encode(obj)
}

func (c *Context) String(code int, format string, values ...interface{}) error {
	c.ResponseWriter.Header().Set("Content-Type", "text/plain")
	c.ResponseWriter.WriteHeader(code)
	_, err := fmt.Fprintf(c.ResponseWriter, format, values...)
	return err
}

func (c *Context) HTML(code int, html string) error {
	c.ResponseWriter.Header().Set("Content-Type", "text/html")
	c.ResponseWriter.WriteHeader(code)
	_, err := c.ResponseWriter.Write([]byte(html))
	return err
}

func (c *Context) Redirect(code int, location string) error {
	c.ResponseWriter.Header().Set("Location", location)
	c.ResponseWriter.WriteHeader(code)
	return nil
}

func (c *Context) Status(code int) {
	c.ResponseWriter.WriteHeader(code)
}

func (c *Context) Header(key, value string) {
	c.ResponseWriter.Header().Set(key, value)
}

func (c *Context) GetHeader(key string) string {
	return c.Request.Header.Get(key)
}

func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.ResponseWriter, cookie)
}

func (c *Context) Next() error {
	c.index++
	for c.index < len(c.middleware) {
		if err := c.middleware[c.index](c); err != nil {
			return err
		}
		c.index++
	}
	return nil
}

func (c *Context) Abort() {
	c.index = len(c.middleware)
}

func (c *Context) IsAborted() bool {
	return c.index >= len(c.middleware)
}

func (c *Context) Method() string {
	return c.Request.Method
}

func (c *Context) URL() string {
	return c.Request.URL.Path
}

func (c *Context) UserAgent() string {
	return c.Request.UserAgent()
}

func (c *Context) RemoteIP() string {
	forwarded := c.Request.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	
	realIP := c.Request.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	
	return strings.Split(c.Request.RemoteAddr, ":")[0]
}