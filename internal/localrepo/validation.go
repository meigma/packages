package localrepo

import (
	"fmt"
	"regexp"
)

var projectNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// RequestValidation records an unprivileged workflow request that passed local validation.
type RequestValidation struct {
	// Project is the selected registry key.
	Project string `json:"project"`
	// Tag is the optional stable release tag supplied by a publish request.
	Tag string `json:"tag,omitempty"`
}

// ValidateRequest verifies a project against the registry and an optional release tag.
func ValidateRequest(registryPath string, project string, tag string) (RequestValidation, error) {
	if !projectNamePattern.MatchString(project) {
		return RequestValidation{}, fmt.Errorf(
			"project %q must use lowercase letters, numbers, and single hyphens",
			project,
		)
	}
	if _, err := loadProject(registryPath, project); err != nil {
		return RequestValidation{}, err
	}
	if tag != "" {
		if _, err := parseVersion(tag); err != nil {
			return RequestValidation{}, err
		}
	}

	return RequestValidation{Project: project, Tag: tag}, nil
}
