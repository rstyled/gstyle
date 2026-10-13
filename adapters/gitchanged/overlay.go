package gitchanged

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// NewOverlayDiff uses the same base as a disk diff, with simulated edits on
// its new side. This keeps changed-function selection equal to an apply run.
func NewOverlayDiff(cwd string, paths []string, base string, overlay map[string][]byte) (*Diff, error) {
	diff, err := NewDiff(cwd, paths, base)
	if err != nil {
		return nil, err
	}
	resolved, err := ResolveBase(diff.Root, base)
	if err != nil {
		return nil, err
	}
	for path, content := range overlay {
		relative := RootRelative(path, diff.Root, cwd)
		ranges, err := overlayRanges(diff.Root, resolved, relative, content)
		if err != nil {
			return nil, err
		}
		diff.Ranges[relative] = ranges
	}
	return diff, nil
}

func overlayRanges(root, base, path string, content []byte) ([]LineRange, error) {
	if base == "" {
		return []LineRange{{Start: 1, End: strings.Count(string(content), "\n") + 1}}, nil
	}
	tracked, err := gitOutput(root, "ls-tree", "--name-only", base, "--", path)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(tracked) == "" {
		return []LineRange{{Start: 1, End: strings.Count(string(content), "\n") + 1}}, nil
	}
	original, err := gitOutput(root, "show", base+":"+path)
	if err != nil {
		return nil, fmt.Errorf("read diff base: %w", err)
	}
	return compareOverlay([]byte(original), content)
}

func compareOverlay(old, next []byte) ([]LineRange, error) {
	directory, err := os.MkdirTemp("", "gstyle-changed-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	before, after := filepath.Join(directory, "before"), filepath.Join(directory, "after")
	if err := writeComparison(before, after, old, next); err != nil {
		return nil, err
	}
	output, err := exec.Command("diff", "-U", "0", before, after).Output()
	if err := comparisonError(err); err != nil {
		return nil, err
	}
	var ranges []LineRange
	for line := range strings.SplitSeq(string(output), "\n") {
		if span, ok := parseHunkRange(line); ok {
			ranges = append(ranges, span)
		}
	}
	return ranges, nil
}

func writeComparison(before, after string, old, next []byte) error {
	if err := os.WriteFile(before, old, 0600); err != nil {
		return err
	}
	return os.WriteFile(after, next, 0600)
}

func comparisonError(err error) error {
	if err == nil {
		return nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return nil
	}
	return err
}
