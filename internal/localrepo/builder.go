package localrepo

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	directoryMode  = 0o755
	publicFileMode = 0o644
	batchFlag      = "--batch"
	verifyFlag     = "--verify"
)

// Request describes one local fixture-to-candidate build.
type Request struct {
	// RegistryPath identifies the YAML project registry.
	RegistryPath string
	// Project selects one registry entry.
	Project string
	// ReleaseDir contains the exact fixture assets named by the registry.
	ReleaseDir string
	// Root is the new candidate-tree directory to create.
	Root string
	// GNUPGHome contains the throwaway signing key used by the local build.
	GNUPGHome string
	// SigningKey is the full fingerprint of the signing subkey.
	SigningKey string
	// BaseURL is the public root URL rendered into install configuration.
	BaseURL string
}

// Result summarizes a successfully verified candidate build.
type Result struct {
	// Project is the selected registry key.
	Project string `json:"project"`
	// PackageName is the package identity from the registry.
	PackageName string `json:"package_name"`
	// Root is the generated candidate-tree directory.
	Root string `json:"root"`
	// DEBArchitecture is the native APT index architecture.
	DEBArchitecture string `json:"deb_architecture"`
	// SigningKey is the fingerprint used for metadata signatures.
	SigningKey string `json:"signing_key"`
}

type builder struct {
	runner commandRunner
}

type candidateAsset struct {
	debPath         string
	rpmPath         string
	rpmArchitecture string
}

// Build creates and verifies a local APT/RPM candidate tree from fixture assets.
func Build(ctx context.Context, request Request) (Result, error) {
	instance := builder{runner: commandRunner{}}

	return instance.build(ctx, request)
}

func (instance builder) build(ctx context.Context, request Request) (Result, error) {
	if validationErr := request.validate(); validationErr != nil {
		return Result{}, validationErr
	}

	project, err := loadProject(request.RegistryPath, request.Project)
	if err != nil {
		return Result{}, err
	}
	assets := []candidateAsset{{
		debPath: filepath.Join(request.ReleaseDir, project.Assets.DEB),
		rpmPath: filepath.Join(request.ReleaseDir, project.Assets.RPM),
	}}

	return instance.buildCandidate(ctx, request, project, assets)
}

func (instance builder) buildCandidate(
	ctx context.Context,
	request Request,
	project projectConfig,
	assets []candidateAsset,
) (Result, error) {
	if rootErr := ensureNewRoot(request.Root); rootErr != nil {
		return Result{}, rootErr
	}

	architecture, err := instance.buildAPT(ctx, request, assets)
	if err != nil {
		return Result{}, cleanupFailedBuild(request.Root, err)
	}
	if rpmErr := instance.buildRPM(ctx, request, assets); rpmErr != nil {
		return Result{}, cleanupFailedBuild(request.Root, rpmErr)
	}
	if verifyErr := instance.exportAndVerify(ctx, request); verifyErr != nil {
		return Result{}, cleanupFailedBuild(request.Root, verifyErr)
	}

	return Result{
		Project:         request.Project,
		PackageName:     project.PackageName,
		Root:            request.Root,
		DEBArchitecture: architecture,
		SigningKey:      request.SigningKey,
	}, nil
}

