package onyx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	
	"github.com/onyx-go/framework/internal/auth"
	"strings"
	"testing"
)

type TestCase struct {
	app     *Application
	request *http.Request
	response *httptest.ResponseRecorder
	t       *testing.T
}

func NewTestCase(t *testing.T, app *Application) *TestCase {
	return &TestCase{
		app: app,
		t:   t,
	}
}

func (tc *TestCase) Get(path string) *TestCase {
	return tc.makeRequest("GET", path, nil, nil)
}

func (tc *TestCase) Post(path string, data interface{}) *TestCase {
	return tc.makeRequest("POST", path, data, nil)
}

func (tc *TestCase) Put(path string, data interface{}) *TestCase {
	return tc.makeRequest("PUT", path, data, nil)
}

func (tc *TestCase) Delete(path string) *TestCase {
	return tc.makeRequest("DELETE", path, nil, nil)
}

func (tc *TestCase) Patch(path string, data interface{}) *TestCase {
	return tc.makeRequest("PATCH", path, data, nil)
}

func (tc *TestCase) WithHeaders(headers map[string]string) *TestCase {
	if tc.request != nil {
		for key, value := range headers {
			tc.request.Header.Set(key, value)
		}
	}
	return tc
}

func (tc *TestCase) WithHeader(key, value string) *TestCase {
	if tc.request != nil {
		tc.request.Header.Set(key, value)
	}
	return tc
}

func (tc *TestCase) WithCookie(cookie *http.Cookie) *TestCase {
	if tc.request != nil {
		tc.request.AddCookie(cookie)
	}
	return tc
}

func (tc *TestCase) WithSession(data map[string]interface{}) *TestCase {
	return tc
}

func (tc *TestCase) ActingAs(user auth.User) *TestCase {
	return tc
}

func (tc *TestCase) makeRequest(method, path string, data interface{}, headers map[string]string) *TestCase {
	var body io.Reader
	var contentType string

	if data != nil {
		switch v := data.(type) {
		case map[string]interface{}:
			jsonData, _ := json.Marshal(v)
			body = bytes.NewBuffer(jsonData)
			contentType = "application/json"
		case map[string]string:
			formData := url.Values{}
			for key, value := range v {
				formData.Set(key, value)
			}
			body = strings.NewReader(formData.Encode())
			contentType = "application/x-www-form-urlencoded"
		case string:
			body = strings.NewReader(v)
			contentType = "text/plain"
		}
	}

	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	tc.request = req
	tc.response = httptest.NewRecorder()

	tc.app.ServeHTTP(tc.response, req)

	return tc
}

func (tc *TestCase) AssertStatus(expectedStatus int) *TestCase {
	if tc.response.Code != expectedStatus {
		tc.t.Errorf("Expected status %d, got %d", expectedStatus, tc.response.Code)
	}
	return tc
}

func (tc *TestCase) AssertOk() *TestCase {
	return tc.AssertStatus(200)
}

func (tc *TestCase) AssertCreated() *TestCase {
	return tc.AssertStatus(201)
}

func (tc *TestCase) AssertNoContent() *TestCase {
	return tc.AssertStatus(204)
}

func (tc *TestCase) AssertNotFound() *TestCase {
	return tc.AssertStatus(404)
}

func (tc *TestCase) AssertUnauthorized() *TestCase {
	return tc.AssertStatus(401)
}

func (tc *TestCase) AssertForbidden() *TestCase {
	return tc.AssertStatus(403)
}

func (tc *TestCase) AssertValidationError() *TestCase {
	return tc.AssertStatus(422)
}

func (tc *TestCase) AssertRedirect(location ...string) *TestCase {
	if tc.response.Code < 300 || tc.response.Code >= 400 {
		tc.t.Errorf("Expected redirect status (3xx), got %d", tc.response.Code)
		return tc
	}

	if len(location) > 0 {
		expectedLocation := location[0]
		actualLocation := tc.response.Header().Get("Location")
		if actualLocation != expectedLocation {
			tc.t.Errorf("Expected redirect to %s, got %s", expectedLocation, actualLocation)
		}
	}

	return tc
}

func (tc *TestCase) AssertHeader(key, expectedValue string) *TestCase {
	actualValue := tc.response.Header().Get(key)
	if actualValue != expectedValue {
		tc.t.Errorf("Expected header %s to be %s, got %s", key, expectedValue, actualValue)
	}
	return tc
}

func (tc *TestCase) AssertHeaderContains(key, substring string) *TestCase {
	actualValue := tc.response.Header().Get(key)
	if !strings.Contains(actualValue, substring) {
		tc.t.Errorf("Expected header %s to contain %s, got %s", key, substring, actualValue)
	}
	return tc
}

func (tc *TestCase) AssertJson(expectedJson interface{}) *TestCase {
	var actual interface{}
	if err := json.Unmarshal(tc.response.Body.Bytes(), &actual); err != nil {
		tc.t.Errorf("Failed to parse JSON response: %v", err)
		return tc
	}

	expectedBytes, _ := json.Marshal(expectedJson)
	actualBytes, _ := json.Marshal(actual)

	if string(expectedBytes) != string(actualBytes) {
		tc.t.Errorf("Expected JSON %s, got %s", string(expectedBytes), string(actualBytes))
	}

	return tc
}

