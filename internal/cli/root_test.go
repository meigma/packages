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

func TestRebuildLocalCommandPassesResolvedRequestAndPrintsResult(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	expectedRequest := localrepo.RebuildRequest{
		RegistryPath: "/fixtures/projects.yml",
		Project:      "phase2-fixture",
		ReleasesDir:  "/fixtures/releases",
		Root:         "/tmp/candidate",
		GNUPGHome:    "/tmp/gnupg",
		SigningKey:   "0123456789ABCDEF",
		BaseURL:      "http://phase2-repo:8080",
	}
	expectedResult := localrepo.RebuildResult{
		Project:            "phase2-fixture",
		Root:               "/tmp/candidate",
		SelectedVersions:   []string{"v2.0.0", "v1.1.0"},
		DesiredStateDigest: "abc123",
		NoOp:               false,
	}
	root := NewRootCommand(Options{
		Out: &stdout,
		Err: &stderr,
		RebuildLocal: func(_ context.Context, request localrepo.RebuildRequest) (localrepo.RebuildResult, error) {
			assert.Equal(t, expectedRequest, request)

			return expectedResult, nil
		},
	})
	root.SetArgs([]string{
		"rebuild-local",
		"--registry", expectedRequest.RegistryPath,
		"--project", expectedRequest.Project,
		"--releases", expectedRequest.ReleasesDir,
		"--root", expectedRequest.Root,
		"--gnupg-home", expectedRequest.GNUPGHome,
		"--signing-key", expectedRequest.SigningKey,
		"--base-url", expectedRequest.BaseURL,
	})

	require.NoError(t, root.ExecuteContext(context.Background()))
	assert.JSONEq(t, `{
		"project":"phase2-fixture",
		"root":"/tmp/candidate",
		"selected_versions":["v2.0.0","v1.1.0"],
		"desired_state_digest":"abc123",
		"no_op":false
	}`, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestPlanSyncCommandPrintsTheResolvedPlan(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	expected := localrepo.SyncPlan{
		CandidateRoot: "/tmp/candidate",
		RemoteRoot:    "/tmp/remote",
		Actions: []localrepo.SyncAction{{
			Stage: "delete",
			Kind:  "delete",
			Path:  "expired.rpm",
		}},
	}
	root := NewRootCommand(Options{
		Out: &stdout,
		Err: &stderr,
		PlanSync: func(candidate string, remote string) (localrepo.SyncPlan, error) {
			assert.Equal(t, expected.CandidateRoot, candidate)
			assert.Equal(t, expected.RemoteRoot, remote)

			return expected, nil
		},
	})
	root.SetArgs([]string{"plan-sync", "--root", expected.CandidateRoot, "--remote", expected.RemoteRoot})

	require.NoError(t, root.ExecuteContext(context.Background()))
	assert.JSONEq(t, `{
		"candidate_root":"/tmp/candidate",
		"remote_root":"/tmp/remote",
		"actions":[{"stage":"delete","kind":"delete","path":"expired.rpm"}]
	}`, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestValidateRequestCommandPassesResolvedInputAndPrintsResult(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRootCommand(Options{
		Out: &stdout,
		Err: &stderr,
		ValidateRequest: func(registry string, project string, tag string) (localrepo.RequestValidation, error) {
			assert.Equal(t, "/fixtures/projects.yml", registry)
			assert.Equal(t, "phase3-fixture", project)
			assert.Equal(t, "v2.1.0", tag)

			return localrepo.RequestValidation{Project: project, Tag: tag}, nil
		},
	})
	root.SetArgs([]string{
		"validate-request",
		"--registry", "/fixtures/projects.yml",
		"--project", "phase3-fixture",
		"--tag", "v2.1.0",
	})

	require.NoError(t, root.ExecuteContext(context.Background()))
	assert.JSONEq(t, `{"project":"phase3-fixture","tag":"v2.1.0"}`, stdout.String())
	assert.Empty(t, stderr.String())
}
