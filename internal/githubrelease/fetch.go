// Package githubrelease downloads and verifies registered GitHub Release assets.
package githubrelease

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/meigma/packages/internal/localrepo"
)

const (
	defaultAPIBaseURL  = "https://api.github.com"
	maxAPIResponse     = 4 << 20
	assetStateUploaded = "uploaded"
)

// Request describes one registered GitHub Release download.
type Request struct {
	// RegistryPath identifies the canonical YAML project registry.
	RegistryPath string
	// Project selects one registry entry.
	Project string
	// Tag is the stable v-prefixed release to fetch.
	Tag string
	// OutputDir is the new directory that receives verified source assets.
	OutputDir string
	// Token optionally authenticates GitHub API and attestation requests.
	Token string
	// APIBaseURL overrides the GitHub API endpoint for tests or GitHub Enterprise.
	APIBaseURL string
}

// Result summarizes one atomic verified release download.
type Result struct {
	// Project is the selected registry key.
	Project string `json:"project"`
	// Repository is the selected GitHub owner/name repository.
	Repository string `json:"repository"`
	// Tag is the verified stable source release tag.
	Tag string `json:"tag"`
	// OutputDir contains the downloaded checksum and package assets.
	OutputDir string `json:"output_dir"`
	// Assets lists every downloaded asset and its GitHub-recorded digest.
	Assets []Asset `json:"assets"`
}

// Asset describes one downloaded GitHub Release asset.
type Asset struct {
	// Name is the release asset file name.
	Name string `json:"name"`
	// SHA256 is the digest verified against GitHub's release metadata.
	SHA256 string `json:"sha256"`
	// Size is the verified number of downloaded bytes.
	Size int64 `json:"size"`
}

type apiRelease struct {
	TagName     string     `json:"tag_name"`
	Draft       bool       `json:"draft"`
	Prerelease  bool       `json:"prerelease"`
	PublishedAt *time.Time `json:"published_at"`
	Assets      []apiAsset `json:"assets"`
}

