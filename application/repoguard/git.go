package repoguard

import (
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

func git(ctx context.Context, root string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", args[0], err)
	}
	return output, nil
}

func fileList(ctx context.Context, root string, paths []string) ([]string, error) {
	args := append([]string{"ls-files", "--cached", "--others", "--exclude-standard", "-z", "--"}, paths...)
	output, err := git(ctx, root, args...)
	if err != nil {
		return nil, err
	}
	files := nulFields(output)
	slices.Sort(files)
	return slices.Compact(files), nil
}

func nulFields(output []byte) []string {
	if len(output) == 0 {
		return nil
	}
	return strings.Split(strings.TrimSuffix(string(output), "\x00"), "\x00")
}

func changedFiles(ctx context.Context, root, base string, paths []string) (map[string]string, error) {
	result := map[string]string{}
	if base != "" {
		args := append([]string{"diff", "--no-ext-diff", "--no-textconv", "--name-status", "--find-renames", "-z", base, "--"}, paths...)
		output, err := git(ctx, root, args...)
		if err != nil {
			return nil, err
		}
		parsed, parseErr := parseChanges(nulFields(output))
		if parseErr != nil {
			return nil, parseErr
		}
		result = parsed
	}
	args := append([]string{"ls-files", "--others", "--exclude-standard", "-z", "--"}, paths...)
	output, err := git(ctx, root, args...)
	if err != nil {
		return nil, err
	}
	for _, file := range nulFields(output) {
		result[file] = ""
	}
	return result, nil
}

func parseChanges(fields []string) (map[string]string, error) {
	result := map[string]string{}
	for len(fields) > 0 {
		item, remaining, err := takeChange(fields)
		if err != nil {
			return nil, err
		}
		fields = remaining
		if !item.deleted {
			result[item.name] = item.previous
		}
	}
	return result, nil
}

type change struct {
	name, previous string
	deleted        bool
}

func takeChange(fields []string) (change, []string, error) {
	if len(fields) < 2 {
		return change{}, nil, fmt.Errorf("invalid git name-status output")
	}
	status, name := fields[0], fields[1]
	if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
		if len(fields) < 3 {
			return change{}, nil, fmt.Errorf("missing git rename destination")
		}
		return change{name: fields[2], previous: name}, fields[3:], nil
	}
	previous := name
	if status == "A" {
		previous = ""
	}
	return change{name: name, previous: previous, deleted: status == "D"}, fields[2:], nil
}
