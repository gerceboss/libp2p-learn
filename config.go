package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// Config represents the application configuration
type Config struct {
	// Network settings
	ListenPort     int      `json:"listen_port"`
	BootstrapPeers []string `json:"bootstrap_peers"`
	
	// Connection management
	MaxConnections int `json:"max_connections"`
	LowWater       int `json:"low_water"`
	HighWater      int `json:"high_water"`
	
	// Features
	EnableRelay       bool `json:"enable_relay"`
	EnableHolePunch   bool `json:"enable_hole_punch"`
	EnableAutoNAT     bool `json:"enable_autonat"`
	EnableWebSocket   bool `json:"enable_websocket"`
	
	// Logging
	LogLevel string `json:"log_level"`
	LogFile  string `json:"log_file"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		ListenPort:     0, // Random port
		BootstrapPeers: []string{
			// Default IPFS bootstrap nodes
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		},
		MaxConnections:    1000,
		LowWater:         50,
		HighWater:        200,
		EnableRelay:       false,
		EnableHolePunch:   true,
		EnableAutoNAT:     true,
		EnableWebSocket:   true,
		LogLevel:         "info",
		LogFile:          "",
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(filepath string) (*Config, error) {
	config := DefaultConfig()

	if filepath == "" {
		return config, nil
	}

	file, err := os.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			logrus.WithField("file", filepath).Info("Config file not found, using defaults")
			return config, nil
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	logrus.WithField("file", filepath).Info("Configuration loaded")
	return config, nil
}

// SaveConfig saves configuration to a file
func (c *Config) SaveConfig(filepath string) error {
	// Create directory if it doesn't exist
	dir := filepath
	if ext := filepath[len(filepath)-5:]; ext == ".json" {
		dir = filepath[:len(filepath)-len(filepath[:])]
	}
	
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	logrus.WithField("file", filepath).Info("Configuration saved")
	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.MaxConnections <= 0 {
		return fmt.Errorf("max_connections must be positive")
	}

	if c.LowWater <= 0 || c.HighWater <= 0 {
		return fmt.Errorf("low_water and high_water must be positive")
	}

	if c.LowWater >= c.HighWater {
		return fmt.Errorf("low_water must be less than high_water")
	}

	if c.ListenPort < 0 || c.ListenPort > 65535 {
		return fmt.Errorf("listen_port must be between 0 and 65535")
	}

	validLogLevels := map[string]bool{
		"trace": true, "debug": true, "info": true,
		"warn": true, "error": true, "fatal": true, "panic": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log_level: %s", c.LogLevel)
	}

	return nil
}

// SetupLogging configures the logging system based on config
func (c *Config) SetupLogging() error {
	level, err := logrus.ParseLevel(c.LogLevel)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	logrus.SetLevel(level)

	// Set JSON formatter for structured logging
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Set up log file if specified
	if c.LogFile != "" {
		// Create log directory
		if err := os.MkdirAll(filepath.Dir(c.LogFile), 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		file, err := os.OpenFile(c.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		logrus.SetOutput(file)
		logrus.WithField("file", c.LogFile).Info("Logging to file")
	}

	return nil
}