type apiAsset struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	State  string `json:"state"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
	URL    string `json:"url"`
}

type assetPattern struct {
	value    string
	expected int
}

type provenanceVerifier interface {
	Verify(context.Context, string, localrepo.ReleaseSource, string, string) error
}

type client struct {
	httpClient *http.Client
	verifier   provenanceVerifier
}

// Fetch downloads a published release and verifies its assets and provenance.
func Fetch(ctx context.Context, request Request) (Result, error) {
	instance := client{httpClient: http.DefaultClient, verifier: ghVerifier{}}

	return instance.fetch(ctx, request)
}

func (instance client) fetch(ctx context.Context, request Request) (Result, error) {
	if err := request.validate(); err != nil {
		return Result{}, err
	}
	if _, err := localrepo.ValidateRequest(request.RegistryPath, request.Project, request.Tag); err != nil {
		return Result{}, err
	}
	source, err := localrepo.LoadReleaseSource(request.RegistryPath, request.Project)
	if err != nil {
		return Result{}, err
	}

	apiBaseURL := strings.TrimRight(request.APIBaseURL, "/")
	if apiBaseURL == "" {
		apiBaseURL = defaultAPIBaseURL
	}
	release, err := instance.getRelease(ctx, apiBaseURL, source.Repository, request.Tag, request.Token)
	if err != nil {
		return Result{}, err
	}
	assets, err := selectAssets(release, source, request.Tag)
	if err != nil {
		return Result{}, err
	}

	parent := filepath.Dir(request.OutputDir)
	if mkdirErr := os.MkdirAll(parent, 0o750); mkdirErr != nil {
		return Result{}, fmt.Errorf("create release output parent: %w", mkdirErr)
	}
	temporary, err := os.MkdirTemp(parent, ".release-download-*")
	if err != nil {
		return Result{}, fmt.Errorf("create release download directory: %w", err)
	}
	defer os.RemoveAll(temporary)

	resultAssets := make([]Asset, 0, len(assets))
	for _, asset := range assets {
		destination := filepath.Join(temporary, asset.Name)
		digest, downloadErr := instance.downloadAsset(
			ctx,
			apiBaseURL,
			source.Repository,
			asset,
			destination,
			request.Token,
		)
		if downloadErr != nil {
			return Result{}, downloadErr
		}
		resultAssets = append(resultAssets, Asset{Name: asset.Name, SHA256: digest, Size: asset.Size})
	}

	for _, asset := range assets {
		if asset.Name == source.Checksums {
			continue
		}
		if err := instance.verifier.Verify(
			ctx,
			filepath.Join(temporary, asset.Name),
			source,
			request.Tag,
			request.Token,
		); err != nil {
			return Result{}, err
		}
	}
	if err := os.Rename(temporary, request.OutputDir); err != nil {
		return Result{}, fmt.Errorf("publish verified release directory: %w", err)
	}
	sort.Slice(resultAssets, func(left, right int) bool {
		return resultAssets[left].Name < resultAssets[right].Name
	})

	return Result{
		Project:    request.Project,
		Repository: source.Repository,
		Tag:        request.Tag,
		OutputDir:  request.OutputDir,
		Assets:     resultAssets,
	}, nil
}

func (request Request) validate() error {
	fields := map[string]string{
		"registry": request.RegistryPath,
		"project":  request.Project,
		"tag":      request.Tag,
		"output":   request.OutputDir,
	}
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if _, err := os.Stat(request.OutputDir); err == nil {
		return fmt.Errorf("release output already exists: %s", request.OutputDir)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect release output: %w", err)
	}

	return nil
}

func (instance client) getRelease(
	ctx context.Context,
	apiBaseURL string,
	repository string,
	tag string,
	token string,
) (apiRelease, error) {
	endpoint := fmt.Sprintf(
		"%s/repos/%s/releases/tags/%s",
		apiBaseURL,
		repository,
		url.PathEscape(tag),
	)
	request, err := newAPIRequest(ctx, http.MethodGet, endpoint, token, "application/vnd.github+json")
	if err != nil {
		return apiRelease{}, err
	}
	response, err := instance.httpClient.Do(request)
	if err != nil {
		return apiRelease{}, fmt.Errorf("fetch GitHub release: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return apiRelease{}, fmt.Errorf("fetch GitHub release: unexpected HTTP status %s", response.Status)
	}
	var release apiRelease
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxAPIResponse))
	if err := decoder.Decode(&release); err != nil {
		return apiRelease{}, fmt.Errorf("decode GitHub release: %w", err)
	}

	return release, nil
}

func selectAssets(release apiRelease, source localrepo.ReleaseSource, tag string) ([]apiAsset, error) {
	if release.TagName != tag {
		return nil, fmt.Errorf("GitHub release tag is %q, expected %q", release.TagName, tag)
	}
	if release.Draft || release.Prerelease || release.PublishedAt == nil {
		return nil, errors.New("GitHub release must be published and must not be a draft or prerelease")
	}

	patterns := []assetPattern{
		{value: source.Checksums, expected: 1},
		{
			value:    localrepo.ExpandAssetPattern(source.DEBPattern, tag),
			expected: len(source.Architectures),
		},
		{
			value:    localrepo.ExpandAssetPattern(source.RPMPattern, tag),
			expected: len(source.Architectures),
		},
	}
	selected := make(map[string]apiAsset)
	for _, pattern := range patterns {
		matched, err := assetsMatchingPattern(release.Assets, pattern.value)
		if err != nil {
			return nil, err
		}
		if len(matched) != pattern.expected {
			return nil, fmt.Errorf(
				"release asset pattern %q matched %d assets, expected %d",
				pattern.value,
				len(matched),
				pattern.expected,
			)
		}
		for _, asset := range matched {
			if _, exists := selected[asset.Name]; exists {
				return nil, fmt.Errorf(
					"release asset %q matches more than one configured pattern",
					asset.Name,
				)
			}
			selected[asset.Name] = asset
		}
	}

	assets := make([]apiAsset, 0, len(selected))
	for _, asset := range selected {
		assets = append(assets, asset)
	}
	sort.Slice(assets, func(left, right int) bool { return assets[left].Name < assets[right].Name })

	return assets, nil
}

func assetsMatchingPattern(assets []apiAsset, pattern string) ([]apiAsset, error) {
	matched := make([]apiAsset, 0)
	for _, asset := range assets {
		matches, err := path.Match(pattern, asset.Name)
		if err != nil {
			return nil, fmt.Errorf("match release asset pattern %q: %w", pattern, err)
		}
		if !matches {
			continue
		}
		if filepath.Base(asset.Name) != asset.Name || asset.State != assetStateUploaded || asset.Size <= 0 {
			return nil, fmt.Errorf("release asset %q is not a complete regular asset", asset.Name)
		}
		matched = append(matched, asset)
	}

	return matched, nil
}

func (instance client) downloadAsset(
	ctx context.Context,
	apiBaseURL string,
	repository string,
	asset apiAsset,
	destination string,
	token string,
) (string, error) {
	wantDigest, err := parseDigest(asset.Digest)
	if err != nil {
		return "", fmt.Errorf("release asset %s: %w", asset.Name, err)
	}
	if validationErr := validateAssetURL(apiBaseURL, repository, asset); validationErr != nil {
		return "", validationErr
	}
	request, err := newAPIRequest(ctx, http.MethodGet, asset.URL, token, "application/octet-stream")
	if err != nil {
		return "", err
	}
	response, err := instance.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("download release asset %s: %w", asset.Name, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download release asset %s: unexpected HTTP status %s", asset.Name, response.Status)
	}

	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create release asset %s: %w", asset.Name, err)
	}
	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hasher), io.LimitReader(response.Body, asset.Size+1))
	closeErr := file.Close()
	if copyErr != nil {
		return "", fmt.Errorf("download release asset %s: %w", asset.Name, copyErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("close release asset %s: %w", asset.Name, closeErr)
	}
	if written != asset.Size {
		return "", fmt.Errorf("release asset %s size is %d bytes, expected %d", asset.Name, written, asset.Size)
	}
	digest := hex.EncodeToString(hasher.Sum(nil))
	if digest != wantDigest {
		return "", fmt.Errorf("release asset %s digest does not match GitHub metadata", asset.Name)
	}

	return digest, nil
}

func validateAssetURL(apiBaseURL string, repository string, asset apiAsset) error {
	base, err := url.Parse(apiBaseURL)
	if err != nil {
		return fmt.Errorf("parse GitHub API URL: %w", err)
	}
	assetURL, err := url.Parse(asset.URL)
	if err != nil {
		return fmt.Errorf("parse release asset URL: %w", err)
	}
	wantPrefix := fmt.Sprintf("/repos/%s/releases/assets/", repository)
	if assetURL.Scheme != base.Scheme || assetURL.Host != base.Host || !strings.HasPrefix(assetURL.Path, wantPrefix) {
		return fmt.Errorf("release asset %s has an unexpected API URL", asset.Name)
	}

	return nil
}

func parseDigest(value string) (string, error) {
	digest := strings.TrimPrefix(value, "sha256:")
	if digest == value || len(digest) != sha256.Size*2 {
		return "", errors.New("GitHub metadata must include a SHA-256 digest")
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return "", fmt.Errorf("GitHub SHA-256 digest must be hexadecimal: %w", err)
	}

	return strings.ToLower(digest), nil
}

func newAPIRequest(
	ctx context.Context,
	method string,
	endpoint string,
	token string,
	accept string,
) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create GitHub API request: %w", err)
	}
	request.Header.Set("Accept", accept)
	request.Header.Set("User-Agent", "meigma-packages")
	request.Header.Set("X-Github-Api-Version", "2022-11-28")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	return request, nil
}

type ghVerifier struct{}

func (ghVerifier) Verify(
	ctx context.Context,
	assetPath string,
	source localrepo.ReleaseSource,
	tag string,
	token string,
) error {
	arguments := []string{
		"attestation", "verify", assetPath,
		"--repo", source.Repository,
		"--signer-workflow", source.SignerWorkflow,
		"--source-ref", "refs/tags/" + tag,
		"--deny-self-hosted-runners",
	}
	command := exec.CommandContext(ctx, "gh", arguments...)
	command.Env = os.Environ()
	if token != "" {
		command.Env = append(command.Env, "GH_TOKEN="+token)
	}
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"verify SLSA provenance for %s: %w: %s",
			filepath.Base(assetPath),
			err,
			strings.TrimSpace(string(output)),
		)
	}

	return nil
}
