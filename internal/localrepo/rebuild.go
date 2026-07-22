package localrepo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const manifestSchema = 1

const (
	checksumFieldCount       = 2
	packageMetadataFields    = 3
	packageFormatsPerRelease = 2
	formatDEB                = "deb"
	formatRPM                = "rpm"
)

var versionPattern = regexp.MustCompile(
	`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`,
)

// RebuildRequest describes a deterministic fixture-release-set rebuild.
type RebuildRequest struct {
	// RegistryPath identifies the YAML project registry.
	RegistryPath string
	// Project selects one registry entry.
	Project string
	// ReleasesDir contains one child directory per v-prefixed release tag.
	ReleasesDir string
	// Root is the candidate tree to create or verify as a no-op.
	Root string
	// GNUPGHome contains the signing keyring used by a changed rebuild.
	GNUPGHome string
	// SigningKey is the full fingerprint of the signing subkey.
	SigningKey string
	// SigningPassphrase unlocks the signing subkey through GPG standard input.
	SigningPassphrase string
	// SigningPassphraseFile is an optional mode-0600 GPG passphrase file.
	SigningPassphraseFile string
	// BaseURL is the public root URL rendered into install configuration.
	BaseURL string
}

// RebuildResult summarizes a deterministic rebuild or verified no-op.
type RebuildResult struct {
	// Project is the selected registry key.
	Project string `json:"project"`
	// Root is the generated or verified candidate-tree directory.
	Root string `json:"root"`
	// SelectedVersions lists retained releases from newest to oldest.
	SelectedVersions []string `json:"selected_versions"`
	// DesiredStateDigest identifies the logical candidate state.
	DesiredStateDigest string `json:"desired_state_digest"`
	// NoOp reports whether an existing verified tree already matched the desired state.
	NoOp bool `json:"no_op"`
}

// Manifest records the reconstructable logical state of a candidate tree.
type Manifest struct {
	// Schema identifies the manifest format.
	Schema int `json:"schema"`
	// RegistrySHA256 identifies the exact registry input.
	RegistrySHA256 string `json:"registry_sha256"`
	// Project is the selected registry key.
	Project string `json:"project"`
	// Retention is the number of qualifying versions selected.
	Retention int `json:"retention"`
	// BaseURL is the public root URL embedded in generated configuration.
	BaseURL string `json:"base_url"`
	// SigningKey is the fingerprint used for metadata signatures.
	SigningKey string `json:"signing_key"`
	// SelectedVersions lists retained releases from newest to oldest.
	SelectedVersions []string `json:"selected_versions"`
	// Packages records every selected and validated package asset.
	Packages []PackageRecord `json:"packages"`
	// DesiredStateDigest identifies all logical inputs except signature timestamps.
	DesiredStateDigest string `json:"desired_state_digest"`
}

// PackageRecord describes one selected release asset.
type PackageRecord struct {
	// Tag is the v-prefixed source release version.
	Tag string `json:"tag"`
	// Format is either deb or rpm.
	Format string `json:"format"`
	// File is the release asset file name.
	File string `json:"file"`
	// PackageName is the identity read from package metadata.
	PackageName string `json:"package_name"`
	// Version is the version read from package metadata.
	Version string `json:"version"`
	// Architecture is the architecture read from package metadata.
	Architecture string `json:"architecture"`
	// RepositoryArchitecture is the architecture exposed by repository metadata.
	RepositoryArchitecture string `json:"repository_architecture,omitempty"`
	// SHA256 is the verified package digest.
	SHA256 string `json:"sha256"`
}

type fixtureRelease struct {
	tag     string
	version [3]int
	dir     string
	assets  []candidateAsset
	records []PackageRecord
}

