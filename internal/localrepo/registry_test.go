package localrepo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadProject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		registry    string
		project     string
		want        projectConfig
		errorSubstr string
	}{
		{
			name: "loads the selected fixture project",
			registry: `schema: 1
projects:
  phase1-fixture:
    package_name: meigma-phase0
    assets:
      deb: meigma-phase0_1.0.0_all.deb
      rpm: meigma-phase0-1.0.0-1.noarch.rpm
`,
			project: "phase1-fixture",
			want: projectConfig{
				PackageName: "meigma-phase0",
				Retention:   5,
				Assets: assetConfig{
					DEB: "meigma-phase0_1.0.0_all.deb",
					RPM: "meigma-phase0-1.0.0-1.noarch.rpm",
				},
			},
		},
		{
			name: "rejects an unsupported schema",
			registry: `schema: 2
projects: {}
`,
			project:     "phase1-fixture",
			errorSubstr: "registry schema must be 1, got 2",
		},
		{
			name: "rejects an unknown project",
			registry: `schema: 1
projects: {}
`,
			project:     "phase1-fixture",
			errorSubstr: `project "phase1-fixture" is not registered`,
		},
		{
			name: "rejects an incomplete project",
			registry: `schema: 1
projects:
  phase1-fixture:
    package_name: meigma-phase0
    assets:
      deb: meigma-phase0_1.0.0_all.deb
`,
			project:     "phase1-fixture",
			errorSubstr: "field assets.rpm is required",
		},
		{
			name: "rejects unknown registry fields",
			registry: `schema: 1
projects:
  phase1-fixture:
    package_name: meigma-phase0
    typo: true
    assets:
      deb: meigma-phase0_1.0.0_all.deb
      rpm: meigma-phase0-1.0.0-1.noarch.rpm
`,
			project:     "phase1-fixture",
			errorSubstr: "field typo not found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := writeRegistry(t, test.registry)
			got, err := loadProject(path, test.project)
			if test.errorSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.errorSubstr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestBuildRejectsAnExistingCandidateRoot(t *testing.T) {
	t.Parallel()

	registryPath := writeRegistry(t, `schema: 1
projects:
  phase1-fixture:
    package_name: meigma-phase0
    assets:
      deb: meigma-phase0_1.0.0_all.deb
      rpm: meigma-phase0-1.0.0-1.noarch.rpm
`)
	root := filepath.Join(t.TempDir(), "candidate")
	require.NoError(t, os.Mkdir(root, 0o755))

	_, err := Build(t.Context(), Request{
		RegistryPath: registryPath,
		Project:      "phase1-fixture",
		ReleaseDir:   "/unused",
		Root:         root,
		GNUPGHome:    "/unused",
		SigningKey:   "0123456789ABCDEF",
		BaseURL:      "http://phase1-repo:8080",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "candidate root already exists")
}

func TestValidateRequest(t *testing.T) {
	t.Parallel()

	registryPath := writeRegistry(t, `schema: 1
projects:
  phase3-fixture:
    package_name: meigma-phase0
    assets:
      deb: meigma-phase0_1.0.0_all.deb
      rpm: meigma-phase0-1.0.0-1.noarch.rpm
`)
	tests := []struct {
		name        string
		project     string
		tag         string
		want        RequestValidation
		errorSubstr string
	}{
		{
			name:    "accepts a registered rebuild project without a tag",
			project: "phase3-fixture",
			want:    RequestValidation{Project: "phase3-fixture"},
		},
		{
			name:    "accepts a registered publish project and stable tag",
			project: "phase3-fixture",
			tag:     "v2.1.0",
			want:    RequestValidation{Project: "phase3-fixture", Tag: "v2.1.0"},
		},
		{
			name:        "rejects an unsafe project name",
			project:     "../phase3-fixture",
			errorSubstr: "must use lowercase letters, numbers, and single hyphens",
		},
		{
			name:        "rejects an unknown project",
			project:     "missing",
			errorSubstr: "is not registered",
		},
		{
			name:        "rejects a prerelease tag",
			project:     "phase3-fixture",
			tag:         "v2.1.0-rc.1",
			errorSubstr: "stable v-prefixed semantic version",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := ValidateRequest(registryPath, test.project, test.tag)
			if test.errorSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.errorSubstr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func writeRegistry(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "projects.yml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	return path
}
