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
	Repository    string                        `yaml:"repository"`
	PackageName   string                        `yaml:"package_name"`
	Retention     int                           `yaml:"retention"`
	Assets        assetConfig                   `yaml:"assets"`
	Architectures map[string]architectureConfig `yaml:"architectures"`
	Provenance    provenanceConfig              `yaml:"provenance"`
}

type assetConfig struct {
	Checksums string `yaml:"checksums"`
	DEB       string `yaml:"deb"`
	RPM       string `yaml:"rpm"`
}

type architectureConfig struct {
	DEB string `yaml:"deb"`
	RPM string `yaml:"rpm"`
}

type provenanceConfig struct {
	SignerWorkflow string `yaml:"signer_workflow"`
}

// ReleaseSource describes the trusted GitHub Release inputs for one project.
type ReleaseSource struct {
	// Repository is the owner/name GitHub repository identifier.
	Repository string
	// PackageName is the package identity expected inside every selected asset.
	PackageName string
	// Checksums is the exact checksum asset name.
	Checksums string
	// DEBPattern selects every expected DEB after ${version} expansion.
	DEBPattern string
	// RPMPattern selects every expected RPM after ${version} expansion.
	RPMPattern string
	// Architectures maps repository architecture names to package metadata values.
	Architectures map[string]ReleaseArchitecture
	// SignerWorkflow is the exact reusable workflow trusted for SLSA provenance.
	SignerWorkflow string
}

// ReleaseArchitecture maps one repository architecture to DEB and RPM metadata.
type ReleaseArchitecture struct {
	// DEB is the architecture recorded inside the DEB package.
	DEB string
	// RPM is the architecture recorded inside the RPM package.
	RPM string
}

// LoadReleaseSource loads and validates one GitHub Release source from the registry.
func LoadReleaseSource(path string, name string) (ReleaseSource, error) {
	project, err := loadProject(path, name)
	if err != nil {
		return ReleaseSource{}, err
	}
	if err := project.validateReleaseSource(name); err != nil {
		return ReleaseSource{}, err
	}

	architectures := make(map[string]ReleaseArchitecture, len(project.Architectures))
	for architecture, mapping := range project.Architectures {
		architectures[architecture] = ReleaseArchitecture(mapping)
	}

	return ReleaseSource{
		Repository:     project.Repository,
		PackageName:    project.PackageName,
		Checksums:      project.Assets.Checksums,
		DEBPattern:     project.Assets.DEB,
		RPMPattern:     project.Assets.RPM,
		Architectures:  architectures,
		SignerWorkflow: project.Provenance.SignerWorkflow,
	}, nil
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

func (config projectConfig) validateReleaseSource(name string) error {
	if !strings.HasPrefix(config.Repository, "meigma/") || strings.Count(config.Repository, "/") != 1 {
		return fmt.Errorf("project %q repository must identify one meigma repository", name)
	}
	if strings.TrimSpace(config.Assets.Checksums) == "" {
		return fmt.Errorf("project %q field assets.checksums is required for release discovery", name)
	}
	if len(config.Architectures) == 0 {
		return fmt.Errorf("project %q must define at least one release architecture", name)
	}
	debMappings := make(map[string]string, len(config.Architectures))
	rpmMappings := make(map[string]string, len(config.Architectures))
	for architecture, mapping := range config.Architectures {
		if !projectNamePattern.MatchString(architecture) {
			return fmt.Errorf("project %q architecture %q is invalid", name, architecture)
		}
		if strings.TrimSpace(mapping.DEB) == "" || strings.TrimSpace(mapping.RPM) == "" {
			return fmt.Errorf("project %q architecture %q must define deb and rpm mappings", name, architecture)
		}
		if other, exists := debMappings[mapping.DEB]; exists {
			return fmt.Errorf(
				"project %q architectures %q and %q use the same DEB mapping %q",
				name,
				other,
				architecture,
				mapping.DEB,
			)
		}
		if other, exists := rpmMappings[mapping.RPM]; exists {
			return fmt.Errorf(
				"project %q architectures %q and %q use the same RPM mapping %q",
				name,
				other,
				architecture,
				mapping.RPM,
			)
		}
		debMappings[mapping.DEB] = architecture
		rpmMappings[mapping.RPM] = architecture
	}
	expectedSignerPrefix := config.Repository + "/.github/workflows/"
	if !strings.HasPrefix(config.Provenance.SignerWorkflow, expectedSignerPrefix) ||
		!strings.HasSuffix(config.Provenance.SignerWorkflow, ".yml") {
		return fmt.Errorf(
			"project %q provenance signer must be a YAML workflow in %s",
			name,
			config.Repository,
		)
	}

	return nil
}

// ExpandAssetPattern substitutes the stable release version into an asset pattern.
func ExpandAssetPattern(pattern string, tag string) string {
	return strings.ReplaceAll(pattern, "${version}", strings.TrimPrefix(tag, "v"))
}
