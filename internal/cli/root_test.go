package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meigma/packages/internal/localrepo"
)

func TestVersionFlagPrintsBuildMetadata(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRootCommand(Options{
		Out: &stdout,
		Err: &stderr,
		Build: BuildInfo{
			Version: "0.1.0",
			Commit:  "abc1234",
			Date:    "2026-05-08T10:00:00Z",
		},
	})
	root.SetArgs([]string{"--version"})

	require.NoError(t, root.ExecuteContext(context.Background()))
	assert.Equal(t, "meigma-packages 0.1.0 (abc1234) built 2026-05-08T10:00:00Z\n", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestRootCommandPrintsHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRootCommand(Options{
		Out: &stdout,
		Err: &stderr,
	})

	require.NoError(t, root.ExecuteContext(context.Background()))
	assert.Contains(t, stdout.String(), "Build and publish Meigma APT and RPM repositories")
	assert.Empty(t, stderr.String())
}

func TestBuildLocalCommandPassesResolvedRequestAndPrintsResult(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	expectedRequest := localrepo.Request{
		RegistryPath: "/fixtures/projects.yml",
		Project:      "phase1-fixture",
		ReleaseDir:   "/fixtures/release",
		Root:         "/tmp/candidate",
		GNUPGHome:    "/tmp/gnupg",
		SigningKey:   "0123456789ABCDEF",
		BaseURL:      "http://phase1-repo:8080",
	}
	expectedResult := localrepo.Result{
		Project:         "phase1-fixture",
		PackageName:     "meigma-phase0",
		Root:            "/tmp/candidate",
		DEBArchitecture: "arm64",
		SigningKey:      "0123456789ABCDEF",
	}
	root := NewRootCommand(Options{
		Out: &stdout,
		Err: &stderr,
		BuildLocal: func(_ context.Context, request localrepo.Request) (localrepo.Result, error) {
			assert.Equal(t, expectedRequest, request)

			return expectedResult, nil
		},
	})
	root.SetArgs([]string{
		"build-local",
		"--registry", expectedRequest.RegistryPath,
		"--project", expectedRequest.Project,
		"--release", expectedRequest.ReleaseDir,
		"--root", expectedRequest.Root,
		"--gnupg-home", expectedRequest.GNUPGHome,
		"--signing-key", expectedRequest.SigningKey,
		"--base-url", expectedRequest.BaseURL,
	})

	require.NoError(t, root.ExecuteContext(context.Background()))
	assert.JSONEq(t, `{
		"project":"phase1-fixture",
		"package_name":"meigma-phase0",
		"root":"/tmp/candidate",
		"deb_architecture":"arm64",
		"signing_key":"0123456789ABCDEF"
	}`, stdout.String())
	assert.Empty(t, stderr.String())
}
