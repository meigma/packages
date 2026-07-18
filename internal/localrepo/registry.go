package localrepo

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

type registry struct {
	Schema   int                      `yaml:"schema"`
	Defaults registryDefaults         `yaml:"defaults"`
	Projects map[string]projectConfig `yaml:"projects"`
}

type registryDefaults struct {
	Retention int `yaml:"retention"`
}

type projectConfig struct {
	PackageName string      `yaml:"package_name"`
	Retention   int         `yaml:"retention"`
	Assets      assetConfig `yaml:"assets"`
}

type assetConfig struct {
	Checksums string `yaml:"checksums"`
	DEB       string `yaml:"deb"`
	RPM       string `yaml:"rpm"`
}

func loadProject(path string, name string) (projectConfig, error) {
	config, _, err := loadRegistry(path)
	if err != nil {
		return projectConfig{}, err
	}

	project, ok := config.Projects[name]
	if !ok {
		return projectConfig{}, fmt.Errorf("project %q is not registered", name)
	}
	if project.Retention == 0 {
		project.Retention = config.Defaults.Retention
	}
	if project.Retention == 0 {
		project.Retention = 5
	}
	if err := project.validate(name); err != nil {
		return projectConfig{}, err
	}

	return project, nil
}

func loadRegistry(path string) (registry, []byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return registry{}, nil, fmt.Errorf("read registry: %w", err)
	}

	var config registry
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		return registry{}, nil, fmt.Errorf("parse registry: %w", err)
	}
	if config.Schema != 1 {
		return registry{}, nil, fmt.Errorf("registry schema must be 1, got %d", config.Schema)
	}
	if config.Defaults.Retention < 0 {
		return registry{}, nil, errors.New("default retention must be positive")
	}
	return config, content, nil
}

func (config projectConfig) validate(name string) error {
	fields := map[string]string{
		"package_name": config.PackageName,
		"assets.deb":   config.Assets.DEB,
		"assets.rpm":   config.Assets.RPM,
	}
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("project %q field %s is required", name, field)
		}
	}
	if config.Retention <= 0 {
		return fmt.Errorf("project %q retention must be positive", name)
	}
	for field, value := range map[string]string{
		"assets.checksums": config.Assets.Checksums,
		"assets.deb":       config.Assets.DEB,
		"assets.rpm":       config.Assets.RPM,
	} {
		if value == "" {
			continue
		}
		if filepath.IsAbs(value) || filepath.Base(value) != value || strings.Contains(value, "..") {
			return fmt.Errorf("project %q field %s must be a file-name pattern", name, field)
		}
	}

	return nil
}
