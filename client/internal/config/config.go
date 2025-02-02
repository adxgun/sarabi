package config

import (
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

const (
	Path       = ".sarabi.yml"
	appName    = ".sarabi"
	configName = ".sarabiconfig.yml"
)

type (
	ApplicationConfig struct {
		ApplicationID uuid.UUID `yaml:"applicationID"`
		Frontend      string
		Backend       string
	}

	Config struct {
		Host string
	}
)

func ParseApplicationConfig() (ApplicationConfig, error) {
	c := ApplicationConfig{}
	fi, err := os.Open(Path)
	if err != nil {
		return c, err
	}

	value, err := io.ReadAll(fi)
	if err != nil {
		return c, err
	}

	if err = yaml.Unmarshal(value, &c); err != nil {
		return c, err
	}

	return c, nil
}

func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	var configDir string
	switch runtime.GOOS {
	case "linux", "darwin":
		configDir = filepath.Join(homeDir, ".config", appName)
	case "windows":
		configDir = filepath.Join(os.Getenv("APPDATA"), appName)
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}

func Parse() (Config, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return Config{}, err
	}

	configPath := filepath.Join(configDir, configName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return Config{}, err
	}

	return *cfg, nil
}

// SaveConfig writes the configuration file to the config directory.
func SaveConfig(cfg Config) error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, configName)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}

	return nil
}
