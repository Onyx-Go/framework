package context

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	httpInternal "github.com/onyx-go/framework/internal/http"
)

// Mock application for testing
type mockApplication struct{}

func (m *mockApplication) ErrorHandler() httpInternal.ErrorHandler {
	return &mockErrorHandler{}
}

func (m *mockApplication) TemplateEngine() httpInternal.TemplateEngine {
	return &mockTemplateEngine{}
}

type mockErrorHandler struct{}

func (m *mockErrorHandler) Handle(ctx httpInternal.Context, err error) {}

type mockTemplateEngine struct{}

func (m *mockTemplateEngine) Render(template string, data interface{}) (string, error) {
	return "", nil
}

func TestNewContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	if ctx == nil {
		t.Fatal("NewContext() returned nil")
	}

	if ctx.request != req {
		t.Error("request not set correctly")
	}

	if ctx.responseWriter != w {
		t.Error("responseWriter not set correctly")
	}

	if ctx.app != app {
		t.Error("app not set correctly")
	}

	if ctx.params == nil {
		t.Error("params should be initialized")
	}

	if ctx.data == nil {
		t.Error("data should be initialized")
	}

	if ctx.middleware == nil {
		t.Error("middleware should be initialized")
	}

	if ctx.index != -1 {
		t.Errorf("expected index -1, got %d", ctx.index)
	}

	if ctx.aborted {
		t.Error("context should not be aborted initially")
	}

	if ctx.statusCode != 200 {
		t.Errorf("expected status code 200, got %d", ctx.statusCode)
	}
}

func TestContextRequestMethods(t *testing.T) {
	req := httptest.NewRequest("POST", "/test/path?foo=bar&empty=", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-Real-IP", "192.168.1.1")
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	if ctx.Method() != "POST" {
		t.Errorf("expected method POST, got %s", ctx.Method())
	}

	if ctx.Path() != "/test/path" {
		t.Errorf("expected path /test/path, got %s", ctx.Path())
	}

	if !strings.Contains(ctx.URL(), "/test/path") {
		t.Errorf("URL should contain path: %s", ctx.URL())
	}

	if ctx.Query("foo") != "bar" {
		t.Errorf("expected query foo=bar, got %s", ctx.Query("foo"))
	}

	if ctx.Query("missing") != "" {
		t.Errorf("expected empty string for missing query, got %s", ctx.Query("missing"))
	}

	if ctx.QueryDefault("missing", "default") != "default" {
		t.Errorf("expected default value, got %s", ctx.QueryDefault("missing", "default"))
	}

	if ctx.QueryDefault("foo", "default") != "bar" {
		t.Errorf("expected existing value bar, got %s", ctx.QueryDefault("foo", "default"))
	}

	if ctx.Header("User-Agent") != "test-agent" {
		t.Errorf("expected User-Agent test-agent, got %s", ctx.Header("User-Agent"))
	}

	if ctx.UserAgent() != "test-agent" {
		t.Errorf("expected UserAgent test-agent, got %s", ctx.UserAgent())
	}

	if ctx.RemoteIP() != "192.168.1.1" {
		t.Errorf("expected RemoteIP 192.168.1.1, got %s", ctx.RemoteIP())
	}
}

func TestContextParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	// Test setting and getting params
	ctx.SetParam("id", "123")
	ctx.SetParam("name", "test")

	if ctx.Param("id") != "123" {
		t.Errorf("expected param id=123, got %s", ctx.Param("id"))
	}

	if ctx.Param("name") != "test" {
		t.Errorf("expected param name=test, got %s", ctx.Param("name"))
	}

	if ctx.Param("missing") != "" {
		t.Errorf("expected empty string for missing param, got %s", ctx.Param("missing"))
	}
}

func TestContextResponseMethods(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	// Test setting headers
	ctx.SetHeader("X-Test", "value")
	if w.Header().Get("X-Test") != "value" {
		t.Errorf("expected header X-Test=value, got %s", w.Header().Get("X-Test"))
	}

	// Test status code
	ctx.Status(404)
	if w.Code != 404 {
		t.Errorf("expected status code 404, got %d", w.Code)
	}
}

