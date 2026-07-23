package localrepo

import (
	"fmt"
	"regexp"
	"strings"
)

var projectNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// RequestValidation records an unprivileged workflow request that passed local validation.
type RequestValidation struct {
	// Project is the selected registry key.
	Project string `json:"project"`
	// PackageName is the registered package identity.
	PackageName string `json:"package_name"`
	// Tag is the optional stable release tag supplied by a publish request.
	Tag string `json:"tag,omitempty"`
	// PackageVersion is the validated tag with exactly one leading v removed.
	PackageVersion string `json:"package_version,omitempty"`
}

// ValidateRequest verifies a project against the registry and an optional release tag.
func ValidateRequest(registryPath string, project string, tag string) (RequestValidation, error) {
	if !projectNamePattern.MatchString(project) {
		return RequestValidation{}, fmt.Errorf(
			"project %q must use lowercase letters, numbers, and single hyphens",
			project,
		)
	}
	config, err := loadProject(registryPath, project)
	if err != nil {
		return RequestValidation{}, err
	}
	validation := RequestValidation{Project: project, PackageName: config.PackageName}
	if tag != "" {
		if _, err := parseVersion(tag); err != nil {
			return RequestValidation{}, err
		}
		validation.Tag = tag
		validation.PackageVersion = strings.TrimPrefix(tag, "v")
	}

	return validation, nil
}