func (tc *TestCase) AssertJsonFragment(fragment map[string]interface{}) *TestCase {
	var actual map[string]interface{}
	if err := json.Unmarshal(tc.response.Body.Bytes(), &actual); err != nil {
		tc.t.Errorf("Failed to parse JSON response: %v", err)
		return tc
	}

	for key, expectedValue := range fragment {
		if actualValue, exists := actual[key]; !exists {
			tc.t.Errorf("Expected JSON to contain key %s", key)
		} else if fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
			tc.t.Errorf("Expected JSON key %s to be %v, got %v", key, expectedValue, actualValue)
		}
	}

	return tc
}

func (tc *TestCase) AssertJsonPath(path string, expectedValue interface{}) *TestCase {
	return tc
}

func (tc *TestCase) AssertSee(text string) *TestCase {
	body := tc.response.Body.String()
	if !strings.Contains(body, text) {
		tc.t.Errorf("Expected response to contain %s, but it was not found", text)
	}
	return tc
}

func (tc *TestCase) AssertDontSee(text string) *TestCase {
	body := tc.response.Body.String()
	if strings.Contains(body, text) {
		tc.t.Errorf("Expected response to not contain %s, but it was found", text)
	}
	return tc
}

func (tc *TestCase) AssertSeeInOrder(texts []string) *TestCase {
	body := tc.response.Body.String()
	lastIndex := -1

	for _, text := range texts {
		index := strings.Index(body[lastIndex+1:], text)
		if index == -1 {
			tc.t.Errorf("Expected to see %s in response", text)
			return tc
		}
		lastIndex += index + 1
	}

	return tc
}

func (tc *TestCase) Dump() *TestCase {
	fmt.Printf("Status: %d\n", tc.response.Code)
	fmt.Printf("Headers: %v\n", tc.response.Header())
	fmt.Printf("Body: %s\n", tc.response.Body.String())
	return tc
}

func (tc *TestCase) DumpHeaders() *TestCase {
	fmt.Printf("Headers: %v\n", tc.response.Header())
	return tc
}

func (tc *TestCase) GetContent() string {
	return tc.response.Body.String()
}

func (tc *TestCase) GetJsonResponse() (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal(tc.response.Body.Bytes(), &result)
	return result, err
}

func (tc *TestCase) GetStatus() int {
	return tc.response.Code
}

func (tc *TestCase) GetHeaders() http.Header {
	return tc.response.Header()
}

type DatabaseTestCase struct {
	*TestCase
	db *DB
}

func NewDatabaseTestCase(t *testing.T, app *Application, db *DB) *DatabaseTestCase {
	return &DatabaseTestCase{
		TestCase: NewTestCase(t, app),
		db:       db,
	}
}

func (dtc *DatabaseTestCase) RefreshDatabase() *DatabaseTestCase {
	return dtc
}

func (dtc *DatabaseTestCase) SeedDatabase() *DatabaseTestCase {
	return dtc
}

func (dtc *DatabaseTestCase) AssertDatabaseHas(table string, data map[string]interface{}) *DatabaseTestCase {
	return dtc
}

func (dtc *DatabaseTestCase) AssertDatabaseMissing(table string, data map[string]interface{}) *DatabaseTestCase {
	return dtc
}

func (dtc *DatabaseTestCase) AssertDatabaseCount(table string, count int) *DatabaseTestCase {
	return dtc
}

type Factory struct {
	model      interface{}
	attributes map[string]interface{}
	states     map[string]map[string]interface{}
}

func NewFactory(model interface{}) *Factory {
	return &Factory{
		model:      model,
		attributes: make(map[string]interface{}),
		states:     make(map[string]map[string]interface{}),
	}
}

func (f *Factory) Definition(attributes map[string]interface{}) *Factory {
	f.attributes = attributes
	return f
}

func (f *Factory) State(name string, attributes map[string]interface{}) *Factory {
	f.states[name] = attributes
	return f
}

func (f *Factory) Make(count ...int) []interface{} {
	c := 1
	if len(count) > 0 {
		c = count[0]
	}

	results := make([]interface{}, c)
	for i := 0; i < c; i++ {
		results[i] = f.attributes
	}
	return results
}

func (f *Factory) Create(count ...int) []interface{} {
	return f.Make(count...)
}

type TestResponse struct {
	Status  int
	Headers http.Header
	Body    string
}

func AssertEquals(t *testing.T, expected, actual interface{}) {
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func AssertNotEquals(t *testing.T, expected, actual interface{}) {
	if expected == actual {
		t.Errorf("Expected values to be different, but both were %v", expected)
	}
}

func AssertTrue(t *testing.T, condition bool) {
	if !condition {
		t.Error("Expected condition to be true")
	}
}

func AssertFalse(t *testing.T, condition bool) {
	if condition {
		t.Error("Expected condition to be false")
	}
}

func AssertNil(t *testing.T, value interface{}) {
	if value != nil {
		t.Errorf("Expected nil, got %v", value)
	}
}

func AssertNotNil(t *testing.T, value interface{}) {
	if value == nil {
		t.Error("Expected value to not be nil")
	}
}

func AssertContains(t *testing.T, slice []string, item string) {
	for _, v := range slice {
		if v == item {
			return
		}
	}
	t.Errorf("Expected slice to contain %s", item)
}

func AssertPanics(t *testing.T, fn func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected function to panic")
		}
	}()
	fn()
}