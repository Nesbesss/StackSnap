package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type StorageType string

const (
	StorageLocal StorageType = "local"
	StorageS3  StorageType = "s3"
)

type Config struct {
	LicenseKey    string    `yaml:"license_key" json:"license_key"`
	LicenseServerURL string    `yaml:"license_server_url,omitempty" json:"license_server_url,omitempty"`
	MachineID    string    `yaml:"machine_id" json:"machine_id"`
	Storage     StorageConfig `yaml:"storage" json:"storage"`
	ManualStacks   []string   `yaml:"manual_stacks,omitempty" json:"manual_stacks,omitempty"`
}

type StorageConfig struct {
	Type    StorageType `yaml:"type" json:"type"`
	Path    string   `yaml:"path,omitempty" json:"path,omitempty"`
	S3Bucket  string   `yaml:"s3_bucket,omitempty" json:"s3_bucket,omitempty"`
	S3Region  string   `yaml:"s3_region,omitempty" json:"s3_region,omitempty"`
	S3Endpoint string   `yaml:"s3_endpoint,omitempty" json:"s3_endpoint,omitempty"`
	S3AccessKey string   `yaml:"s3_access_key,omitempty" json:"s3_access_key,omitempty"`
	S3SecretKey string   `yaml:"s3_secret_key,omitempty" json:"s3_secret_key,omitempty"`
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".stacksnap")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func VerificationsPath() string {
	return filepath.Join(ConfigDir(), "verifications.json")
}

func Load() (*Config, error) {
	path := ConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config not found")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath(), data, 0600)
}