// Rebuild creates a retained, verified candidate from fixture release sets.
//
// When Root already contains the same logical manifest, Rebuild verifies its
// signatures and returns a no-op without regenerating metadata or signatures.
func Rebuild(ctx context.Context, request RebuildRequest) (RebuildResult, error) {
	if err := request.validate(); err != nil {
		return RebuildResult{}, err
	}

	project, registryContent, err := loadRebuildProject(request.RegistryPath, request.Project)
	if err != nil {
		return RebuildResult{}, err
	}
	instance := builder{runner: commandRunner{}}
	releases, err := instance.selectReleases(ctx, request.ReleasesDir, project)
	if err != nil {
		return RebuildResult{}, err
	}
	manifest, err := newManifest(request, project, registryContent, releases)
	if err != nil {
		return RebuildResult{}, err
	}

	if _, statErr := os.Stat(request.Root); statErr == nil {
		return instance.verifyNoOp(ctx, request.Root, manifest)
	} else if !os.IsNotExist(statErr) {
		return RebuildResult{}, fmt.Errorf("inspect candidate root: %w", statErr)
	}

	assets := make([]candidateAsset, 0, len(releases)*packageFormatsPerRelease)
	for _, release := range releases {
		assets = append(assets, release.assets...)
	}
	buildRequest := Request{
		RegistryPath:          request.RegistryPath,
		Project:               request.Project,
		ReleaseDir:            request.ReleasesDir,
		Root:                  request.Root,
		GNUPGHome:             request.GNUPGHome,
		SigningKey:            request.SigningKey,
		SigningPassphrase:     request.SigningPassphrase,
		SigningPassphraseFile: request.SigningPassphraseFile,
		BaseURL:               request.BaseURL,
	}
	if _, err := instance.buildCandidate(ctx, buildRequest, project, assets); err != nil {
		return RebuildResult{}, err
	}
	if err := writeManifest(request.Root, manifest); err != nil {
		return RebuildResult{}, cleanupFailedBuild(request.Root, err)
	}

	return rebuildResult(request.Root, manifest, false), nil
}

func (request RebuildRequest) validate() error {
	fields := map[string]string{
		"registry":    request.RegistryPath,
		"project":     request.Project,
		"releases":    request.ReleasesDir,
		"root":        request.Root,
		"gnupg-home":  request.GNUPGHome,
		"signing-key": request.SigningKey,
		"base-url":    request.BaseURL,
	}
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if err := validatePassphraseFile(request.SigningPassphraseFile); err != nil {
		return err
	}

	return nil
}

func loadRebuildProject(path string, name string) (projectConfig, []byte, error) {
	config, content, err := loadRegistry(path)
	if err != nil {
		return projectConfig{}, nil, err
	}
	project, ok := config.Projects[name]
	if !ok {
		return projectConfig{}, nil, fmt.Errorf("project %q is not registered", name)
	}
	if project.Retention == 0 {
		project.Retention = config.Defaults.Retention
	}
	if project.Retention == 0 {
		project.Retention = 5
	}
	if err := project.validate(name); err != nil {
		return projectConfig{}, nil, err
	}
	if strings.TrimSpace(project.Assets.Checksums) == "" {
		return projectConfig{}, nil, fmt.Errorf("project %q field assets.checksums is required for rebuild", name)
	}

	return project, content, nil
}

func (instance builder) selectReleases(
	ctx context.Context,
	releasesDir string,
	project projectConfig,
) ([]fixtureRelease, error) {
	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		return nil, fmt.Errorf("read fixture releases: %w", err)
	}
	releases := make([]fixtureRelease, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		version, err := parseVersion(entry.Name())
		if err != nil {
			return nil, err
		}
		releases = append(releases, fixtureRelease{
			tag:     entry.Name(),
			version: version,
			dir:     filepath.Join(releasesDir, entry.Name()),
		})
	}
	if len(releases) == 0 {
		return nil, errors.New("fixture releases must contain at least one v-prefixed semantic version directory")
	}
	sort.Slice(releases, func(left, right int) bool {
		comparison := compareVersion(releases[left].version, releases[right].version)
		if comparison != 0 {
			return comparison > 0
		}

		return releases[left].tag > releases[right].tag
	})
	if len(releases) > project.Retention {
		releases = releases[:project.Retention]
	}

	for index := range releases {
		if err := instance.validateRelease(ctx, project, &releases[index]); err != nil {
			return nil, fmt.Errorf("validate release %s: %w", releases[index].tag, err)
		}
	}

	return releases, nil
}

func (instance builder) validateRelease(
	ctx context.Context,
	project projectConfig,
	release *fixtureRelease,
) error {
	checksumPath, err := resolveOne(release.dir, ExpandAssetPattern(project.Assets.Checksums, release.tag))
	if err != nil {
		return err
	}
	checksums, err := readChecksums(checksumPath)
	if err != nil {
		return err
	}
	if len(project.Architectures) == 0 {
		return instance.validateLegacyRelease(ctx, project, release, checksums)
	}

	debPaths, err := resolveMany(
		release.dir,
		ExpandAssetPattern(project.Assets.DEB, release.tag),
		len(project.Architectures),
	)
	if err != nil {
		return err
	}
	rpmPaths, err := resolveMany(
		release.dir,
		ExpandAssetPattern(project.Assets.RPM, release.tag),
		len(project.Architectures),
	)
	if err != nil {
		return err
	}

	debRecords, err := instance.inspectArchitectureRecords(
		ctx,
		project,
		release,
		checksums,
		formatDEB,
		debPaths,
	)
	if err != nil {
		return err
	}
	rpmRecords, err := instance.inspectArchitectureRecords(
		ctx,
		project,
		release,
		checksums,
		formatRPM,
		rpmPaths,
	)
	if err != nil {
		return err
	}

	return assembleArchitectureAssets(release, project, debRecords, rpmRecords)
}

