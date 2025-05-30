package onyx

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// VersionedDocumentation manages documentation across API versions
type VersionedDocumentation struct {
	versionManager *APIVersionManager
	builders       map[string]*APIDocumentationBuilder
	config         *VersionedDocsConfig
	mu             sync.RWMutex
}

// VersionedDocsConfig configuration for versioned documentation
type VersionedDocsConfig struct {
	BaseTitle       string `json:"base_title"`
	BaseDescription string `json:"base_description"`
	IncludeVersion  bool   `json:"include_version_in_title"`
	ShowDeprecated  bool   `json:"show_deprecated"`
	ShowEOL         bool   `json:"show_eol"`
	DefaultVersion  string `json:"default_version"`
}

// VersionedSpec represents a version-specific OpenAPI specification
type VersionedSpec struct {
	Version string       `json:"version"`
	Spec    *OpenAPISpec `json:"spec"`
	Status  APIVersionStatus `json:"status"`
	Meta    *VersionMeta `json:"meta"`
}

// VersionMeta contains metadata about a version
type VersionMeta struct {
	Released     time.Time  `json:"released"`
	Deprecated   bool       `json:"deprecated"`
	DeprecatedAt *time.Time `json:"deprecated_at,omitempty"`
	EOLDate      *time.Time `json:"eol_date,omitempty"`
	Changes      []string   `json:"changes,omitempty"`
	Breaking     []string   `json:"breaking_changes,omitempty"`
}

// NewVersionedDocumentation creates a new versioned documentation manager
func NewVersionedDocumentation(versionManager *APIVersionManager, config *VersionedDocsConfig) *VersionedDocumentation {
	if config == nil {
		config = &VersionedDocsConfig{
			BaseTitle:       "API Documentation",
			BaseDescription: "REST API Documentation",
			IncludeVersion:  true,
			ShowDeprecated:  true,
			ShowEOL:         false,
		}
	}

	return &VersionedDocumentation{
		versionManager: versionManager,
		builders:       make(map[string]*APIDocumentationBuilder),
		config:         config,
	}
}

// GetBuilder gets or creates a documentation builder for a version
func (vd *VersionedDocumentation) GetBuilder(version string) *APIDocumentationBuilder {
	vd.mu.Lock()
	defer vd.mu.Unlock()

	if builder, exists := vd.builders[version]; exists {
		return builder
	}

	// Get version info
	versionInfo, exists := vd.versionManager.GetVersion(version)
	if !exists {
		return nil
	}

	// Create version-specific config
	config := &APIDocConfig{
		Title:       vd.buildVersionTitle(version, versionInfo),
		Description: vd.buildVersionDescription(version, versionInfo),
		Version:     version,
	}

	builder := NewAPIDocumentationBuilder(config)
	vd.builders[version] = builder

	return builder
}

// buildVersionTitle builds a version-specific title
func (vd *VersionedDocumentation) buildVersionTitle(version string, versionInfo *APIVersion) string {
	title := vd.config.BaseTitle
	
	if vd.config.IncludeVersion {
		if versionInfo.Name != "" {
			title = fmt.Sprintf("%s - %s", title, versionInfo.Name)
		} else {
			title = fmt.Sprintf("%s - %s", title, version)
		}
	}
	
	if versionInfo.Deprecated {
		title += " (Deprecated)"
	}
	
	if versionInfo.Status == VersionStatusEOL {
		title += " (End of Life)"
	}
	
	return title
}

// buildVersionDescription builds a version-specific description
func (vd *VersionedDocumentation) buildVersionDescription(version string, versionInfo *APIVersion) string {
	description := vd.config.BaseDescription
	
	if versionInfo.Description != "" {
		description = fmt.Sprintf("%s\n\n%s", description, versionInfo.Description)
	}
	
	// Add version-specific notices
	var notices []string
	
	if versionInfo.Deprecated {
		notice := fmt.Sprintf("⚠️ **This version (%s) is deprecated**", version)
		if versionInfo.EOLDate != nil {
			notice += fmt.Sprintf(" and will be discontinued on %s", versionInfo.EOLDate.Format("2006-01-02"))
		}
		notices = append(notices, notice)
	}
	
	if versionInfo.Status == VersionStatusEOL {
		notices = append(notices, fmt.Sprintf("🚫 **This version (%s) is no longer supported**", version))
	}
	
	if len(notices) > 0 {
		description = fmt.Sprintf("%s\n\n%s", description, strings.Join(notices, "\n\n"))
	}
	
	return description
}

