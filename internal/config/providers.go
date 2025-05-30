package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// EnvProvider loads configuration from environment variables
type EnvProvider struct{}

func (ep *EnvProvider) Name() string {
	return "env"
}

func (ep *EnvProvider) Load() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	for _, env := range os.Environ() {
		if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
			key := strings.ToLower(parts[0])
			value := parts[1]
			
			// Try to parse as different types
			if parsed := ParseValue(value); parsed != nil {
				result[key] = parsed
			} else {
				result[key] = value
			}
		}
	}
	
	return result, nil
}

func (ep *EnvProvider) Watch() (<-chan ConfigEvent, error) {
	// Environment variables can't be watched easily
	return nil, errors.New("environment provider doesn't support watching")
}

// DotEnvProvider loads configuration from .env files
type DotEnvProvider struct {
	filepath string
}

func NewDotEnvProvider(filepath string) (*DotEnvProvider, error) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, fmt.Errorf("env file %s does not exist", filepath)
	}
	
	return &DotEnvProvider{filepath: filepath}, nil
}

func (dep *DotEnvProvider) Name() string {
	return fmt.Sprintf("dotenv:%s", dep.filepath)
}

func (dep *DotEnvProvider) Load() (map[string]interface{}, error) {
	file, err := os.Open(dep.filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	result := make(map[string]interface{})
	scanner := bufio.NewScanner(file)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parse key=value pairs
		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			
			// Remove quotes if present
			value = RemoveQuotes(value)
			
			// Expand variables
			value = ExpandVariables(value, result)
			
			// Parse value
			if parsed := ParseValue(value); parsed != nil {
				result[strings.ToLower(key)] = parsed
			} else {
				result[strings.ToLower(key)] = value
			}
		} else {
			return nil, fmt.Errorf("invalid syntax in %s at line %d: %s", dep.filepath, lineNum, line)
		}
	}
	
	return result, scanner.Err()
}

func (dep *DotEnvProvider) Watch() (<-chan ConfigEvent, error) {
	// File watching would require a file watcher implementation
	return nil, errors.New("dotenv provider doesn't support watching yet")
}

// FileProvider loads configuration from JSON/YAML files in a directory
type FileProvider struct {
	BasePath string
}

func (fp *FileProvider) Name() string {
	return fmt.Sprintf("file:%s", fp.BasePath)
}

func (fp *FileProvider) Load() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	if _, err := os.Stat(fp.BasePath); os.IsNotExist(err) {
		return result, nil // No config directory is fine
	}
	
	err := filepath.WalkDir(fp.BasePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() {
			return nil
		}
		
		// Only process JSON files for now (YAML support can be added later)
		if !strings.HasSuffix(path, ".json") {
			return nil
		}
		
		// Read and parse file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		
		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}
		
		// Use filename (without extension) as key
		filename := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		result[filename] = config
		
		return nil
	})
	
	return result, err
}

func (fp *FileProvider) Watch() (<-chan ConfigEvent, error) {
	return nil, errors.New("file provider doesn't support watching yet")
}

// MemoryProvider allows for in-memory configuration (useful for testing)
type MemoryProvider struct {
	name   string
	values map[string]interface{}
}

func NewMemoryProvider(name string, values map[string]interface{}) *MemoryProvider {
	return &MemoryProvider{
		name:   name,
		values: values,
	}
}

func (mp *MemoryProvider) Name() string {
	return fmt.Sprintf("memory:%s", mp.name)
}

func (mp *MemoryProvider) Load() (map[string]interface{}, error) {
	// Return a copy to prevent external modification
	result := make(map[string]interface{})
	for k, v := range mp.values {
		result[k] = v
	}
	return result, nil
}

func (mp *MemoryProvider) Watch() (<-chan ConfigEvent, error) {
	return nil, errors.New("memory provider doesn't support watching")
}

// Set allows updating values in the memory provider
func (mp *MemoryProvider) Set(key string, value interface{}) {
	if mp.values == nil {
		mp.values = make(map[string]interface{})
	}
	mp.values[key] = value
}

// Delete removes a key from the memory provider
func (mp *MemoryProvider) Delete(key string) {
	delete(mp.values, key)
}

// Clear removes all values from the memory provider
func (mp *MemoryProvider) Clear() {
	mp.values = make(map[string]interface{})
}