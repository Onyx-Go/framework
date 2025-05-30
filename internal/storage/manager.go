package storage

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// DefaultManager implements the Manager interface
type DefaultManager struct {
	config         *ManagerConfig
	drivers        map[string]Driver
	defaultDriver  string
	stats          *StatsCollector
	mutex          sync.RWMutex
}

// NewManager creates a new storage manager
func NewManager(config *ManagerConfig) *DefaultManager {
	if config == nil {
		config = DefaultManagerConfig()
	}
	
	return &DefaultManager{
		config:        config,
		drivers:       make(map[string]Driver),
		defaultDriver: config.DefaultDriver,
		stats:         NewStatsCollector(),
	}
}

// Driver management

// RegisterDriver registers a storage driver
func (m *DefaultManager) RegisterDriver(name string, driver Driver) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Validate driver
	if err := driver.HealthCheck(context.Background()); err != nil {
		return fmt.Errorf("driver health check failed: %w", err)
	}
	
	m.drivers[name] = driver
	
	// Initialize driver stats
	m.stats.InitializeDriver(name, driver.GetConfig().Type)
	
	return nil
}

// GetDriver returns a driver by name
func (m *DefaultManager) GetDriver(name string) (Driver, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	driver, exists := m.drivers[name]
	if !exists {
		return nil, fmt.Errorf("storage driver '%s' not found", name)
	}
	
	return driver, nil
}

// SetDefaultDriver sets the default driver
func (m *DefaultManager) SetDefaultDriver(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.defaultDriver = name
}

// GetDefaultDriver returns the default driver name
func (m *DefaultManager) GetDefaultDriver() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.defaultDriver
}

// ListDrivers returns all registered driver names
func (m *DefaultManager) ListDrivers() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	names := make([]string, 0, len(m.drivers))
	for name := range m.drivers {
		names = append(names, name)
	}
	
	return names
}

// Disk operations

// Disk returns a driver (default or specified)
func (m *DefaultManager) Disk(name ...string) Driver {
	driverName := m.defaultDriver
	if len(name) > 0 && name[0] != "" {
		driverName = name[0]
	}
	
	m.mutex.RLock()
	driver, exists := m.drivers[driverName]
	m.mutex.RUnlock()
	
	if !exists {
		// Try to create the driver from config
		return m.createDriverFromConfig(driverName)
	}
	
	return &instrumentedDriver{
		Driver: driver,
		stats:  m.stats,
		name:   driverName,
	}
}

// Global configuration

// GetConfig returns the manager configuration
func (m *DefaultManager) GetConfig() *ManagerConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config
}

// UpdateConfig updates the manager configuration
func (m *DefaultManager) UpdateConfig(config *ManagerConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.config = config
	m.defaultDriver = config.DefaultDriver
	
	return nil
}

// Health and monitoring

// HealthCheck checks the health of drivers
func (m *DefaultManager) HealthCheck(ctx context.Context, driverName ...string) error {
	if len(driverName) > 0 {
		// Check specific driver
		driver, err := m.GetDriver(driverName[0])
		if err != nil {
			return err
		}
		return driver.HealthCheck(ctx)
	}
	
	// Check all drivers
	m.mutex.RLock()
	drivers := make(map[string]Driver)
	for name, driver := range m.drivers {
		drivers[name] = driver
	}
	m.mutex.RUnlock()
	
	var firstError error
	for name, driver := range drivers {
		if err := driver.HealthCheck(ctx); err != nil {
			if firstError == nil {
				firstError = fmt.Errorf("driver %s health check failed: %w", name, err)
			}
		}
	}
	
	return firstError
}

// GetStats returns storage statistics
func (m *DefaultManager) GetStats(ctx context.Context) (*Stats, error) {
	return m.stats.GetStats(), nil
}

// Lifecycle

// Close closes all drivers
func (m *DefaultManager) Close(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	var firstError error
	
	for name, driver := range m.drivers {
		if err := driver.Close(ctx); err != nil {
			if firstError == nil {
				firstError = fmt.Errorf("error closing driver %s: %w", name, err)
			}
		}
	}
	
	return firstError
}

// Private methods

// createDriverFromConfig creates a driver from configuration
func (m *DefaultManager) createDriverFromConfig(name string) Driver {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Check if already created
	if driver, exists := m.drivers[name]; exists {
		return &instrumentedDriver{
			Driver: driver,
			stats:  m.stats,
			name:   name,
		}
	}
	
	// Get config for this driver
	config, exists := m.config.Drivers[name]
	if !exists {
		// Use default config
		config = DefaultConfig()
		config.LocalPath = fmt.Sprintf("storage/app/%s", name)
	}
	
	// Create driver based on type
	var driver Driver
	var err error
	
	switch config.Type {
	case "local":
		driver, err = NewLocalDriver(name, config)
	default:
		// Default to local storage
		driver, err = NewLocalDriver(name, config)
	}
	
	if err != nil {
		// Return a null driver that returns errors
		return &nullDriver{name: name, err: err}
	}
	
	// Register the driver
	m.drivers[name] = driver
	m.stats.InitializeDriver(name, config.Type)
	
	return &instrumentedDriver{
		Driver: driver,
		stats:  m.stats,
		name:   name,
	}
}

