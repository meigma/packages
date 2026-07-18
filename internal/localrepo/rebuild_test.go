package localrepo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		tag         string
		want        [3]int
		errorSubstr string
	}{
		{name: "accepts a stable v-prefixed version", tag: "v12.3.40", want: [3]int{12, 3, 40}},
		{name: "accepts stable build metadata", tag: "v12.3.40+build.2", want: [3]int{12, 3, 40}},
		{name: "rejects a missing prefix", tag: "1.2.3", errorSubstr: "stable v-prefixed semantic version"},
		{name: "rejects a prerelease", tag: "v1.2.3-rc.1", errorSubstr: "stable v-prefixed semantic version"},
		{name: "rejects leading zeroes", tag: "v1.02.3", errorSubstr: "stable v-prefixed semantic version"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseVersion(test.tag)
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

func TestReadChecksums(t *testing.T) {
	t.Parallel()

	assetPath := filepath.Join(t.TempDir(), "fixture.deb")
	require.NoError(t, os.WriteFile(assetPath, []byte("fixture"), 0o644))
	digest, err := digestFile(assetPath)
	require.NoError(t, err)
	checksumsPath := filepath.Join(filepath.Dir(assetPath), "checksums.txt")
	require.NoError(t, os.WriteFile(checksumsPath, []byte(digest+"  fixture.deb\n"), 0o644))

	checksums, err := readChecksums(checksumsPath)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"fixture.deb": digest}, checksums)
}

func TestNewManifestIsStableAcrossReleaseDiscoveryOrder(t *testing.T) {
	t.Parallel()

	request := RebuildRequest{
		Project:    "fixture",
		SigningKey: "ABCDEF",
		BaseURL:    "https://pkgs.example.test/",
	}
	project := projectConfig{Retention: 2}
	releases := []fixtureRelease{
		{
			tag: "v2.0.0",
			records: []PackageRecord{{
				Tag: "v2.0.0", Format: "deb", File: "fixture_2.0.0_all.deb", SHA256: "two",
			}},
		},
		{
			tag: "v1.0.0",
			records: []PackageRecord{{
				Tag: "v1.0.0", Format: "deb", File: "fixture_1.0.0_all.deb", SHA256: "one",
			}},
		},
	}

	first, err := newManifest(request, project, []byte("registry"), releases)
	require.NoError(t, err)
	second, err := newManifest(request, project, []byte("registry"), releases)
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.Equal(t, "https://pkgs.example.test", first.BaseURL)
	assert.NotEmpty(t, first.DesiredStateDigest)
}

func TestVerifyManifestPackagesRejectsTampering(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "apt", "pool", "fixture", "fixture_1.0.0_all.deb")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("tampered"), 0o644))
	manifest := Manifest{
		Project: "fixture",
		Packages: []PackageRecord{{
			Format: "deb",
			File:   "fixture_1.0.0_all.deb",
			SHA256: digestSHA256([]byte("original")),
		}},
	}

	err := verifyManifestPackages(root, manifest)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch for retained package")
}

func TestManifestDigestChangesWithLogicalInputs(t *testing.T) {
	t.Parallel()

	manifest := Manifest{
		Schema:           1,
		Project:          "fixture",
		SelectedVersions: []string{"v1.0.0"},
	}
	first, err := manifestDigest(manifest)
	require.NoError(t, err)
	manifest.SelectedVersions = []string{"v2.0.0"}
	second, err := manifestDigest(manifest)
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
}
