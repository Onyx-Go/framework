package onyx

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// APIVersion represents an API version configuration
type APIVersion struct {
	Version      string                 `json:"version"`       // e.g., "v1", "v2", "1.0", "2.1"
	Name         string                 `json:"name"`         // e.g., "Version 1.0"
	Description  string                 `json:"description"`   // Version description
	Released     time.Time              `json:"released"`      // Release date
	Deprecated   bool                   `json:"deprecated"`    // Whether this version is deprecated
	DeprecatedAt *time.Time             `json:"deprecated_at"` // When it was deprecated
	EOLDate      *time.Time             `json:"eol_date"`      // End of life date
	Status       APIVersionStatus       `json:"status"`        // Current status
	Handler      *VersionHandler        `json:"-"`             // Version-specific handler
	Middleware   []MiddlewareFunc       `json:"-"`             // Version-specific middleware
	Config       map[string]interface{} `json:"config"`        // Version-specific configuration
}

// APIVersionStatus represents the status of an API version
type APIVersionStatus string

const (
	VersionStatusDevelopment APIVersionStatus = "development"
	VersionStatusStable      APIVersionStatus = "stable"
	VersionStatusDeprecated  APIVersionStatus = "deprecated"
	VersionStatusEOL         APIVersionStatus = "eol"
)

// VersionHandler handles version-specific routing
type VersionHandler struct {
	Version string
	Router  *Router
	Routes  map[string]*Route
}

// APIVersionManager manages API versions
type APIVersionManager struct {
	versions           map[string]*APIVersion
	defaultVersion     string
	headerName         string
	queryParamName     string
	pathPrefix         string
	pathPattern        *regexp.Regexp
	versionExtractor   VersionExtractor
	deprecationHandler DeprecationHandler
	mu                 sync.RWMutex
}

// VersionExtractor extracts version from request
type VersionExtractor interface {
	ExtractVersion(c Context) string
}

// DeprecationHandler handles deprecated version access
type DeprecationHandler interface {
	HandleDeprecated(c Context, version *APIVersion) error
}

// NewAPIVersionManager creates a new API version manager
func NewAPIVersionManager() *APIVersionManager {
	return &APIVersionManager{
		versions:       make(map[string]*APIVersion),
		headerName:     "API-Version",
		queryParamName: "version",
		pathPrefix:     "/api/",
		pathPattern:    regexp.MustCompile(`^/api/v(\d+(?:\.\d+)?)/`),
		versionExtractor: &DefaultVersionExtractor{},
		deprecationHandler: &DefaultDeprecationHandler{},
	}
}

// SetDefaultVersion sets the default API version
func (vm *APIVersionManager) SetDefaultVersion(version string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.defaultVersion = version
}

// SetHeaderName sets the version header name
func (vm *APIVersionManager) SetHeaderName(name string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.headerName = name
}

// SetQueryParamName sets the version query parameter name
func (vm *APIVersionManager) SetQueryParamName(name string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.queryParamName = name
}

// SetPathPrefix sets the API path prefix
func (vm *APIVersionManager) SetPathPrefix(prefix string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.pathPrefix = prefix
	// Update pattern based on new prefix
	pattern := strings.TrimSuffix(prefix, "/") + `/v(\d+(?:\.\d+)?)/`
	vm.pathPattern = regexp.MustCompile("^" + pattern)
}

// SetVersionExtractor sets a custom version extractor
func (vm *APIVersionManager) SetVersionExtractor(extractor VersionExtractor) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.versionExtractor = extractor
}

// SetDeprecationHandler sets a custom deprecation handler
func (vm *APIVersionManager) SetDeprecationHandler(handler DeprecationHandler) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.deprecationHandler = handler
}