func (instance builder) inspectArchitectureRecords(
	ctx context.Context,
	project projectConfig,
	release *fixtureRelease,
	checksums map[string]string,
	format string,
	paths []string,
) (map[string]PackageRecord, error) {
	records := make(map[string]PackageRecord, len(paths))
	for _, packagePath := range paths {
		var record PackageRecord
		var err error
		switch format {
		case formatDEB:
			record, err = instance.inspectDEB(
				ctx,
				release.tag,
				packagePath,
				project.PackageName,
				checksums,
			)
		case formatRPM:
			record, err = instance.inspectRPM(
				ctx,
				release.tag,
				packagePath,
				project.PackageName,
				checksums,
			)
		default:
			return nil, fmt.Errorf("unsupported package format %q", format)
		}
		if err != nil {
			return nil, err
		}
		architecture, err := repositoryArchitecture(project, format, record.Architecture)
		if err != nil {
			return nil, err
		}
		if _, exists := records[architecture]; exists {
			return nil, fmt.Errorf(
				"multiple %s assets map to repository architecture %s",
				strings.ToUpper(format),
				architecture,
			)
		}
		record.RepositoryArchitecture = architecture
		records[architecture] = record
	}

	return records, nil
}

func assembleArchitectureAssets(
	release *fixtureRelease,
	project projectConfig,
	debRecords map[string]PackageRecord,
	rpmRecords map[string]PackageRecord,
) error {
	architectures := make([]string, 0, len(project.Architectures))
	for architecture := range project.Architectures {
		architectures = append(architectures, architecture)
	}
	sort.Strings(architectures)
	for _, architecture := range architectures {
		debRecord, debOK := debRecords[architecture]
		rpmRecord, rpmOK := rpmRecords[architecture]
		if !debOK || !rpmOK {
			return fmt.Errorf("release is missing the DEB or RPM for repository architecture %s", architecture)
		}
		release.assets = append(release.assets, candidateAsset{
			debPath:                   filepath.Join(release.dir, debRecord.File),
			debRepositoryArchitecture: architecture,
			rpmPath:                   filepath.Join(release.dir, rpmRecord.File),
			rpmArchitecture:           rpmRecord.Architecture,
		})
		release.records = append(release.records, debRecord, rpmRecord)
	}

	return nil
}

func (instance builder) validateLegacyRelease(
	ctx context.Context,
	project projectConfig,
	release *fixtureRelease,
	checksums map[string]string,
) error {
	debPath, err := resolveOne(release.dir, ExpandAssetPattern(project.Assets.DEB, release.tag))
	if err != nil {
		return err
	}
	rpmPath, err := resolveOne(release.dir, ExpandAssetPattern(project.Assets.RPM, release.tag))
	if err != nil {
		return err
	}
	debRecord, err := instance.inspectDEB(ctx, release.tag, debPath, project.PackageName, checksums)
	if err != nil {
		return err
	}
	rpmRecord, err := instance.inspectRPM(ctx, release.tag, rpmPath, project.PackageName, checksums)
	if err != nil {
		return err
	}
	release.assets = []candidateAsset{{
		debPath:         debPath,
		rpmPath:         rpmPath,
		rpmArchitecture: rpmRecord.Architecture,
	}}
	release.records = []PackageRecord{debRecord, rpmRecord}

	return nil
}

func repositoryArchitecture(project projectConfig, format string, packageArchitecture string) (string, error) {
	for architecture, mapping := range project.Architectures {
		var expected string
		switch format {
		case formatDEB:
			expected = mapping.DEB
		case formatRPM:
			expected = mapping.RPM
		default:
			return "", fmt.Errorf("unsupported package format %q", format)
		}
		if packageArchitecture == expected {
			return architecture, nil
		}
	}

	return "", fmt.Errorf("%s architecture %q is not registered", strings.ToUpper(format), packageArchitecture)
}

func resolveOne(dir string, pattern string) (string, error) {
	matches, err := resolveMany(dir, pattern, 1)
	if err != nil {
		return "", err
	}

	return matches[0], nil
}

