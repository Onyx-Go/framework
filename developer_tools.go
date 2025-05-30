package onyx

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// APIPlayground provides interactive API testing capabilities
type APIPlayground struct {
	requestHistory []*PlaygroundRequest
	maxHistory     int
}

// PlaygroundRequest represents a playground request
type PlaygroundRequest struct {
	ID          string                 `json:"id"`
	Method      string                 `json:"method"`
	URL         string                 `json:"url"`
	Headers     map[string]string      `json:"headers"`
	Body        string                 `json:"body"`
	Response    *PlaygroundResponse    `json:"response,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Duration    time.Duration          `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PlaygroundResponse represents a playground response
type PlaygroundResponse struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       string              `json:"body"`
	Size       int64               `json:"size"`
	Duration   time.Duration       `json:"duration"`
}

// NewAPIPlayground creates a new API playground
func NewAPIPlayground() *APIPlayground {
	return &APIPlayground{
		requestHistory: make([]*PlaygroundRequest, 0),
		maxHistory:     100,
	}
}

// HandleRequest handles a playground API request
func (ap *APIPlayground) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req struct {
		Method  string            `json:"method"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Execute the request
	playgroundReq := &PlaygroundRequest{
		ID:        fmt.Sprintf("req_%d", time.Now().UnixNano()),
		Method:    req.Method,
		URL:       req.URL,
		Headers:   req.Headers,
		Body:      req.Body,
		Timestamp: time.Now(),
	}

	response, err := ap.executeRequest(playgroundReq)
	if err != nil {
		playgroundReq.Response = &PlaygroundResponse{
			StatusCode: 0,
			Body:       fmt.Sprintf("Request failed: %v", err),
			Duration:   time.Since(playgroundReq.Timestamp),
		}
	} else {
		playgroundReq.Response = response
	}

	playgroundReq.Duration = time.Since(playgroundReq.Timestamp)

	// Add to history
	ap.addToHistory(playgroundReq)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(playgroundReq)
}

// executeRequest executes an HTTP request
func (ap *APIPlayground) executeRequest(playgroundReq *PlaygroundRequest) (*PlaygroundResponse, error) {
	startTime := time.Now()

	// Create HTTP request
	var bodyReader io.Reader
	if playgroundReq.Body != "" {
		bodyReader = strings.NewReader(playgroundReq.Body)
	}

	req, err := http.NewRequest(playgroundReq.Method, playgroundReq.URL, bodyReader)
	if err != nil {
		return nil, err
	}

	// Add headers
	for key, value := range playgroundReq.Headers {
		req.Header.Set(key, value)
	}

	// Execute request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	duration := time.Since(startTime)

	return &PlaygroundResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       string(body),
		Size:       int64(len(body)),
		Duration:   duration,
	}, nil
}

// addToHistory adds a request to the history
func (ap *APIPlayground) addToHistory(req *PlaygroundRequest) {
	ap.requestHistory = append(ap.requestHistory, req)

	// Trim history if too long
	if len(ap.requestHistory) > ap.maxHistory {
		ap.requestHistory = ap.requestHistory[1:]
	}
}

// GetHistory returns the request history
func (ap *APIPlayground) GetHistory() []*PlaygroundRequest {
	return ap.requestHistory
}

// CodeGenerator generates client code for APIs
type CodeGenerator struct {
	supportedLanguages map[string]*Language
}

// Language represents a supported programming language
type Language struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Extension   string `json:"extension"`
	Generator   CodeGeneratorFunc
}

// CodeGeneratorFunc generates code for a specific language
type CodeGeneratorFunc func(spec *OpenAPISpec, options *GenerationOptions) (*GeneratedCode, error)

// GenerationOptions options for code generation
type GenerationOptions struct {
	PackageName    string            `json:"package_name"`
	Namespace      string            `json:"namespace"`
	OutputFormat   string            `json:"output_format"` // client, examples, models
	IncludeTests   bool              `json:"include_tests"`
	IncludeDocs    bool              `json:"include_docs"`
	CustomOptions  map[string]interface{} `json:"custom_options"`
}

// GeneratedCode represents generated code
type GeneratedCode struct {
	Language    string            `json:"language"`
	Type        string            `json:"type"`
	Files       map[string]string `json:"files"`
	PackageName string            `json:"package_name"`
	Instructions string           `json:"instructions"`
	Dependencies []string         `json:"dependencies"`
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator() *CodeGenerator {
	cg := &CodeGenerator{
		supportedLanguages: make(map[string]*Language),
	}

	// Register supported languages
	cg.registerDefaultLanguages()

	return cg
}

// registerDefaultLanguages registers default supported languages
func (cg *CodeGenerator) registerDefaultLanguages() {
	// JavaScript/TypeScript
	cg.supportedLanguages["javascript"] = &Language{
		ID:          "javascript",
		Name:        "JavaScript",
		Description: "JavaScript/Node.js client library",
		Extension:   "js",
		Generator:   cg.generateJavaScript,
	}

	// Python
	cg.supportedLanguages["python"] = &Language{
		ID:          "python",
		Name:        "Python",
		Description: "Python client library with requests",
		Extension:   "py",
		Generator:   cg.generatePython,
	}

	// Go
	cg.supportedLanguages["go"] = &Language{
		ID:          "go",
		Name:        "Go",
		Description: "Go client library with net/http",
		Extension:   "go",
		Generator:   cg.generateGo,
	}

	// Java
	cg.supportedLanguages["java"] = &Language{
		ID:          "java",
		Name:        "Java",
		Description: "Java client library with OkHttp",
		Extension:   "java",
		Generator:   cg.generateJava,
	}

	// C#
	cg.supportedLanguages["csharp"] = &Language{
		ID:          "csharp",
		Name:        "C#",
		Description: "C# client library with HttpClient",
		Extension:   "cs",
		Generator:   cg.generateCSharp,
	}

	// PHP
	cg.supportedLanguages["php"] = &Language{
		ID:          "php",
		Name:        "PHP",
		Description: "PHP client library with Guzzle",
		Extension:   "php",
		Generator:   cg.generatePHP,
	}
}

// GetSupportedLanguages returns supported languages
func (cg *CodeGenerator) GetSupportedLanguages() map[string]*Language {
	return cg.supportedLanguages
}

// HandleRequest handles code generation requests
func (cg *CodeGenerator) HandleRequest(w http.ResponseWriter, r *http.Request, docManager *APIDocumentationManager) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req struct {
		Language    string                 `json:"language"`
		SpecURL     string                 `json:"spec_url"`
		Options     *GenerationOptions     `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get language
	language, exists := cg.supportedLanguages[req.Language]
	if !exists {
		http.Error(w, "Unsupported language", http.StatusBadRequest)
		return
	}

	// Get OpenAPI spec
	var spec *OpenAPISpec
	var err error

	if req.SpecURL != "" {
		// Fetch spec from URL
		spec, err = cg.fetchSpecFromURL(req.SpecURL)
	} else if docManager != nil {
		// Use current spec
		spec, err = docManager.GenerateDocumentation()
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get OpenAPI spec: %v", err), http.StatusInternalServerError)
		return
	}

	// Set default options
	if req.Options == nil {
		req.Options = &GenerationOptions{
			PackageName:  "api-client",
			OutputFormat: "client",
			IncludeTests: true,
			IncludeDocs:  true,
		}
	}

	// Generate code
	generatedCode, err := language.Generator(spec, req.Options)
	if err != nil {
		http.Error(w, fmt.Sprintf("Code generation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return generated code
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(generatedCode)
}

// fetchSpecFromURL fetches OpenAPI spec from URL
func (cg *CodeGenerator) fetchSpecFromURL(url string) (*OpenAPISpec, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var spec OpenAPISpec
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		return nil, err
	}

	return &spec, nil
}

// Language-specific generators

// generateJavaScript generates JavaScript client code
func (cg *CodeGenerator) generateJavaScript(spec *OpenAPISpec, options *GenerationOptions) (*GeneratedCode, error) {
	files := make(map[string]string)

	switch options.OutputFormat {
	case "client":
		files["index.js"] = cg.generateJSClient(spec, options)
		files["package.json"] = cg.generateJSPackageJSON(options)
	case "examples":
		files["examples.js"] = cg.generateJSExamples(spec, options)
	case "models":
		files["models.js"] = cg.generateJSModels(spec, options)
	}

	return &GeneratedCode{
		Language:     "javascript",
		Type:         options.OutputFormat,
		Files:        files,
		PackageName:  options.PackageName,
		Instructions: "Install dependencies with 'npm install', then import and use the client library.",
		Dependencies: []string{"axios"},
	}, nil
}

// generatePython generates Python client code
func (cg *CodeGenerator) generatePython(spec *OpenAPISpec, options *GenerationOptions) (*GeneratedCode, error) {
	files := make(map[string]string)

	switch options.OutputFormat {
	case "client":
		files["client.py"] = cg.generatePythonClient(spec, options)
		files["setup.py"] = cg.generatePythonSetup(options)
	case "examples":
		files["examples.py"] = cg.generatePythonExamples(spec, options)
	case "models":
		files["models.py"] = cg.generatePythonModels(spec, options)
	}

	return &GeneratedCode{
		Language:     "python",
		Type:         options.OutputFormat,
		Files:        files,
		PackageName:  options.PackageName,
		Instructions: "Install dependencies with 'pip install requests', then import and use the client library.",
		Dependencies: []string{"requests", "typing"},
	}, nil
}

// generateGo generates Go client code
func (cg *CodeGenerator) generateGo(spec *OpenAPISpec, options *GenerationOptions) (*GeneratedCode, error) {
	files := make(map[string]string)

	switch options.OutputFormat {
	case "client":
		files["client.go"] = cg.generateGoClient(spec, options)
		files["go.mod"] = cg.generateGoMod(options)
	case "examples":
		files["examples.go"] = cg.generateGoExamples(spec, options)
	case "models":
		files["models.go"] = cg.generateGoModels(spec, options)
	}

	return &GeneratedCode{
		Language:     "go",
		Type:         options.OutputFormat,
		Files:        files,
		PackageName:  options.PackageName,
		Instructions: "Run 'go mod tidy' to install dependencies, then import and use the client package.",
		Dependencies: []string{},
	}, nil
}

// generateJava generates Java client code
func (cg *CodeGenerator) generateJava(spec *OpenAPISpec, options *GenerationOptions) (*GeneratedCode, error) {
	files := make(map[string]string)

	switch options.OutputFormat {
	case "client":
		files["ApiClient.java"] = cg.generateJavaClient(spec, options)
		files["pom.xml"] = cg.generateJavaPom(options)
	case "examples":
		files["Examples.java"] = cg.generateJavaExamples(spec, options)
	case "models":
		files["Models.java"] = cg.generateJavaModels(spec, options)
	}

	return &GeneratedCode{
		Language:     "java",
		Type:         options.OutputFormat,
		Files:        files,
		PackageName:  options.PackageName,
		Instructions: "Build with Maven using 'mvn compile', then use the generated client classes.",
		Dependencies: []string{"okhttp", "gson"},
	}, nil
}

// generateCSharp generates C# client code
func (cg *CodeGenerator) generateCSharp(spec *OpenAPISpec, options *GenerationOptions) (*GeneratedCode, error) {
	files := make(map[string]string)

	switch options.OutputFormat {
	case "client":
		files["ApiClient.cs"] = cg.generateCSharpClient(spec, options)
		files[options.PackageName+".csproj"] = cg.generateCSharpProject(options)
	case "examples":
		files["Examples.cs"] = cg.generateCSharpExamples(spec, options)
	case "models":
		files["Models.cs"] = cg.generateCSharpModels(spec, options)
	}

	return &GeneratedCode{
		Language:     "csharp",
		Type:         options.OutputFormat,
		Files:        files,
		PackageName:  options.PackageName,
		Instructions: "Build with 'dotnet build', then reference the generated library.",
		Dependencies: []string{"Newtonsoft.Json"},
	}, nil
}

// generatePHP generates PHP client code
func (cg *CodeGenerator) generatePHP(spec *OpenAPISpec, options *GenerationOptions) (*GeneratedCode, error) {
	files := make(map[string]string)

	switch options.OutputFormat {
	case "client":
		files["ApiClient.php"] = cg.generatePHPClient(spec, options)
		files["composer.json"] = cg.generatePHPComposer(options)
	case "examples":
		files["examples.php"] = cg.generatePHPExamples(spec, options)
	case "models":
		files["Models.php"] = cg.generatePHPModels(spec, options)
	}

	return &GeneratedCode{
		Language:     "php",
		Type:         options.OutputFormat,
		Files:        files,
		PackageName:  options.PackageName,
		Instructions: "Install dependencies with 'composer install', then require and use the client.",
		Dependencies: []string{"guzzlehttp/guzzle"},
	}, nil
}

// Template generation methods (simplified implementations)

func (cg *CodeGenerator) generateJSClient(spec *OpenAPISpec, options *GenerationOptions) string {
	return fmt.Sprintf(`// %s - JavaScript API Client
class ApiClient {
    constructor(baseURL, apiKey) {
        this.baseURL = baseURL;
        this.apiKey = apiKey;
        this.axios = require('axios').create({
            baseURL: this.baseURL,
            headers: {
                'Authorization': 'Bearer ' + this.apiKey,
                'Content-Type': 'application/json'
            }
        });
    }

    async request(method, path, data = null) {
        try {
            const response = await this.axios.request({
                method,
                url: path,
                data
            });
            return response.data;
        } catch (error) {
            throw new Error('API request failed: ' + error.message);
        }
    }

    // Generated methods for each endpoint would go here
}

module.exports = ApiClient;`, options.PackageName)
}

func (cg *CodeGenerator) generateJSPackageJSON(options *GenerationOptions) string {
	return fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "description": "Generated API client",
  "main": "index.js",
  "dependencies": {
    "axios": "^0.27.0"
  },
  "author": "Generated by Onyx",
  "license": "MIT"
}`, options.PackageName)
}

func (cg *CodeGenerator) generateJSExamples(spec *OpenAPISpec, options *GenerationOptions) string {
	return fmt.Sprintf(`// %s - Usage Examples
const ApiClient = require('./index');

const client = new ApiClient('https://api.example.com', 'your-api-key');

async function examples() {
    try {
        // Example API calls would be generated here based on the OpenAPI spec
        console.log('API client ready');
    } catch (error) {
        console.error('Error:', error);
    }
}

examples();`, options.PackageName)
}

func (cg *CodeGenerator) generateJSModels(spec *OpenAPISpec, options *GenerationOptions) string {
	return fmt.Sprintf(`// %s - Data Models
// TypeScript interfaces/types would be generated here based on OpenAPI schemas

module.exports = {
    // Export generated models
};`, options.PackageName)
}

func (cg *CodeGenerator) generatePythonClient(spec *OpenAPISpec, options *GenerationOptions) string {
	return fmt.Sprintf(`# %s - Python API Client
import requests
from typing import Dict, Any, Optional

class ApiClient:
    def __init__(self, base_url: str, api_key: str):
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.session = requests.Session()
        self.session.headers.update({
            'Authorization': f'Bearer {api_key}',
            'Content-Type': 'application/json'
        })

    def request(self, method: str, path: str, data: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        url = f'{self.base_url}{path}'
        response = self.session.request(method, url, json=data)
        response.raise_for_status()
        return response.json()

    # Generated methods for each endpoint would go here
`, strings.ReplaceAll(options.PackageName, "-", "_"))
}

func (cg *CodeGenerator) generatePythonSetup(options *GenerationOptions) string {
	return fmt.Sprintf(`from setuptools import setup, find_packages

setup(
    name="%s",
    version="1.0.0",
    description="Generated API client",
    packages=find_packages(),
    install_requires=[
        "requests>=2.25.0",
    ],
    author="Generated by Onyx",
    license="MIT",
)`, options.PackageName)
}

func (cg *CodeGenerator) generatePythonExamples(spec *OpenAPISpec, options *GenerationOptions) string {
	return fmt.Sprintf(`# %s - Usage Examples
from client import ApiClient

client = ApiClient('https://api.example.com', 'your-api-key')

def examples():
    try:
        # Example API calls would be generated here based on the OpenAPI spec
        print('API client ready')
    except Exception as error:
        print(f'Error: {error}')

if __name__ == '__main__':
    examples()`, strings.ReplaceAll(options.PackageName, "-", "_"))
}

func (cg *CodeGenerator) generatePythonModels(spec *OpenAPISpec, options *GenerationOptions) string {
	return fmt.Sprintf(`# %s - Data Models
from dataclasses import dataclass
from typing import Optional, List
from datetime import datetime

# Generated dataclasses based on OpenAPI schemas would go here
`, strings.ReplaceAll(options.PackageName, "-", "_"))
}

func (cg *CodeGenerator) generateGoClient(spec *OpenAPISpec, options *GenerationOptions) string {
	return fmt.Sprintf(`// Package %s provides an API client
package %s

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client represents the API client
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Request makes an HTTP request to the API
func (c *Client) Request(method, path string, data interface{}) (*http.Response, error) {
	url := c.BaseURL + path
	
	var body *bytes.Buffer
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonData)
	}
	
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	
	return c.HTTPClient.Do(req)
}

// Generated methods for each endpoint would go here
`, strings.ReplaceAll(options.PackageName, "-", ""), strings.ReplaceAll(options.PackageName, "-", ""))
}

func (cg *CodeGenerator) generateGoMod(options *GenerationOptions) string {
	return fmt.Sprintf(`module %s

go 1.19

// No external dependencies required for basic HTTP client
`, strings.ReplaceAll(options.PackageName, "-", ""))
}

func (cg *CodeGenerator) generateGoExamples(spec *OpenAPISpec, options *GenerationOptions) string {
	return fmt.Sprintf(`// Package main demonstrates usage of %s
package main

import (
	"fmt"
	"log"
	
	"%s"
)

func main() {
	client := %s.NewClient("https://api.example.com", "your-api-key")
	
	// Example API calls would be generated here based on the OpenAPI spec
	fmt.Println("API client ready")
}`, options.PackageName, options.PackageName, strings.ReplaceAll(options.PackageName, "-", ""))
}

func (cg *CodeGenerator) generateGoModels(spec *OpenAPISpec, options *GenerationOptions) string {
	return fmt.Sprintf(`// Package %s contains data models
package %s

import "time"

// Generated structs based on OpenAPI schemas would go here
`, strings.ReplaceAll(options.PackageName, "-", ""), strings.ReplaceAll(options.PackageName, "-", ""))
}

// Java, C#, and PHP generators would follow similar patterns...
// (Simplified implementations for brevity)

func (cg *CodeGenerator) generateJavaClient(spec *OpenAPISpec, options *GenerationOptions) string {
	return fmt.Sprintf(`// %s - Java API Client
public class ApiClient {
    // Java implementation would go here
}`, options.PackageName)
}

func (cg *CodeGenerator) generateJavaPom(options *GenerationOptions) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>%s</artifactId>
    <version>1.0.0</version>
</project>`, options.PackageName)
}