func (request Request) validate() error {
	fields := map[string]string{
		"registry":    request.RegistryPath,
		"project":     request.Project,
		"release":     request.ReleaseDir,
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

	return nil
}

func (instance builder) buildAPT(
	ctx context.Context,
	request Request,
	assets []candidateAsset,
) (string, error) {
	architectureOutput, err := instance.runner.run(ctx, "", nil, "dpkg", "--print-architecture")
	if err != nil {
		return "", err
	}
	architecture := strings.TrimSpace(string(architectureOutput))
	aptRoot := filepath.Join(request.Root, "apt")
	binaryDir := filepath.Join("dists", "stable", request.Project, "binary-"+architecture)
	poolDir := filepath.Join(aptRoot, "pool", request.Project)

	if mkdirErr := os.MkdirAll(filepath.Join(aptRoot, binaryDir), directoryMode); mkdirErr != nil {
		return "", fmt.Errorf("create APT index directory: %w", mkdirErr)
	}
	for _, asset := range assets {
		if copyErr := copyAssetPath(asset.debPath, poolDir); copyErr != nil {
			return "", copyErr
		}
	}

	packages, err := instance.runner.run(
		ctx,
		aptRoot,
		nil,
		"apt-ftparchive",
		"packages",
		filepath.ToSlash(filepath.Join("pool", request.Project)),
	)
	if err != nil {
		return "", err
	}
	packagesPath := filepath.Join(aptRoot, binaryDir, "Packages")
	if err := os.WriteFile(packagesPath, packages, publicFileMode); err != nil {
		return "", fmt.Errorf("write APT Packages index: %w", err)
	}
	if err := writeGzip(packagesPath); err != nil {
		return "", err
	}
	if err := writeByHash(packagesPath); err != nil {
		return "", err
	}
	if err := writeByHash(packagesPath + ".gz"); err != nil {
		return "", err
	}
	if err := instance.writeAndSignRelease(ctx, request, aptRoot, architecture); err != nil {
		return "", err
	}

	return architecture, nil
}

func (instance builder) writeAndSignRelease(
	ctx context.Context,
	request Request,
	aptRoot string,
	architecture string,
) error {
	releaseDir := filepath.Join(aptRoot, "dists", "stable")
	release, err := instance.runner.run(
		ctx,
		aptRoot,
		nil,
		"apt-ftparchive",
		"-o", "APT::FTPArchive::Release::Origin=Meigma",
		"-o", "APT::FTPArchive::Release::Label=Meigma",
		"-o", "APT::FTPArchive::Release::Suite=stable",
		"-o", "APT::FTPArchive::Release::Codename=stable",
		"-o", "APT::FTPArchive::Release::Architectures="+architecture,
		"-o", "APT::FTPArchive::Release::Components="+request.Project,
		"-o", "APT::FTPArchive::Release::Acquire-By-Hash=yes",
		"-o", "APT::FTPArchive::Release::Description=Meigma local candidate",
		"release", filepath.ToSlash(filepath.Join("dists", "stable")),
	)
	if err != nil {
		return err
	}
	releasePath := filepath.Join(releaseDir, "Release")
	if err := os.WriteFile(releasePath, release, publicFileMode); err != nil {
		return fmt.Errorf("write APT Release: %w", err)
	}

	environment := []string{"GNUPGHOME=" + request.GNUPGHome}
	key := request.SigningKey + "!"
	if _, err := instance.runner.run(
		ctx, "", environment, "gpg", batchFlag, "--yes", "--local-user", key,
		"--armor", "--detach-sign", "--output", releasePath+".gpg", releasePath,
	); err != nil {
		return err
	}
	if _, err := instance.runner.run(
		ctx, "", environment, "gpg", batchFlag, "--yes", "--local-user", key,
		"--clearsign", "--output", filepath.Join(releaseDir, "InRelease"), releasePath,
	); err != nil {
		return err
	}

	return nil
}

func (instance builder) buildRPM(
	ctx context.Context,
	request Request,
	assets []candidateAsset,
) error {
	rpmRoot := filepath.Join(request.Root, "rpm", request.Project)
	for _, asset := range assets {
		architecture := asset.rpmArchitecture
		if architecture == "" {
			architecture = "noarch"
		}
		if err := copyAssetPath(asset.rpmPath, filepath.Join(rpmRoot, architecture)); err != nil {
			return err
		}
	}
	if _, err := instance.runner.run(ctx, "", nil, "createrepo_c", rpmRoot); err != nil {
		return err
	}

	repomdPath := filepath.Join(rpmRoot, "repodata", "repomd.xml")
	if _, err := instance.runner.run(
		ctx,
		"",
		[]string{"GNUPGHOME=" + request.GNUPGHome},
		"gpg", batchFlag, "--yes", "--local-user", request.SigningKey+"!",
		"--armor", "--detach-sign", "--output", repomdPath+".asc", repomdPath,
	); err != nil {
		return err
	}

	config := fmt.Sprintf(
		"[meigma-%s]\nname=Meigma %s\nbaseurl=%s/rpm/%s\nenabled=1\n"+
			"repo_gpgcheck=1\ngpgcheck=0\ngpgkey=%s/meigma.asc\n",
		request.Project,
		request.Project,
		strings.TrimRight(request.BaseURL, "/"),
		request.Project,
		strings.TrimRight(request.BaseURL, "/"),
	)
	if err := os.WriteFile(filepath.Join(rpmRoot, "meigma.repo"), []byte(config), publicFileMode); err != nil {
		return fmt.Errorf("write RPM repository config: %w", err)
	}

	return nil
}

func (instance builder) exportAndVerify(ctx context.Context, request Request) error {
	publicKeyPath := filepath.Join(request.Root, "meigma.asc")
	publicKey, err := instance.runner.run(
		ctx,
		"",
		[]string{"GNUPGHOME=" + request.GNUPGHome},
		"gpg", batchFlag, "--armor", "--export", request.SigningKey,
	)
	if err != nil {
		return err
	}
	if writeErr := os.WriteFile(publicKeyPath, publicKey, publicFileMode); writeErr != nil {
		return fmt.Errorf("write public key: %w", writeErr)
	}

	return instance.verifyRoot(ctx, request.Root)
}

func (instance builder) verifyRoot(ctx context.Context, root string) error {
	publicKeyPath := filepath.Join(root, "meigma.asc")
	verifyHome, err := os.MkdirTemp("", "meigma-packages-verify-*")
	if err != nil {
		return fmt.Errorf("create verification keyring: %w", err)
	}
	defer os.RemoveAll(verifyHome)
	environment := []string{"GNUPGHOME=" + verifyHome}
	if _, importErr := instance.runner.run(
		ctx,
		"",
		environment,
		"gpg",
		batchFlag,
		"--import",
		publicKeyPath,
	); importErr != nil {
		return importErr
	}

	releaseDir := filepath.Join(root, "apt", "dists", "stable")
	rpmRoots, err := filepath.Glob(filepath.Join(root, "rpm", "*", "repodata", "repomd.xml"))
	if err != nil {
		return fmt.Errorf("locate RPM metadata: %w", err)
	}
	if len(rpmRoots) != 1 {
		return fmt.Errorf("verify candidate: expected one RPM repository, got %d", len(rpmRoots))
	}
	repomdPath := rpmRoots[0]
	verifications := [][]string{
		{batchFlag, verifyFlag, filepath.Join(releaseDir, "InRelease")},
		{batchFlag, verifyFlag, filepath.Join(releaseDir, "Release.gpg"), filepath.Join(releaseDir, "Release")},
		{batchFlag, verifyFlag, repomdPath + ".asc", repomdPath},
	}
	for _, arguments := range verifications {
		if _, err := instance.runner.run(ctx, "", environment, "gpg", arguments...); err != nil {
			return err
		}
	}

	return nil
}

func ensureNewRoot(root string) error {
	if _, err := os.Stat(root); err == nil {
		return fmt.Errorf("candidate root already exists: %s", root)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect candidate root: %w", err)
	}
	if err := os.MkdirAll(root, directoryMode); err != nil {
		return fmt.Errorf("create candidate root: %w", err)
	}

	return nil
}

func cleanupFailedBuild(root string, buildErr error) error {
	if err := os.RemoveAll(root); err != nil {
		return errors.Join(buildErr, fmt.Errorf("clean failed candidate: %w", err))
	}

	return buildErr
}

func copyAssetPath(sourcePath string, destinationDir string) error {
	name := filepath.Base(sourcePath)
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open release asset %s: %w", name, err)
	}
	defer source.Close()

	if mkdirErr := os.MkdirAll(destinationDir, directoryMode); mkdirErr != nil {
		return fmt.Errorf("create asset directory: %w", mkdirErr)
	}
	destinationPath := filepath.Join(destinationDir, name)
	destination, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("create candidate asset %s: %w", name, err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return fmt.Errorf("copy release asset %s: %w", name, err)
	}

	return nil
}