// instrumentedDriver wraps a driver to collect statistics
type instrumentedDriver struct {
	Driver
	stats *StatsCollector
	name  string
}

// Override methods to collect statistics
func (id *instrumentedDriver) Put(ctx context.Context, path string, contents []byte) error {
	start := time.Now()
	err := id.Driver.Put(ctx, path, contents)
	duration := time.Since(start)
	
	if err != nil {
		id.stats.RecordError(id.name, "Put", duration)
	} else {
		id.stats.RecordOperation(id.name, "Put", duration, int64(len(contents)))
	}
	
	return err
}

func (id *instrumentedDriver) Get(ctx context.Context, path string) ([]byte, error) {
	start := time.Now()
	data, err := id.Driver.Get(ctx, path)
	duration := time.Since(start)
	
	if err != nil {
		id.stats.RecordError(id.name, "Get", duration)
	} else {
		id.stats.RecordOperation(id.name, "Get", duration, int64(len(data)))
	}
	
	return data, err
}

func (id *instrumentedDriver) Delete(ctx context.Context, path string) error {
	start := time.Now()
	err := id.Driver.Delete(ctx, path)
	duration := time.Since(start)
	
	if err != nil {
		id.stats.RecordError(id.name, "Delete", duration)
	} else {
		id.stats.RecordOperation(id.name, "Delete", duration, 0)
	}
	
	return err
}

// nullDriver is a driver that always returns errors
type nullDriver struct {
	name string
	err  error
}

func (nd *nullDriver) Put(ctx context.Context, path string, contents []byte) error {
	return fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) PutFile(ctx context.Context, path string, file io.Reader) error {
	return fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) PutFileWithMetadata(ctx context.Context, path string, file io.Reader, metadata map[string]string) error {
	return fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) Get(ctx context.Context, path string) ([]byte, error) {
	return nil, fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) GetStream(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) Exists(ctx context.Context, path string) (bool, error) {
	return false, fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) Delete(ctx context.Context, path string) error {
	return fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) Copy(ctx context.Context, from, to string) error {
	return fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) Move(ctx context.Context, from, to string) error {
	return fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) Size(ctx context.Context, path string) (int64, error) {
	return 0, fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) LastModified(ctx context.Context, path string) (time.Time, error) {
	return time.Time{}, fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) MimeType(ctx context.Context, path string) (string, error) {
	return "", fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) Files(ctx context.Context, directory string) ([]FileInfo, error) {
	return nil, fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) AllFiles(ctx context.Context, directory string) ([]FileInfo, error) {
	return nil, fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) Directories(ctx context.Context, directory string) ([]string, error) {
	return nil, fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) AllDirectories(ctx context.Context, directory string) ([]string, error) {
	return nil, fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) MakeDirectory(ctx context.Context, path string) error {
	return fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) DeleteDirectory(ctx context.Context, path string) error {
	return fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) URL(path string) (string, error) {
	return "", fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) TemporaryURL(path string, expiration time.Duration) (string, error) {
	return "", fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) SignedURL(path string, expiration time.Duration, permissions map[string]interface{}) (string, error) {
	return "", fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) SetVisibility(ctx context.Context, path string, visibility Visibility) error {
	return fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) GetVisibility(ctx context.Context, path string) (Visibility, error) {
	return "", fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) PutMany(ctx context.Context, files map[string][]byte) error {
	return fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) DeleteMany(ctx context.Context, paths []string) error {
	return fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) GetMetadata(ctx context.Context, path string) (map[string]string, error) {
	return nil, fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) SetMetadata(ctx context.Context, path string, metadata map[string]string) error {
	return fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) GetName() string {
	return nd.name
}

func (nd *nullDriver) GetConfig() Config {
	return Config{}
}

func (nd *nullDriver) HealthCheck(ctx context.Context) error {
	return fmt.Errorf("driver %s not available: %w", nd.name, nd.err)
}

func (nd *nullDriver) Close(ctx context.Context) error {
	return nil
}

// SetupManager creates a manager with default drivers
func SetupManager(config *ManagerConfig) (*DefaultManager, error) {
	manager := NewManager(config)
	
	// Create drivers from configuration
	for name, driverConfig := range config.Drivers {
		var driver Driver
		var err error
		
		switch driverConfig.Type {
		case "local":
			driver, err = NewLocalDriver(name, driverConfig)
		default:
			err = fmt.Errorf("unsupported driver type: %s", driverConfig.Type)
		}
		
		if err != nil {
			return nil, fmt.Errorf("failed to create driver %s: %w", name, err)
		}
		
		if err := manager.RegisterDriver(name, driver); err != nil {
			return nil, fmt.Errorf("failed to register driver %s: %w", name, err)
		}
	}
	
	return manager, nil
}