func TestContextString(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	err := ctx.String(200, "Hello World")
	if err != nil {
		t.Errorf("String() returned error: %v", err)
	}

	if w.Code != 200 {
		t.Errorf("expected status code 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("expected Content-Type text/plain, got %s", w.Header().Get("Content-Type"))
	}

	if w.Body.String() != "Hello World" {
		t.Errorf("expected body 'Hello World', got %s", w.Body.String())
	}
}

func TestContextJSON(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	data := map[string]interface{}{
		"message": "hello",
		"count":   42,
	}

	err := ctx.JSON(201, data)
	if err != nil {
		t.Errorf("JSON() returned error: %v", err)
	}

	if w.Code != 201 {
		t.Errorf("expected status code 201, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Errorf("failed to unmarshal JSON response: %v", err)
	}

	if result["message"] != "hello" {
		t.Errorf("expected message=hello, got %v", result["message"])
	}

	if result["count"] != float64(42) { // JSON numbers are float64
		t.Errorf("expected count=42, got %v", result["count"])
	}
}

func TestContextHTML(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	html := "<h1>Hello World</h1>"

	err := ctx.HTML(200, html)
	if err != nil {
		t.Errorf("HTML() returned error: %v", err)
	}

	if w.Code != 200 {
		t.Errorf("expected status code 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "text/html" {
		t.Errorf("expected Content-Type text/html, got %s", w.Header().Get("Content-Type"))
	}

	if w.Body.String() != html {
		t.Errorf("expected body '%s', got %s", html, w.Body.String())
	}
}

func TestContextDataMethods(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	// Test setting and getting data
	ctx.Set("key1", "value1")
	ctx.Set("key2", 42)

	value1, exists1 := ctx.Get("key1")
	if !exists1 {
		t.Error("expected key1 to exist")
	}
	if value1 != "value1" {
		t.Errorf("expected value1, got %v", value1)
	}

	value2, exists2 := ctx.Get("key2")
	if !exists2 {
		t.Error("expected key2 to exist")
	}
	if value2 != 42 {
		t.Errorf("expected 42, got %v", value2)
	}

	_, exists3 := ctx.Get("missing")
	if exists3 {
		t.Error("expected missing key to not exist")
	}
}

func TestContextMiddleware(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	callOrder := []string{}

	middleware1 := func(c httpInternal.Context) error {
		callOrder = append(callOrder, "middleware1")
		return c.Next()
	}

	middleware2 := func(c httpInternal.Context) error {
		callOrder = append(callOrder, "middleware2")
		return c.Next()
	}

	middleware3 := func(c httpInternal.Context) error {
		callOrder = append(callOrder, "middleware3")
		return nil // Don't call Next()
	}

	ctx.AddMiddleware(middleware1, middleware2, middleware3)

	err := ctx.Next()
	if err != nil {
		t.Errorf("Next() returned error: %v", err)
	}

	expectedOrder := []string{"middleware1", "middleware2", "middleware3"}
	if len(callOrder) != len(expectedOrder) {
		t.Errorf("expected %d middleware calls, got %d", len(expectedOrder), len(callOrder))
	}

	for i, expected := range expectedOrder {
		if i >= len(callOrder) || callOrder[i] != expected {
			t.Errorf("expected middleware call %d to be %s, got %v", i, expected, callOrder)
		}
	}
}

func TestContextAbort(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	callOrder := []string{}

	middleware1 := func(c httpInternal.Context) error {
		callOrder = append(callOrder, "middleware1")
		c.Abort()
		return nil
	}

	middleware2 := func(c httpInternal.Context) error {
		callOrder = append(callOrder, "middleware2")
		return nil
	}

	ctx.AddMiddleware(middleware1, middleware2)

	if ctx.IsAborted() {
		t.Error("context should not be aborted initially")
	}

	err := ctx.Next()
	if err != nil {
		t.Errorf("Next() returned error: %v", err)
	}

	if !ctx.IsAborted() {
		t.Error("context should be aborted after calling Abort()")
	}

	// Only middleware1 should have been called
	if len(callOrder) != 1 || callOrder[0] != "middleware1" {
		t.Errorf("expected only middleware1 to be called, got %v", callOrder)
	}
}

func TestContextBind(t *testing.T) {
	// Test JSON binding
	jsonData := `{"name":"test","value":42}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	var data struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	err := ctx.Bind(&data)
	if err != nil {
		t.Errorf("Bind() returned error: %v", err)
	}

	if data.Name != "test" {
		t.Errorf("expected name=test, got %s", data.Name)
	}

	if data.Value != 42 {
		t.Errorf("expected value=42, got %d", data.Value)
	}
}

func TestContextBody(t *testing.T) {
	bodyContent := "test body content"
	req := httptest.NewRequest("POST", "/test", strings.NewReader(bodyContent))
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	body, err := ctx.Body()
	if err != nil {
		t.Errorf("Body() returned error: %v", err)
	}

	if string(body) != bodyContent {
		t.Errorf("expected body '%s', got '%s'", bodyContent, string(body))
	}
}

func TestContextQueryInt(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?page=5&invalid=abc", nil)
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	// Test valid integer
	page, err := ctx.QueryInt("page")
	if err != nil {
		t.Errorf("QueryInt() returned error for valid int: %v", err)
	}
	if page != 5 {
		t.Errorf("expected page=5, got %d", page)
	}

	// Test invalid integer
	_, err = ctx.QueryInt("invalid")
	if err == nil {
		t.Error("QueryInt() should return error for invalid int")
	}

	// Test missing parameter
	_, err = ctx.QueryInt("missing")
	if err == nil {
		t.Error("QueryInt() should return error for missing parameter")
	}
}

func TestContextCookies(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "test", Value: "value"})
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	// Test getting cookie
	cookie, err := ctx.Cookie("test")
	if err != nil {
		t.Errorf("Cookie() returned error: %v", err)
	}
	if cookie.Value != "value" {
		t.Errorf("expected cookie value 'value', got '%s'", cookie.Value)
	}

	// Test setting cookie
	newCookie := &http.Cookie{Name: "new", Value: "newvalue"}
	ctx.SetCookie(newCookie)

	setCookieHeader := w.Header().Get("Set-Cookie")
	if !strings.Contains(setCookieHeader, "new=newvalue") {
		t.Errorf("expected Set-Cookie header to contain 'new=newvalue', got '%s'", setCookieHeader)
	}
}

func TestContextRedirect(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	err := ctx.Redirect(302, "/new-location")
	if err != nil {
		t.Errorf("Redirect() returned error: %v", err)
	}

	if w.Code != 302 {
		t.Errorf("expected status code 302, got %d", w.Code)
	}

	if w.Header().Get("Location") != "/new-location" {
		t.Errorf("expected Location header '/new-location', got '%s'", w.Header().Get("Location"))
	}
}

func TestContextPostForm(t *testing.T) {
	form := url.Values{}
	form.Add("username", "testuser")
	form.Add("password", "testpass")

	req := httptest.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	if ctx.PostForm("username") != "testuser" {
		t.Errorf("expected username=testuser, got %s", ctx.PostForm("username"))
	}

	if ctx.PostForm("password") != "testpass" {
		t.Errorf("expected password=testpass, got %s", ctx.PostForm("password"))
	}

	if ctx.PostForm("missing") != "" {
		t.Errorf("expected empty string for missing form field, got %s", ctx.PostForm("missing"))
	}
}

func TestContextResponseWriter(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	if ctx.ResponseWriter() != w {
		t.Error("ResponseWriter() should return the original response writer")
	}

	if ctx.Request() != req {
		t.Error("Request() should return the original request")
	}
}

func TestContextApplication(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app := &mockApplication{}

	ctx := NewContext(w, req, app)

	if ctx.Application() != app {
		t.Error("Application() should return the application instance")
	}
}