// RegisterVersion registers a new API version
func (vm *APIVersionManager) RegisterVersion(version *APIVersion) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	
	if version.Version == "" {
		return fmt.Errorf("version string cannot be empty")
	}
	
	if _, exists := vm.versions[version.Version]; exists {
		return fmt.Errorf("version %s already registered", version.Version)
	}
	
	// Set default values
	if version.Status == "" {
		version.Status = VersionStatusStable
	}
	if version.Released.IsZero() {
		version.Released = time.Now()
	}
	
	// Create version handler
	version.Handler = &VersionHandler{
		Version: version.Version,
		Router:  NewRouter(),
		Routes:  make(map[string]*Route),
	}
	
	vm.versions[version.Version] = version
	
	// Set as default if it's the first version
	if vm.defaultVersion == "" {
		vm.defaultVersion = version.Version
	}
	
	return nil
}

// GetVersion gets a specific version
func (vm *APIVersionManager) GetVersion(version string) (*APIVersion, bool) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	v, exists := vm.versions[version]
	return v, exists
}

// GetAllVersions gets all registered versions
func (vm *APIVersionManager) GetAllVersions() map[string]*APIVersion {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	
	result := make(map[string]*APIVersion)
	for k, v := range vm.versions {
		result[k] = v
	}
	return result
}

// GetActiveVersions gets all non-EOL versions
func (vm *APIVersionManager) GetActiveVersions() map[string]*APIVersion {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	
	result := make(map[string]*APIVersion)
	for k, v := range vm.versions {
		if v.Status != VersionStatusEOL {
			result[k] = v
		}
	}
	return result
}

// DeprecateVersion marks a version as deprecated
func (vm *APIVersionManager) DeprecateVersion(version string, eolDate *time.Time) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	
	v, exists := vm.versions[version]
	if !exists {
		return fmt.Errorf("version %s not found", version)
	}
	
	now := time.Now()
	v.Deprecated = true
	v.DeprecatedAt = &now
	v.Status = VersionStatusDeprecated
	
	if eolDate != nil {
		v.EOLDate = eolDate
	}
	
	return nil
}

// EOLVersion marks a version as end-of-life
func (vm *APIVersionManager) EOLVersion(version string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	
	v, exists := vm.versions[version]
	if !exists {
		return fmt.Errorf("version %s not found", version)
	}
	
	v.Status = VersionStatusEOL
	if v.EOLDate == nil {
		now := time.Now()
		v.EOLDate = &now
	}
	
	return nil
}

// ExtractRequestVersion extracts version from request context
func (vm *APIVersionManager) ExtractRequestVersion(c Context) string {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	
	if vm.versionExtractor != nil {
		if version := vm.versionExtractor.ExtractVersion(c); version != "" {
			return version
		}
	}
	
	return vm.defaultVersion
}

// CreateVersionMiddleware creates middleware for version handling
func (vm *APIVersionManager) CreateVersionMiddleware() MiddlewareFunc {
	return func(c Context) error {
		// Extract version from request
		requestedVersion := vm.ExtractRequestVersion(c)
		
		// Get version info
		version, exists := vm.GetVersion(requestedVersion)
		if !exists {
			// Try to find the closest version
			if fallbackVersion := vm.findClosestVersion(requestedVersion); fallbackVersion != "" {
				version, exists = vm.GetVersion(fallbackVersion)
				if exists {
					c.SetHeader("API-Version-Used", fallbackVersion)
					c.SetHeader("API-Version-Requested", requestedVersion)
				}
			}
			
			if !exists {
				return NewHTTPError(400, fmt.Sprintf("Unsupported API version: %s", requestedVersion))
			}
		}
		
		// Check if version is EOL
		if version.Status == VersionStatusEOL {
			return NewHTTPError(410, fmt.Sprintf("API version %s is no longer supported", version.Version))
		}
		
		// Handle deprecated versions
		if version.Deprecated && vm.deprecationHandler != nil {
			if err := vm.deprecationHandler.HandleDeprecated(c, version); err != nil {
				return err
			}
		}
		
		// Set version info in context
		c.Set("api_version", version.Version)
		c.Set("api_version_info", version)
		
		// Set response headers
		c.SetHeader("API-Version", version.Version)
		if version.Deprecated {
			c.SetHeader("API-Deprecated", "true")
			if version.EOLDate != nil {
				c.SetHeader("API-EOL-Date", version.EOLDate.Format(time.RFC3339))
			}
		}
		
		return c.Next()
	}
}

