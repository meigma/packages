package localrepo

import (
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

type registry struct {
	Schema   int                      `yaml:"schema"`
	Projects map[string]projectConfig `yaml:"projects"`
}

type projectConfig struct {
	PackageName string      `yaml:"package_name"`
	Assets      assetConfig `yaml:"assets"`
}

type assetConfig struct {
	DEB string `yaml:"deb"`
	RPM string `yaml:"rpm"`
}

func loadProject(path string, name string) (projectConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return projectConfig{}, fmt.Errorf("read registry: %w", err)
	}

	var config registry
	if err := yaml.Unmarshal(content, &config); err != nil {
		return projectConfig{}, fmt.Errorf("parse registry: %w", err)
	}
	if config.Schema != 1 {
		return projectConfig{}, fmt.Errorf("registry schema must be 1, got %d", config.Schema)
	}

	project, ok := config.Projects[name]
	if !ok {
		return projectConfig{}, fmt.Errorf("project %q is not registered", name)
	}
	if err := project.validate(name); err != nil {
		return projectConfig{}, err
	}

	return project, nil
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

	return nil
}
