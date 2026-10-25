package changerisk

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func command(ctx context.Context, root, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = root
	var stderr strings.Builder
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w: %s", name, args[0], err, strings.TrimSpace(stderr.String()))
	}
	return data, nil
}

func revision(ctx context.Context, root, ref string) (string, error) {
	data, err := command(ctx, root, "git", "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func fields(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	return strings.Split(strings.TrimSuffix(string(data), "\x00"), "\x00")
}

type changeSet struct {
	paths   []string
	warning string
}

func changedPaths(ctx context.Context, root, base, head string) (changeSet, error) {
	data, err := command(ctx, root, "git", "diff", "--no-ext-diff", "--no-textconv", "--no-renames", "--ignore-submodules=none", "--raw", "-z", base, head, "--")
	if err != nil {
		return changeSet{}, err
	}
	return parseChangedPaths(fields(data))
}

func parseChangedPaths(records []string) (changeSet, error) {
	if len(records)%2 != 0 {
		return changeSet{}, fmt.Errorf("incomplete Git file metadata")
	}
	result := changeSet{}
	var unsupported []string
	for index := 0; index < len(records); index += 2 {
		name := records[index+1]
		regular, err := regularChange(records[index], name)
		if err != nil {
			return changeSet{}, err
		}
		result.paths = append(result.paths, name)
		if !regular {
			unsupported = append(unsupported, name)
		}
	}
	if len(unsupported) != 0 {
		result.warning = "changed executable, symlink or submodule paths need review: " + strings.Join(unsupported, ", ")
	}
	return result, nil
}

func regularChange(header, name string) (bool, error) {
	metadata := strings.Fields(header)
	if len(metadata) != 5 || !strings.HasPrefix(metadata[0], ":") || !fs.ValidPath(name) {
		return false, fmt.Errorf("invalid Git file metadata")
	}
	modes := []string{"000000", "100644"}
	return slices.Contains(modes, strings.TrimPrefix(metadata[0], ":")) && slices.Contains(modes, metadata[1]), nil
}

func withSnapshot(ctx context.Context, root, commit string, visit func(string) error) (err error) {
	temporary, err := os.MkdirTemp("", "gstyle-risk-")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(temporary)) }()
	checkout := filepath.Join(temporary, "source")
	if _, err := command(ctx, root, "git", "worktree", "add", "--quiet", "--detach", checkout, commit); err != nil {
		return err
	}
	defer func() {
		cleanupContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		_, cleanup := command(cleanupContext, root, "git", "worktree", "remove", "--force", checkout)
		err = errors.Join(err, cleanup)
	}()
	if err := visit(checkout); err != nil {
		return err
	}
	status, statusErr := command(ctx, checkout, "git", "status", "--porcelain=v1", "--untracked-files=all", "--ignored=matching")
	if statusErr != nil {
		return statusErr
	}
	if len(status) != 0 {
		return fmt.Errorf("snapshot changed during analysis; no report produced")
	}
	return nil
}
