package localrepo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanSyncOrdersActivationBeforeDeletion(t *testing.T) {
	t.Parallel()

	candidate := t.TempDir()
	remote := t.TempDir()
	writeTreeFile(t, candidate, "apt/pool/fixture/new.deb", "new package")
	writeTreeFile(t, candidate, "apt/dists/stable/fixture/binary-amd64/Packages.gz", "new index")
	writeTreeFile(t, candidate, "apt/dists/stable/InRelease", "new activation")
	writeTreeFile(t, candidate, "_state/manifest.json", "new state")
	writeTreeFile(t, remote, "apt/pool/fixture/old.deb", "old package")
	writeTreeFile(t, remote, "apt/dists/stable/fixture/binary-amd64/Packages.gz", "old index")
	writeTreeFile(t, remote, "apt/dists/stable/InRelease", "old activation")
	writeTreeFile(t, remote, "_state/manifest.json", "old state")

	plan, err := PlanSync(candidate, remote)

	require.NoError(t, err)
	assert.Equal(t, []SyncAction{
		{Stage: "content", Kind: "create", Path: "apt/pool/fixture/new.deb"},
		{Stage: "index", Kind: "replace", Path: "apt/dists/stable/fixture/binary-amd64/Packages.gz"},
		{Stage: "activate", Kind: "replace", Path: "apt/dists/stable/InRelease"},
		{Stage: "state", Kind: "replace", Path: "_state/manifest.json"},
		{Stage: "delete", Kind: "delete", Path: "apt/pool/fixture/old.deb"},
	}, plan.Actions)
}

func TestPlanSyncFailurePointsNeverDeleteCandidateContent(t *testing.T) {
	t.Parallel()

	candidate := t.TempDir()
	remote := t.TempDir()
	retainedPath := "apt/pool/fixture/retained.deb"
	expiredPath := "apt/pool/fixture/expired.deb"
	writeTreeFile(t, candidate, retainedPath, "retained")
	writeTreeFile(t, candidate, "apt/dists/stable/InRelease", "new")
	writeTreeFile(t, remote, retainedPath, "retained")
	writeTreeFile(t, remote, expiredPath, "expired")
	writeTreeFile(t, remote, "apt/dists/stable/InRelease", "old")

	plan, err := PlanSync(candidate, remote)
	require.NoError(t, err)
	firstDelete := len(plan.Actions)
	for index, action := range plan.Actions {
		if action.Kind == kindDelete {
			firstDelete = index
			break
		}
	}

	for stopBefore := 0; stopBefore <= len(plan.Actions); stopBefore++ {
		live := t.TempDir()
		copySnapshot(t, remote, live)
		applyPlanPrefix(t, candidate, live, plan, stopBefore)

		_, statErr := os.Stat(filepath.Join(live, filepath.FromSlash(retainedPath)))
		require.NoError(t, statErr, "failure before action %d removed retained content", stopBefore)
		_, statErr = os.Stat(filepath.Join(live, filepath.FromSlash(expiredPath)))
		if stopBefore <= firstDelete {
			require.NoError(t, statErr, "failure before action %d deleted content before the delete stage", stopBefore)
		} else {
			require.ErrorIs(t, statErr, os.ErrNotExist)
		}
	}

	final := t.TempDir()
	copySnapshot(t, remote, final)
	applyPlanPrefix(t, candidate, final, plan, len(plan.Actions))
	want, err := snapshotTree(candidate, false)
	require.NoError(t, err)
	got, err := snapshotTree(final, false)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func writeTreeFile(t *testing.T, root string, relative string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relative))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func copySnapshot(t *testing.T, source string, destination string) {
	t.Helper()

	files, err := snapshotTree(source, false)
	require.NoError(t, err)
	for relative := range files {
		copyTreeFile(t, source, destination, relative)
	}
}

func applyPlanPrefix(t *testing.T, candidate string, remote string, plan SyncPlan, stopBefore int) {
	t.Helper()

	for index, action := range plan.Actions {
		if index == stopBefore {
			return
		}
		if action.Kind == kindDelete {
			require.NoError(t, os.Remove(filepath.Join(remote, filepath.FromSlash(action.Path))))
			continue
		}
		copyTreeFile(t, candidate, remote, action.Path)
	}
}

func copyTreeFile(t *testing.T, source string, destination string, relative string) {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(relative)))
	require.NoError(t, err)
	path := filepath.Join(destination, filepath.FromSlash(relative))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, content, 0o644))
}