// findClosestVersion finds the closest available version
func (vm *APIVersionManager) findClosestVersion(requested string) string {
	// Extract numeric version
	re := regexp.MustCompile(`v?(\d+)(?:\.(\d+))?`)
	matches := re.FindStringSubmatch(requested)
	if len(matches) < 2 {
		return vm.defaultVersion
	}
	
	major, _ := strconv.Atoi(matches[1])
	minor := 0
	if len(matches) > 2 && matches[2] != "" {
		minor, _ = strconv.Atoi(matches[2])
	}
	
	var closest string
	var closestDiff int = 999999
	
	for version := range vm.versions {
		vMatches := re.FindStringSubmatch(version)
		if len(vMatches) < 2 {
			continue
		}
		
		vMajor, _ := strconv.Atoi(vMatches[1])
		vMinor := 0
		if len(vMatches) > 2 && vMatches[2] != "" {
			vMinor, _ = strconv.Atoi(vMatches[2])
		}
		
		// Calculate difference
		diff := (major-vMajor)*100 + (minor - vMinor)
		if diff < 0 {
			diff = -diff
		}
		
		if diff < closestDiff {
			closestDiff = diff
			closest = version
		}
	}
	
	return closest
}

// VersionedRouter provides version-aware routing
type VersionedRouter struct {
	manager *APIVersionManager
	routers map[string]*Router
}

// NewVersionedRouter creates a new versioned router
func NewVersionedRouter(manager *APIVersionManager) *VersionedRouter {
	return &VersionedRouter{
		manager: manager,
		routers: make(map[string]*Router),
	}
}

// Version gets or creates a router for a specific version
func (vr *VersionedRouter) Version(version string) *Router {
	if router, exists := vr.routers[version]; exists {
		return router
	}
	
	router := NewRouter()
	vr.routers[version] = router
	return router
}

// Route routes request to the appropriate version
func (vr *VersionedRouter) Route(c *Context) error {
	version := vr.manager.ExtractRequestVersion(c)
	
	router, exists := vr.routers[version]
	if !exists {
		return NewHTTPError(404, "Version not found")
	}
	
	// Delegate to version-specific router
	router.ServeHTTP(c.ResponseWriter, c.Request)
	return nil
}

// Default version extractors

// DefaultVersionExtractor extracts version using multiple strategies
type DefaultVersionExtractor struct{}

// ExtractVersion extracts version from request
func (dve *DefaultVersionExtractor) ExtractVersion(c Context) string {
	// 1. Try path-based version (e.g., /api/v1/users)
	path := c.Request().URL.Path
	re := regexp.MustCompile(`^/api/v(\d+(?:\.\d+)?)/`)
	if matches := re.FindStringSubmatch(path); len(matches) > 1 {
		return "v" + matches[1]
	}
	
	// 2. Try header-based version
	if version := c.Header("API-Version"); version != "" {
		return version
	}
	
	// 3. Try query parameter
	if version := c.Query("version"); version != "" {
		return version
	}
	
	// 4. Try Accept header versioning (e.g., application/vnd.api+json;version=1)
	accept := c.Header("Accept")
	if accept != "" {
		re := regexp.MustCompile(`version=(\d+(?:\.\d+)?)`)
		if matches := re.FindStringSubmatch(accept); len(matches) > 1 {
			return "v" + matches[1]
		}
	}
	
	return ""
}

// HeaderVersionExtractor extracts version from header only
type HeaderVersionExtractor struct {
	HeaderName string
}

// ExtractVersion extracts version from header
func (hve *HeaderVersionExtractor) ExtractVersion(c Context) string {
	return c.Header(hve.HeaderName)
}

// PathVersionExtractor extracts version from path only
type PathVersionExtractor struct {
	Pattern *regexp.Regexp
}

