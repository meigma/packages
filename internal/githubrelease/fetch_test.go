package githubrelease

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meigma/packages/internal/localrepo"
)

const testProjectRegistry = `schema: 1
projects:
  incus-gh-runner:
    repository: meigma/incus-gh-runner
    package_name: incus-gh-runner
    assets:
      checksums: checksums.txt
      deb: 'incus-gh-runner_${version}_*.deb'
      rpm: 'incus-gh-runner-${version}-1.*.rpm'
    architectures:
      amd64:
        deb: amd64
        rpm: x86_64
      arm64:
        deb: arm64
        rpm: aarch64
    provenance:
      signer_workflow: meigma/incus-gh-runner/.github/workflows/attest.yml
`

type verifierCall struct {
	assetPath string
	source    localrepo.ReleaseSource
	tag       string
	token     string
}

type recordingVerifier struct {
	calls []verifierCall
	err   error
}

func (verifier *recordingVerifier) Verify(
	_ context.Context,
	assetPath string,
	source localrepo.ReleaseSource,
	tag string,
	token string,
) error {
	verifier.calls = append(verifier.calls, verifierCall{
		assetPath: assetPath,
		source:    source,
		tag:       tag,
		token:     token,
	})

	return verifier.err
}

func TestFetchDownloadsAndVerifiesRegisteredRelease(t *testing.T) {
	t.Parallel()

	contents := testReleaseContents()
	server := newReleaseServer(t, contents, "")
	defer server.Close()

	registryPath := writeTestRegistry(t)
	outputDir := filepath.Join(t.TempDir(), "releases", "v1.0.0")
	verifier := &recordingVerifier{}
	instance := client{httpClient: server.Client(), verifier: verifier}

	result, err := instance.fetch(t.Context(), Request{
		RegistryPath: registryPath,
		Project:      "incus-gh-runner",
		Tag:          "v1.0.0",
		OutputDir:    outputDir,
		Token:        "test-token",
		APIBaseURL:   server.URL,
	})

	require.NoError(t, err)
	assert.Equal(t, "meigma/incus-gh-runner", result.Repository)
	assert.Equal(t, "v1.0.0", result.Tag)
	assert.Len(t, result.Assets, len(contents))
	assert.Len(t, verifier.calls, 4)
	for _, call := range verifier.calls {
		assert.Equal(t, "meigma/incus-gh-runner", call.source.Repository)
		assert.Equal(t, "v1.0.0", call.tag)
		assert.Equal(t, "test-token", call.token)
		assert.NotEqual(t, "checksums.txt", filepath.Base(call.assetPath))
	}
	for name, content := range contents {
		got, readErr := os.ReadFile(filepath.Join(outputDir, name))
		require.NoError(t, readErr)
		assert.Equal(t, content, got)
	}
	assetNames := make([]string, 0, len(result.Assets))
	for _, asset := range result.Assets {
		assetNames = append(assetNames, asset.Name)
	}
	assert.True(t, sort.StringsAreSorted(assetNames))
}