// GenerateVersionedSpecs generates OpenAPI specs for all versions
func (vd *VersionedDocumentation) GenerateVersionedSpecs() (map[string]*VersionedSpec, error) {
	vd.mu.RLock()
	defer vd.mu.RUnlock()

	specs := make(map[string]*VersionedSpec)
	versions := vd.versionManager.GetAllVersions()

	for version, versionInfo := range versions {
		// Skip EOL versions if configured
		if !vd.config.ShowEOL && versionInfo.Status == VersionStatusEOL {
			continue
		}

		// Skip deprecated versions if configured
		if !vd.config.ShowDeprecated && versionInfo.Deprecated {
			continue
		}

		builder := vd.GetBuilder(version)
		if builder == nil {
			continue
		}

		spec, err := builder.GenerateOpenAPISpec()
		if err != nil {
			return nil, fmt.Errorf("failed to generate spec for version %s: %w", version, err)
		}

		// Add version-specific information to spec
		vd.enhanceSpecWithVersionInfo(spec, versionInfo)

		specs[version] = &VersionedSpec{
			Version: version,
			Spec:    spec,
			Status:  versionInfo.Status,
			Meta: &VersionMeta{
				Released:     versionInfo.Released,
				Deprecated:   versionInfo.Deprecated,
				DeprecatedAt: versionInfo.DeprecatedAt,
				EOLDate:      versionInfo.EOLDate,
			},
		}
	}

	return specs, nil
}

// enhanceSpecWithVersionInfo adds version-specific information to the OpenAPI spec
func (vd *VersionedDocumentation) enhanceSpecWithVersionInfo(spec *OpenAPISpec, versionInfo *APIVersion) {
	// Add version info to servers
	for i := range spec.Servers {
		if !strings.Contains(spec.Servers[i].URL, versionInfo.Version) {
			spec.Servers[i].URL = strings.Replace(spec.Servers[i].URL, "/api/", fmt.Sprintf("/api/%s/", versionInfo.Version), 1)
		}
	}

	// Add deprecation info to spec info
	if versionInfo.Deprecated {
		spec.Info.Description += "\n\n**⚠️ This API version is deprecated.**"
		if versionInfo.EOLDate != nil {
			spec.Info.Description += fmt.Sprintf(" It will be discontinued on %s.", versionInfo.EOLDate.Format("2006-01-02"))
		}
	}

	// Add custom extensions
	if spec.Info.Title != "" {
		// Add version status as extension
		if spec.Components == nil {
			spec.Components = &Components{}
		}
	}
}

// GenerateVersionMatrix generates a version compatibility matrix
func (vd *VersionedDocumentation) GenerateVersionMatrix() *VersionMatrix {
	versions := vd.versionManager.GetAllVersions()
	matrix := &VersionMatrix{
		Versions: make([]*VersionInfo, 0, len(versions)),
		Matrix:   vd.versionManager.GetVersionCompatibilityMatrix(),
	}

	// Convert to sorted slice
	for _, version := range versions {
		matrix.Versions = append(matrix.Versions, &VersionInfo{
			Version:      version.Version,
			Name:         version.Name,
			Description:  version.Description,
			Status:       version.Status,
			Released:     version.Released,
			Deprecated:   version.Deprecated,
			DeprecatedAt: version.DeprecatedAt,
			EOLDate:      version.EOLDate,
		})
	}

	// Sort by version
	sort.Slice(matrix.Versions, func(i, j int) bool {
		return vd.compareVersions(matrix.Versions[i].Version, matrix.Versions[j].Version) < 0
	})

	return matrix
}

// VersionMatrix represents version compatibility information
type VersionMatrix struct {
	Versions []*VersionInfo           `json:"versions"`
	Matrix   map[string]map[string]bool `json:"compatibility_matrix"`
}

