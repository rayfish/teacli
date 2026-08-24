// Package config handles configuration management for teacli.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-ini/ini"
)

const (
	configDirName     = "teacli"
	defaultConfigFile = "teacli.ini"
)

// Config represents the CLI configuration
type Config struct {
	file          *ini.File
	path          string
	servers       map[string]*ServerConfig
	defaultServer string
}

// ServerConfig holds configuration for a single Gitea server
type ServerConfig struct {
	URL     string `ini:"url"`
	Token   string `ini:"token"`
	Timeout int    `ini:"timeout,omitempty"`
}

// Load loads configuration from the default location
func Load() (*Config, error) {
	return LoadFrom("")
}

// LoadFrom loads configuration from path, or from the default location when
// path is empty. This is what the --config flag resolves to.
func LoadFrom(path string) (*Config, error) {
	configPath := path
	if configPath == "" {
		configPath = getConfigPath()
	}

	cfg := &Config{
		servers: make(map[string]*ServerConfig),
		path:    configPath,
	}

	// Create config file if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg.file = ini.Empty()
		return cfg, nil
	}

	// Load existing config
	var err error
	cfg.file, err = ini.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Parse servers
	for _, section := range cfg.file.Sections() {
		if section.Name() == "DEFAULT" || section.Name() == "global" {
			continue
		}

		server := &ServerConfig{}
		if err := section.MapTo(server); err != nil {
			return nil, fmt.Errorf("failed to parse server %s: %w", section.Name(), err)
		}
		cfg.servers[section.Name()] = server
	}

	// Get default server
	if section, err := cfg.file.GetSection("global"); err == nil {
		cfg.defaultServer = section.Key("default_server").String()
	}

	return cfg, nil
}

// Save persists the configuration to disk
func (c *Config) Save() error {
	if c.file == nil {
		c.file = ini.Empty()
	}

	// Ensure global section exists
	if _, err := c.file.GetSection("global"); err != nil {
		c.file.NewSection("global")
	}

	if c.defaultServer != "" {
		c.file.Section("global").Key("default_server").SetValue(c.defaultServer)
	}

	// Update server sections
	for name, server := range c.servers {
		section, err := c.file.GetSection(name)
		if err != nil {
			section, _ = c.file.NewSection(name)
		}
		section.Key("url").SetValue(server.URL)
		section.Key("token").SetValue(server.Token)
		if server.Timeout > 0 {
			section.Key("timeout").SetValue(fmt.Sprintf("%d", server.Timeout))
		}
	}

	// An explicit --config may name a file in a directory that does not exist
	// yet; the default path is created by getConfigPath.
	if dir := filepath.Dir(c.path); dir != "" {
		os.MkdirAll(dir, 0755)
	}

	return c.file.SaveTo(c.path)
}

// AddServer adds or updates a server configuration
func (c *Config) AddServer(name, url, token string) error {
	c.servers[name] = &ServerConfig{
		URL:   url,
		Token: token,
	}
	return c.Save()
}

// RemoveServer removes a server configuration
func (c *Config) RemoveServer(name string) error {
	if _, exists := c.servers[name]; !exists {
		return fmt.Errorf("server not found: %s", name)
	}
	delete(c.servers, name)
	return c.Save()
}

// GetServer retrieves a server configuration by name
func (c *Config) GetServer(name string) (*ServerConfig, error) {
	server, exists := c.servers[name]
	if !exists {
		return nil, fmt.Errorf("server not configured: %s", name)
	}
	return server, nil
}

// GetDefaultServer returns the default server configuration
func (c *Config) GetDefaultServer() (*ServerConfig, error) {
	if c.defaultServer == "" {
		// Return first available server
		for name := range c.servers {
			c.defaultServer = name
			return c.GetServer(name)
		}
		return nil, fmt.Errorf("no servers configured")
	}
	return c.GetServer(c.defaultServer)
}

// SetDefaultServer sets the default server
func (c *Config) SetDefaultServer(name string) error {
	if _, exists := c.servers[name]; !exists {
		return fmt.Errorf("server not found: %s", name)
	}
	c.defaultServer = name
	return c.Save()
}

// ListServers returns all configured server names
func (c *Config) ListServers() []string {
	servers := make([]string, 0, len(c.servers))
	for name := range c.servers {
		servers = append(servers, name)
	}
	return servers
}

// getConfigPath returns the path to the config file, ~/.config/teacli/teacli.ini.
func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return defaultConfigFile
	}

	configDir := filepath.Join(homeDir, ".config", configDirName)
	os.MkdirAll(configDir, 0755)
	return filepath.Join(configDir, defaultConfigFile)
}