func resolveMany(dir string, pattern string, expected int) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return nil, fmt.Errorf("match release asset %q: %w", pattern, err)
	}
	if len(matches) != expected {
		return nil, fmt.Errorf("asset pattern %q matched %d files, expected %d", pattern, len(matches), expected)
	}
	for _, match := range matches {
		info, statErr := os.Lstat(match)
		if statErr != nil {
			return nil, fmt.Errorf("inspect release asset %s: %w", filepath.Base(match), statErr)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("release asset %s must be a regular file", filepath.Base(match))
		}
	}

	return matches, nil
}

func readChecksums(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read checksums: %w", err)
	}
	checksums := make(map[string]string)
	for lineNumber, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != checksumFieldCount {
			return nil, fmt.Errorf("parse checksums line %d", lineNumber+1)
		}
		digest := strings.ToLower(fields[0])
		if len(digest) != sha256.Size*2 {
			return nil, fmt.Errorf("checksum for %s must be SHA-256", fields[1])
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return nil, fmt.Errorf("checksum for %s must be hexadecimal: %w", fields[1], err)
		}
		name := strings.TrimPrefix(fields[1], "*")
		if filepath.Base(name) != name {
			return nil, fmt.Errorf("checksum file name %q must not contain a path", name)
		}
		if _, exists := checksums[name]; exists {
			return nil, fmt.Errorf("duplicate checksum for %s", name)
		}
		checksums[name] = digest
	}

	return checksums, nil
}

func (instance builder) inspectDEB(
	ctx context.Context,
	tag string,
	path string,
	wantName string,
	checksums map[string]string,
) (PackageRecord, error) {
	fields := make([]string, 0, packageMetadataFields)
	for _, field := range []string{"Package", "Version", "Architecture"} {
		output, err := instance.runner.output(ctx, "", nil, "dpkg-deb", "--field", path, field)
		if err != nil {
			return PackageRecord{}, err
		}
		fields = append(fields, strings.TrimSpace(string(output)))
	}

	return validatePackageRecord(tag, formatDEB, path, fields[0], fields[1], fields[2], wantName, checksums)
}

func (instance builder) inspectRPM(
	ctx context.Context,
	tag string,
	path string,
	wantName string,
	checksums map[string]string,
) (PackageRecord, error) {
	output, err := instance.runner.output(
		ctx,
		"",
		nil,
		"rpm",
		"-qp",
		"--qf",
		"%{NAME}\n%{VERSION}\n%{ARCH}\n",
		path,
	)
	if err != nil {
		return PackageRecord{}, err
	}
	fields := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(fields) != packageMetadataFields {
		return PackageRecord{}, fmt.Errorf("inspect RPM metadata: expected three fields, got %d", len(fields))
	}

	return validatePackageRecord(tag, formatRPM, path, fields[0], fields[1], fields[2], wantName, checksums)
}

func validatePackageRecord(
	tag string,
	format string,
	path string,
	name string,
	version string,
	architecture string,
	wantName string,
	checksums map[string]string,
) (PackageRecord, error) {
	file := filepath.Base(path)
	if name != wantName {
		return PackageRecord{}, fmt.Errorf("package %s name is %q, expected %q", file, name, wantName)
	}
	if version != strings.TrimPrefix(tag, "v") {
		return PackageRecord{}, fmt.Errorf(
			"package %s version is %q, expected %q",
			file,
			version,
			strings.TrimPrefix(tag, "v"),
		)
	}
	if strings.TrimSpace(architecture) == "" {
		return PackageRecord{}, fmt.Errorf("package %s architecture is empty", file)
	}
	digest, err := digestFile(path)
	if err != nil {
		return PackageRecord{}, err
	}
	wantDigest, ok := checksums[file]
	if !ok {
		return PackageRecord{}, fmt.Errorf("checksums do not cover %s", file)
	}
	if digest != wantDigest {
		return PackageRecord{}, fmt.Errorf("checksum mismatch for %s", file)
	}

	return PackageRecord{
		Tag:          tag,
		Format:       format,
		File:         file,
		PackageName:  name,
		Version:      version,
		Architecture: architecture,
		SHA256:       digest,
	}, nil
}

func digestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("read %s for checksum: %w", filepath.Base(path), err)
	}
	defer file.Close()

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", fmt.Errorf("checksum %s: %w", filepath.Base(path), err)
	}

	return hex.EncodeToString(digest.Sum(nil)), nil
}