// VersionInfo contains basic version information
type VersionInfo struct {
	Version      string           `json:"version"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Status       APIVersionStatus `json:"status"`
	Released     time.Time        `json:"released"`
	Deprecated   bool             `json:"deprecated"`
	DeprecatedAt *time.Time       `json:"deprecated_at,omitempty"`
	EOLDate      *time.Time       `json:"eol_date,omitempty"`
}

// compareVersions compares two version strings
func (vd *VersionedDocumentation) compareVersions(v1, v2 string) int {
	// Simple version comparison - would use semver package in production
	if v1 == v2 {
		return 0
	}
	if v1 < v2 {
		return -1
	}
	return 1
}

// GenerateChangelog generates a changelog between versions
func (vd *VersionedDocumentation) GenerateChangelog() *Changelog {
	versions := vd.versionManager.GetAllVersions()
	changelog := &Changelog{
		Entries: make([]*ChangelogEntry, 0),
	}

	// Create sorted version list
	var sortedVersions []*APIVersion
	for _, version := range versions {
		sortedVersions = append(sortedVersions, version)
	}

	// Sort by release date (newest first)
	sort.Slice(sortedVersions, func(i, j int) bool {
		return sortedVersions[i].Released.After(sortedVersions[j].Released)
	})

	// Generate changelog entries
	for _, version := range sortedVersions {
		entry := &ChangelogEntry{
			Version:     version.Version,
			Name:        version.Name,
			Released:    version.Released,
			Status:      version.Status,
			Description: version.Description,
		}

		// Add deprecation info
		if version.Deprecated {
			entry.Deprecated = true
			entry.DeprecatedAt = version.DeprecatedAt
			entry.EOLDate = version.EOLDate
		}

		// Extract changes from config
		if changes, exists := version.Config["changes"]; exists {
			if changesList, ok := changes.([]string); ok {
				entry.Changes = changesList
			}
		}

		if breaking, exists := version.Config["breaking_changes"]; exists {
			if breakingList, ok := breaking.([]string); ok {
				entry.BreakingChanges = breakingList
			}
		}

		changelog.Entries = append(changelog.Entries, entry)
	}

	return changelog
}

// Changelog represents API version changelog
type Changelog struct {
	Entries []*ChangelogEntry `json:"entries"`
}

// ChangelogEntry represents a single changelog entry
type ChangelogEntry struct {
	Version         string           `json:"version"`
	Name            string           `json:"name"`
	Released        time.Time        `json:"released"`
	Status          APIVersionStatus `json:"status"`
	Description     string           `json:"description"`
	Changes         []string         `json:"changes,omitempty"`
	BreakingChanges []string         `json:"breaking_changes,omitempty"`
	Deprecated      bool             `json:"deprecated"`
	DeprecatedAt    *time.Time       `json:"deprecated_at,omitempty"`
	EOLDate         *time.Time       `json:"eol_date,omitempty"`
}

// GenerateMarkdownChangelog generates a markdown changelog
func (vd *VersionedDocumentation) GenerateMarkdownChangelog() (string, error) {
	changelog := vd.GenerateChangelog()
	
	var md strings.Builder
	md.WriteString("# API Changelog\n\n")
	
	for _, entry := range changelog.Entries {
		md.WriteString(fmt.Sprintf("## %s", entry.Version))
		if entry.Name != "" {
			md.WriteString(fmt.Sprintf(" - %s", entry.Name))
		}
		md.WriteString("\n\n")
		
		md.WriteString(fmt.Sprintf("**Released:** %s\n", entry.Released.Format("2006-01-02")))
		md.WriteString(fmt.Sprintf("**Status:** %s\n", entry.Status))
		
		if entry.Deprecated {
			md.WriteString("**⚠️ Deprecated**")
			if entry.DeprecatedAt != nil {
				md.WriteString(fmt.Sprintf(" (since %s)", entry.DeprecatedAt.Format("2006-01-02")))
			}
			if entry.EOLDate != nil {
				md.WriteString(fmt.Sprintf(" - End of Life: %s", entry.EOLDate.Format("2006-01-02")))
			}
			md.WriteString("\n")
		}
		
		md.WriteString("\n")
		
		if entry.Description != "" {
			md.WriteString(fmt.Sprintf("%s\n\n", entry.Description))
		}
		
		if len(entry.Changes) > 0 {
			md.WriteString("### Changes\n\n")
			for _, change := range entry.Changes {
				md.WriteString(fmt.Sprintf("- %s\n", change))
			}
			md.WriteString("\n")
		}
		
		if len(entry.BreakingChanges) > 0 {
			md.WriteString("### ⚠️ Breaking Changes\n\n")
			for _, breaking := range entry.BreakingChanges {
				md.WriteString(fmt.Sprintf("- %s\n", breaking))
			}
			md.WriteString("\n")
		}
		
		md.WriteString("---\n\n")
	}
	
	return md.String(), nil
}

// GetLatestVersion gets the latest stable version
func (vd *VersionedDocumentation) GetLatestVersion() string {
	versions := vd.versionManager.GetAllVersions()
	
	var latest string
	var latestTime time.Time
	
	for version, info := range versions {
		if info.Status == VersionStatusStable && info.Released.After(latestTime) {
			latest = version
			latestTime = info.Released
		}
	}
	
	if latest == "" {
		// Fallback to any version
		for version := range versions {
			return version
		}
	}
	
	return latest
}

// GetDefaultSpec gets the default version spec
func (vd *VersionedDocumentation) GetDefaultSpec() (*VersionedSpec, error) {
	defaultVersion := vd.config.DefaultVersion
	if defaultVersion == "" {
		defaultVersion = vd.GetLatestVersion()
	}
	
	if defaultVersion == "" {
		return nil, fmt.Errorf("no default version available")
	}
	
	specs, err := vd.GenerateVersionedSpecs()
	if err != nil {
		return nil, err
	}
	
	spec, exists := specs[defaultVersion]
	if !exists {
		// Return any available spec
		for _, s := range specs {
			return s, nil
		}
		return nil, fmt.Errorf("no specifications available")
	}
	
	return spec, nil
}

// ValidateVersionedSpecs validates all version specifications
func (vd *VersionedDocumentation) ValidateVersionedSpecs() map[string][]string {
	errors := make(map[string][]string)
	
	for version, builder := range vd.builders {
		if validationErrors := builder.ValidateSpec(); len(validationErrors) > 0 {
			errors[version] = validationErrors
		}
	}
	
	return errors
}

// Documentation migration helpers

// MigrateRouteDocumentation migrates route documentation between versions
func (vd *VersionedDocumentation) MigrateRouteDocumentation(fromVersion, toVersion string, routePattern string) error {
	fromBuilder := vd.GetBuilder(fromVersion)
	toBuilder := vd.GetBuilder(toVersion)
	
	if fromBuilder == nil || toBuilder == nil {
		return fmt.Errorf("version not found")
	}
	
	// Get documentation from source version
	fromDocs := fromBuilder.GetAllRoutes()
	
	// Find matching routes and migrate
	for route, doc := range fromDocs {
		if routePattern == "" || strings.Contains(route, routePattern) {
			// Migrate to target version
			parts := strings.SplitN(route, " ", 2)
			if len(parts) == 2 {
				toBuilder.DocumentRoute(parts[0], parts[1], doc)
			}
		}
	}
	
	return nil
}

// CompareVersions generates a comparison between two API versions
func (vd *VersionedDocumentation) CompareVersions(version1, version2 string) (*VersionComparison, error) {
	builder1 := vd.GetBuilder(version1)
	builder2 := vd.GetBuilder(version2)
	
	if builder1 == nil || builder2 == nil {
		return nil, fmt.Errorf("one or both versions not found")
	}
	
	spec1, err1 := builder1.GenerateOpenAPISpec()
	spec2, err2 := builder2.GenerateOpenAPISpec()
	
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("failed to generate specs for comparison")
	}
	
	comparison := &VersionComparison{
		Version1: version1,
		Version2: version2,
		Added:    make([]string, 0),
		Removed:  make([]string, 0),
		Modified: make([]string, 0),
	}
	
	// Compare paths
	paths1 := make(map[string]bool)
	for path := range spec1.Paths {
		paths1[path] = true
	}
	
	paths2 := make(map[string]bool)
	for path := range spec2.Paths {
		paths2[path] = true
		if !paths1[path] {
			comparison.Added = append(comparison.Added, path)
		}
	}
	
	for path := range paths1 {
		if !paths2[path] {
			comparison.Removed = append(comparison.Removed, path)
		}
	}
	
	return comparison, nil
}

// VersionComparison represents a comparison between two API versions
type VersionComparison struct {
	Version1 string   `json:"version1"`
	Version2 string   `json:"version2"`
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Modified []string `json:"modified"`
}