func writeGzip(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read APT index for compression: %w", err)
	}
	destination, err := os.Create(path + ".gz")
	if err != nil {
		return fmt.Errorf("create compressed APT index: %w", err)
	}
	defer destination.Close()

	writer, err := gzip.NewWriterLevel(destination, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("create gzip writer: %w", err)
	}
	if _, err := writer.Write(content); err != nil {
		return fmt.Errorf("compress APT index: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close compressed APT index: %w", err)
	}

	return nil
}

func writeByHash(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read APT by-hash source: %w", err)
	}
	hashes := map[string]string{
		"SHA256": digestSHA256(content),
		"SHA512": digestSHA512(content),
	}
	for algorithm, digest := range hashes {
		destinationDir := filepath.Join(filepath.Dir(path), "by-hash", algorithm)
		if err := os.MkdirAll(destinationDir, directoryMode); err != nil {
			return fmt.Errorf("create APT by-hash directory: %w", err)
		}
		// #nosec G703 -- digest is hex-encoded from local content, not a caller-controlled path.
		if err := os.WriteFile(filepath.Join(destinationDir, digest), content, publicFileMode); err != nil {
			return fmt.Errorf("write APT by-hash object: %w", err)
		}
	}

	return nil
}

func digestSHA256(content []byte) string {
	digest := sha256.Sum256(content)

	return hex.EncodeToString(digest[:])
}

func digestSHA512(content []byte) string {
	digest := sha512.Sum512(content)

	return hex.EncodeToString(digest[:])
}