func parseVersion(tag string) ([3]int, error) {
	matches := versionPattern.FindStringSubmatch(tag)
	if matches == nil {
		return [3]int{}, fmt.Errorf("release tag %q must be a stable v-prefixed semantic version", tag)
	}
	var version [3]int
	for index := range version {
		value, err := strconv.Atoi(matches[index+1])
		if err != nil {
			return [3]int{}, fmt.Errorf("parse release version %q: %w", tag, err)
		}
		version[index] = value
	}

	return version, nil
}

func compareVersion(left [3]int, right [3]int) int {
	for index := range left {
		if left[index] < right[index] {
			return -1
		}
		if left[index] > right[index] {
			return 1
		}
	}

	return 0
}

func newManifest(
	request RebuildRequest,
	project projectConfig,
	registryContent []byte,
	releases []fixtureRelease,
) (Manifest, error) {
	manifest := Manifest{
		Schema:           manifestSchema,
		RegistrySHA256:   digestSHA256(registryContent),
		Project:          request.Project,
		Retention:        project.Retention,
		BaseURL:          strings.TrimRight(request.BaseURL, "/"),
		SigningKey:       request.SigningKey,
		SelectedVersions: make([]string, 0, len(releases)),
		Packages:         make([]PackageRecord, 0, len(releases)*packageFormatsPerRelease),
	}
	for _, release := range releases {
		manifest.SelectedVersions = append(manifest.SelectedVersions, release.tag)
		manifest.Packages = append(manifest.Packages, release.records...)
	}
	digest, err := manifestDigest(manifest)
	if err != nil {
		return Manifest{}, err
	}
	manifest.DesiredStateDigest = digest

	return manifest, nil
}

func writeManifest(root string, manifest Manifest) error {
	directory := filepath.Join(root, "_state")
	if err := os.MkdirAll(directory, directoryMode); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state manifest: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), content, publicFileMode); err != nil {
		return fmt.Errorf("write state manifest: %w", err)
	}

	return nil
}

func (instance builder) verifyNoOp(
	ctx context.Context,
	root string,
	want Manifest,
) (RebuildResult, error) {
	content, err := os.ReadFile(filepath.Join(root, "_state", "manifest.json"))
	if err != nil {
		return RebuildResult{}, fmt.Errorf("read existing state manifest: %w", err)
	}
	var got Manifest
	if decodeErr := json.Unmarshal(content, &got); decodeErr != nil {
		return RebuildResult{}, fmt.Errorf("parse existing state manifest: %w", decodeErr)
	}
	gotDigest, err := manifestDigest(got)
	if err != nil {
		return RebuildResult{}, err
	}
	if gotDigest != got.DesiredStateDigest {
		return RebuildResult{}, fmt.Errorf(
			"existing state manifest digest is %s, computed %s",
			got.DesiredStateDigest,
			gotDigest,
		)
	}
	if got.DesiredStateDigest != want.DesiredStateDigest {
		return RebuildResult{}, fmt.Errorf(
			"candidate root exists with desired state %s, requested %s",
			got.DesiredStateDigest,
			want.DesiredStateDigest,
		)
	}
	if err := verifyManifestPackages(root, want); err != nil {
		return RebuildResult{}, fmt.Errorf("verify existing candidate packages: %w", err)
	}
	if err := instance.verifyRoot(ctx, root); err != nil {
		return RebuildResult{}, fmt.Errorf("verify existing candidate: %w", err)
	}

	return rebuildResult(root, want, true), nil
}

func manifestDigest(manifest Manifest) (string, error) {
	manifest.DesiredStateDigest = ""
	content, err := json.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("encode desired state: %w", err)
	}

	return digestSHA256(content), nil
}

func verifyManifestPackages(root string, manifest Manifest) error {
	for _, record := range manifest.Packages {
		var path string
		switch record.Format {
		case formatDEB:
			path = filepath.Join(root, "apt", "pool", manifest.Project, record.RepositoryArchitecture, record.File)
		case formatRPM:
			path = filepath.Join(root, "rpm", manifest.Project, record.Architecture, record.File)
		default:
			return fmt.Errorf("unsupported package format %q", record.Format)
		}
		digest, err := digestFile(path)
		if err != nil {
			return err
		}
		if digest != record.SHA256 {
			return fmt.Errorf("checksum mismatch for retained package %s", record.File)
		}
	}

	return nil
}

func rebuildResult(root string, manifest Manifest, noOp bool) RebuildResult {
	return RebuildResult{
		Project:            manifest.Project,
		Root:               root,
		SelectedVersions:   manifest.SelectedVersions,
		DesiredStateDigest: manifest.DesiredStateDigest,
		NoOp:               noOp,
	}
}