func TestFetchLeavesNoOutputWhenVerificationFails(t *testing.T) {
	t.Parallel()

	contents := testReleaseContents()
	server := newReleaseServer(t, contents, "")
	defer server.Close()

	outputDir := filepath.Join(t.TempDir(), "releases", "v1.0.0")
	instance := client{
		httpClient: server.Client(),
		verifier:   &recordingVerifier{err: assert.AnError},
	}
	_, err := instance.fetch(t.Context(), Request{
		RegistryPath: writeTestRegistry(t),
		Project:      "incus-gh-runner",
		Tag:          "v1.0.0",
		OutputDir:    outputDir,
		Token:        "test-token",
		APIBaseURL:   server.URL,
	})

	require.Error(t, err)
	require.ErrorIs(t, err, assert.AnError)
	_, statErr := os.Stat(outputDir)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestFetchRejectsGitHubDigestMismatch(t *testing.T) {
	t.Parallel()

	contents := testReleaseContents()
	server := newReleaseServer(t, contents, "incus-gh-runner_1.0.0_amd64.deb")
	defer server.Close()

	outputDir := filepath.Join(t.TempDir(), "v1.0.0")
	instance := client{httpClient: server.Client(), verifier: &recordingVerifier{}}
	_, err := instance.fetch(t.Context(), Request{
		RegistryPath: writeTestRegistry(t),
		Project:      "incus-gh-runner",
		Tag:          "v1.0.0",
		OutputDir:    outputDir,
		Token:        "test-token",
		APIBaseURL:   server.URL,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "digest does not match GitHub metadata")
	_, statErr := os.Stat(outputDir)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestSelectAssetsRejectsIncompleteReleaseSets(t *testing.T) {
	t.Parallel()

	source := localrepo.ReleaseSource{
		Checksums:  "checksums.txt",
		DEBPattern: "incus-gh-runner_${version}_*.deb",
		RPMPattern: "incus-gh-runner-${version}-1.*.rpm",
		Architectures: map[string]localrepo.ReleaseArchitecture{
			"amd64": {DEB: "amd64", RPM: "x86_64"},
			"arm64": {DEB: "arm64", RPM: "aarch64"},
		},
	}
	publishedAt := time.Now()
	release := apiRelease{
		TagName:     "v1.0.0",
		PublishedAt: &publishedAt,
		Assets: []apiAsset{
			{Name: "checksums.txt", State: assetStateUploaded, Size: 1},
			{Name: "incus-gh-runner_1.0.0_amd64.deb", State: assetStateUploaded, Size: 1},
			{Name: "incus-gh-runner_1.0.0_arm64.deb", State: assetStateUploaded, Size: 1},
			{Name: "incus-gh-runner-1.0.0-1.x86_64.rpm", State: assetStateUploaded, Size: 1},
		},
	}

	_, err := selectAssets(release, source, "v1.0.0")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "matched 1 assets, expected 2")
}

func newReleaseServer(
	t *testing.T,
	contents map[string][]byte,
	mismatchedDigestAsset string,
) *httptest.Server {
	t.Helper()

	names := make([]string, 0, len(contents))
	for name := range contents {
		names = append(names, name)
	}
	sort.Strings(names)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "Bearer test-token", request.Header.Get("Authorization"))
		switch {
		case request.URL.Path == "/repos/meigma/incus-gh-runner/releases/tags/v1.0.0":
			assert.Equal(t, "application/vnd.github+json", request.Header.Get("Accept"))
			assets := make([]apiAsset, 0, len(names)+1)
			for index, name := range names {
				digest := digest(contents[name])
				if name == mismatchedDigestAsset {
					digest = strings.Repeat("0", sha256.Size*2)
				}
				assets = append(assets, apiAsset{
					ID:     int64(index + 1),
					Name:   name,
					State:  assetStateUploaded,
					Size:   int64(len(contents[name])),
					Digest: "sha256:" + digest,
					URL: fmt.Sprintf(
						"%s/repos/meigma/incus-gh-runner/releases/assets/%d",
						server.URL,
						index+1,
					),
				})
			}
			assets = append(assets, apiAsset{Name: "incus-gh-runner_1.0.0_linux_amd64.tar.gz"})
			writer.Header().Set("Content-Type", "application/json")
			encodeErr := json.NewEncoder(writer).Encode(apiRelease{
				TagName:     "v1.0.0",
				PublishedAt: new(time.Time),
				Assets:      assets,
			})
			assert.NoError(t, encodeErr)
		case strings.HasPrefix(request.URL.Path, "/repos/meigma/incus-gh-runner/releases/assets/"):
			assert.Equal(t, "application/octet-stream", request.Header.Get("Accept"))
			id := strings.TrimPrefix(request.URL.Path, "/repos/meigma/incus-gh-runner/releases/assets/")
			var index int
			_, err := fmt.Sscanf(id, "%d", &index)
			if !assert.NoError(t, err) || !assert.GreaterOrEqual(t, index, 1) ||
				!assert.LessOrEqual(t, index, len(names)) {
				http.Error(writer, "invalid fixture asset", http.StatusInternalServerError)

				return
			}
			_, err = writer.Write(contents[names[index-1]])
			assert.NoError(t, err)
		default:
			http.NotFound(writer, request)
		}
	}))

	return server
}

func testReleaseContents() map[string][]byte {
	return map[string][]byte{
		"checksums.txt":                       []byte("checksums"),
		"incus-gh-runner_1.0.0_amd64.deb":     []byte("deb-amd64"),
		"incus-gh-runner_1.0.0_arm64.deb":     []byte("deb-arm64"),
		"incus-gh-runner-1.0.0-1.x86_64.rpm":  []byte("rpm-amd64"),
		"incus-gh-runner-1.0.0-1.aarch64.rpm": []byte("rpm-arm64"),
	}
}

func writeTestRegistry(t *testing.T) string {
	t.Helper()

	registryPath := filepath.Join(t.TempDir(), "projects.yml")
	require.NoError(t, os.WriteFile(registryPath, []byte(testProjectRegistry), 0o644))

	return registryPath
}

func digest(content []byte) string {
	sum := sha256.Sum256(content)

	return hex.EncodeToString(sum[:])
}