// ExtractVersion extracts version from path
func (pve *PathVersionExtractor) ExtractVersion(c Context) string {
	path := c.Request().URL.Path
	if matches := pve.Pattern.FindStringSubmatch(path); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// Default deprecation handlers

// DefaultDeprecationHandler provides default deprecation handling
type DefaultDeprecationHandler struct{}

// HandleDeprecated handles deprecated version access
func (ddh *DefaultDeprecationHandler) HandleDeprecated(c Context, version *APIVersion) error {
	// Set deprecation headers
	c.SetHeader("API-Deprecated", "true")
	c.SetHeader("API-Deprecation-Date", version.DeprecatedAt.Format(time.RFC3339))
	
	if version.EOLDate != nil {
		c.SetHeader("API-EOL-Date", version.EOLDate.Format(time.RFC3339))
		
		// Add warning if EOL is soon
		daysUntilEOL := time.Until(*version.EOLDate).Hours() / 24
		if daysUntilEOL <= 30 {
			warning := fmt.Sprintf("API version %s will be discontinued in %.0f days", version.Version, daysUntilEOL)
			c.SetHeader("Warning", warning)
		}
	}
	
	// Log deprecation warning
	if logger := Log(); logger != nil {
		logger.Warn("Deprecated API version accessed", map[string]interface{}{
			"version":    version.Version,
			"user_agent": c.Header("User-Agent"),
			"remote_ip":  c.RemoteIP(),
			"path":       c.Request().URL.Path,
		})
	}
	
	return nil
}

// StrictDeprecationHandler blocks access to deprecated versions
type StrictDeprecationHandler struct{}

// HandleDeprecated blocks deprecated version access
func (sdh *StrictDeprecationHandler) HandleDeprecated(c Context, version *APIVersion) error {
	return NewHTTPError(410, fmt.Sprintf("API version %s is deprecated and no longer available", version.Version))
}

// LoggingDeprecationHandler logs deprecation access
type LoggingDeprecationHandler struct {
	AllowAccess bool
}

// HandleDeprecated logs and optionally blocks deprecated version access
func (ldh *LoggingDeprecationHandler) HandleDeprecated(c Context, version *APIVersion) error {
	// Always log
	if logger := Log(); logger != nil {
		logger.Warn("Deprecated API version accessed", map[string]interface{}{
			"version":     version.Version,
			"user_agent":  c.Header("User-Agent"),
			"remote_ip":   c.RemoteIP(),
			"path":        c.Request().URL.Path,
			"deprecated_at": version.DeprecatedAt,
			"eol_date":    version.EOLDate,
		})
	}
	
	// Block if not allowed
	if !ldh.AllowAccess {
		return NewHTTPError(410, fmt.Sprintf("API version %s is deprecated", version.Version))
	}
	
	// Set headers
	c.SetHeader("API-Deprecated", "true")
	if version.DeprecatedAt != nil {
		c.SetHeader("API-Deprecation-Date", version.DeprecatedAt.Format(time.RFC3339))
	}
	if version.EOLDate != nil {
		c.SetHeader("API-EOL-Date", version.EOLDate.Format(time.RFC3339))
	}
	
	return nil
}

// Version-aware route group

// VersionedRouteGroup provides version-aware route grouping
type VersionedRouteGroup struct {
	manager    *APIVersionManager
	version    string
	baseGroup  *RouteGroup
	middleware []MiddlewareFunc
}

// NewVersionedRouteGroup creates a versioned route group
func NewVersionedRouteGroup(manager *APIVersionManager, version string, router *Router) *VersionedRouteGroup {
	baseGroup := router.Group(fmt.Sprintf("/api/%s", version))
	
	return &VersionedRouteGroup{
		manager:   manager,
		version:   version,
		baseGroup: baseGroup,
	}
}

// Use adds middleware to the versioned group
func (vg *VersionedRouteGroup) Use(middleware ...MiddlewareFunc) {
	vg.middleware = append(vg.middleware, middleware...)
	vg.baseGroup.router.Use(middleware...)
}

// Get adds a GET route to the versioned group
func (vg *VersionedRouteGroup) Get(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	allMiddleware := append(vg.middleware, middleware...)
	vg.baseGroup.Get(pattern, handler, allMiddleware...)
}

// Post adds a POST route to the versioned group
func (vg *VersionedRouteGroup) Post(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	allMiddleware := append(vg.middleware, middleware...)
	vg.baseGroup.Post(pattern, handler, allMiddleware...)
}

// Put adds a PUT route to the versioned group
func (vg *VersionedRouteGroup) Put(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	allMiddleware := append(vg.middleware, middleware...)
	vg.baseGroup.Put(pattern, handler, allMiddleware...)
}

// Delete adds a DELETE route to the versioned group
func (vg *VersionedRouteGroup) Delete(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	allMiddleware := append(vg.middleware, middleware...)
	vg.baseGroup.Delete(pattern, handler, allMiddleware...)
}

// Patch adds a PATCH route to the versioned group
func (vg *VersionedRouteGroup) Patch(pattern string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	allMiddleware := append(vg.middleware, middleware...)
	vg.baseGroup.Patch(pattern, handler, allMiddleware...)
}

// Utility functions

// CreateAPIVersionConfig creates a standard API version configuration
func CreateAPIVersionConfig(version, name, description string) *APIVersion {
	return &APIVersion{
		Version:     version,
		Name:        name,
		Description: description,
		Released:    time.Now(),
		Status:      VersionStatusStable,
		Config:      make(map[string]interface{}),
	}
}

// CreateDeprecatedAPIVersionConfig creates a deprecated API version configuration
func CreateDeprecatedAPIVersionConfig(version, name, description string, eolDate time.Time) *APIVersion {
	now := time.Now()
	return &APIVersion{
		Version:      version,
		Name:         name,
		Description:  description,
		Released:     time.Now(),
		Status:       VersionStatusDeprecated,
		Deprecated:   true,
		DeprecatedAt: &now,
		EOLDate:      &eolDate,
		Config:       make(map[string]interface{}),
	}
}

// GetVersionFromContext extracts the API version from context
func GetVersionFromContext(c Context) string {
	if version, exists := c.Get("api_version"); exists {
		if v, ok := version.(string); ok {
			return v
		}
	}
	return ""
}

// GetVersionInfoFromContext extracts the API version info from context
func GetVersionInfoFromContext(c Context) *APIVersion {
	if versionInfo, exists := c.Get("api_version_info"); exists {
		if v, ok := versionInfo.(*APIVersion); ok {
			return v
		}
	}
	return nil
}

// IsVersionDeprecated checks if the current request version is deprecated
func IsVersionDeprecated(c *Context) bool {
	if versionInfo := GetVersionInfoFromContext(c); versionInfo != nil {
		return versionInfo.Deprecated
	}
	return false
}

// GetVersionCompatibilityMatrix returns compatibility information between versions
func (vm *APIVersionManager) GetVersionCompatibilityMatrix() map[string]map[string]bool {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	
	matrix := make(map[string]map[string]bool)
	
	for v1 := range vm.versions {
		matrix[v1] = make(map[string]bool)
		for v2 := range vm.versions {
			// Simple compatibility: same major version
			matrix[v1][v2] = vm.isCompatible(v1, v2)
		}
	}
	
	return matrix
}

// isCompatible checks if two versions are compatible
func (vm *APIVersionManager) isCompatible(v1, v2 string) bool {
	// Extract major versions
	re := regexp.MustCompile(`v?(\d+)`)
	
	matches1 := re.FindStringSubmatch(v1)
	matches2 := re.FindStringSubmatch(v2)
	
	if len(matches1) < 2 || len(matches2) < 2 {
		return v1 == v2
	}
	
	major1, _ := strconv.Atoi(matches1[1])
	major2, _ := strconv.Atoi(matches2[1])
	
	return major1 == major2
}