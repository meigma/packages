package localrepo

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	kindCreate  = "create"
	kindReplace = "replace"
	kindDelete  = "delete"

	stageContent  = "content"
	stageIndex    = "index"
	stageActivate = "activate"
	stageState    = "state"
	stageDelete   = "delete"
)

const (
	contentStageRank = iota + 1
	indexStageRank
	activateStageRank
	stateStageRank
	deleteStageRank
	unknownStageRank
)

// SyncPlan describes an ordered candidate-to-remote filesystem change set.
type SyncPlan struct {
	// CandidateRoot is the desired candidate tree.
	CandidateRoot string `json:"candidate_root"`
	// RemoteRoot is the existing tree compared with the candidate.
	RemoteRoot string `json:"remote_root"`
	// Actions lists creates, replacements, and deletions in application order.
	Actions []SyncAction `json:"actions"`
}

// SyncAction describes one ordered filesystem mutation.
type SyncAction struct {
	// Stage is the publication stage that owns the action.
	Stage string `json:"stage"`
	// Kind is create, replace, or delete.
	Kind string `json:"kind"`
	// Path is relative to the candidate and remote roots.
	Path string `json:"path"`
}

// PlanSync compares a verified candidate tree with an existing filesystem tree.
//
// The returned order activates metadata only after referenced content exists and
// defers every deletion until all creates and replacements have completed.
func PlanSync(candidateRoot string, remoteRoot string) (SyncPlan, error) {
	if strings.TrimSpace(candidateRoot) == "" {
		return SyncPlan{}, errors.New("candidate root is required")
	}
	if strings.TrimSpace(remoteRoot) == "" {
		return SyncPlan{}, errors.New("remote root is required")
	}
	candidate, err := snapshotTree(candidateRoot, false)
	if err != nil {
		return SyncPlan{}, fmt.Errorf("snapshot candidate: %w", err)
	}
	remote, err := snapshotTree(remoteRoot, true)
	if err != nil {
		return SyncPlan{}, fmt.Errorf("snapshot remote: %w", err)
	}

	actions := make([]SyncAction, 0, len(candidate)+len(remote))
	for path, digest := range candidate {
		remoteDigest, exists := remote[path]
		if !exists {
			actions = append(actions, SyncAction{Stage: publicationStage(path), Kind: kindCreate, Path: path})
			continue
		}
		if remoteDigest != digest {
			actions = append(actions, SyncAction{Stage: publicationStage(path), Kind: kindReplace, Path: path})
		}
	}
	for path := range remote {
		if _, exists := candidate[path]; !exists {
			actions = append(actions, SyncAction{Stage: stageDelete, Kind: kindDelete, Path: path})
		}
	}
	sort.Slice(actions, func(left, right int) bool {
		leftRank := stageRank(actions[left].Stage)
		rightRank := stageRank(actions[right].Stage)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if actions[left].Path != actions[right].Path {
			return actions[left].Path < actions[right].Path
		}

		return actions[left].Kind < actions[right].Kind
	})

	return SyncPlan{
		CandidateRoot: candidateRoot,
		RemoteRoot:    remoteRoot,
		Actions:       actions,
	}, nil
}

func snapshotTree(root string, allowMissing bool) (map[string]string, error) {
	if _, err := os.Stat(root); err != nil {
		if allowMissing && os.IsNotExist(err) {
			return map[string]string{}, nil
		}

		return nil, err
	}
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root || entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("tree contains symlink: %s", path)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("tree contains non-regular file: %s", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("resolve relative path: %w", err)
		}
		digest, err := digestFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = digest

		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func publicationStage(path string) string {
	base := filepath.Base(path)
	if strings.HasSuffix(base, ".deb") || strings.HasSuffix(base, ".rpm") ||
		strings.Contains(path, "/by-hash/") ||
		(strings.Contains(path, "/repodata/") && base != "repomd.xml" && base != "repomd.xml.asc") {
		return stageContent
	}
	if base == "Packages" || base == "Packages.gz" {
		return stageIndex
	}
	if base == "InRelease" || base == "Release" || base == "Release.gpg" ||
		base == "repomd.xml" || base == "repomd.xml.asc" {
		return stageActivate
	}

	return stageState
}

func stageRank(stage string) int {
	switch stage {
	case stageContent:
		return contentStageRank
	case stageIndex:
		return indexStageRank
	case stageActivate:
		return activateStageRank
	case stageState:
		return stateStageRank
	case stageDelete:
		return deleteStageRank
	default:
		return unknownStageRank
	}
}