func (cg *CodeGenerator) generateJavaExamples(spec *OpenAPISpec, options *GenerationOptions) string {
	return "// Java examples implementation"
}

func (cg *CodeGenerator) generateJavaModels(spec *OpenAPISpec, options *GenerationOptions) string {
	return "// Java models implementation"
}

func (cg *CodeGenerator) generateCSharpClient(spec *OpenAPISpec, options *GenerationOptions) string {
	return "// C# client implementation"
}

func (cg *CodeGenerator) generateCSharpProject(options *GenerationOptions) string {
	return fmt.Sprintf(`<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net6.0</TargetFramework>
  </PropertyGroup>
</Project>`)
}

func (cg *CodeGenerator) generateCSharpExamples(spec *OpenAPISpec, options *GenerationOptions) string {
	return "// C# examples implementation"
}

func (cg *CodeGenerator) generateCSharpModels(spec *OpenAPISpec, options *GenerationOptions) string {
	return "// C# models implementation"
}

func (cg *CodeGenerator) generatePHPClient(spec *OpenAPISpec, options *GenerationOptions) string {
	return "<?php\n// PHP client implementation"
}

func (cg *CodeGenerator) generatePHPComposer(options *GenerationOptions) string {
	return fmt.Sprintf(`{
    "name": "%s",
    "version": "1.0.0",
    "require": {
        "guzzlehttp/guzzle": "^7.0"
    }
}`, options.PackageName)
}

func (cg *CodeGenerator) generatePHPExamples(spec *OpenAPISpec, options *GenerationOptions) string {
	return "<?php\n// PHP examples implementation"
}

func (cg *CodeGenerator) generatePHPModels(spec *OpenAPISpec, options *GenerationOptions) string {
	return "<?php\n// PHP models implementation"
}