package changerisk

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestGitFileMetadataCannotDisappearFromTheGate(t *testing.T) {
	for _, mode := range []string{"100755", "120000", "160000", "unexpected"} {
		changes, err := parseChangedPaths([]string{":100644 " + mode + " abc def M", "docs/guide.md"})
		if err != nil || len(changes.paths) != 1 || !strings.Contains(changes.warning, "docs/guide.md") {
			t.Fatalf("unsafe Git mode %s was ignored: %+v %v", mode, changes, err)
		}
	}
	for _, records := range [][]string{{"missing path"}, {"invalid", "README.md"}, {":100644 100644 abc def M", "../README.md"}} {
		if _, err := parseChangedPaths(records); err == nil {
			t.Fatalf("malformed Git metadata was accepted: %v", records)
		}
	}
	for _, modes := range []string{":100644 100644 abc def M", ":000000 100644 abc def A", ":100644 000000 abc def D"} {
		changes, err := parseChangedPaths([]string{modes, "docs/guide\nname.md"})
		if err != nil || changes.warning != "" || changes.paths[0] != "docs/guide\nname.md" {
			t.Fatalf("regular Git paths were changed: %+v %v", changes, err)
		}
	}
}

func TestRiskCheckRejectsAnUninitializedGitlink(t *testing.T) {
	root := repository(t)
	write(t, root, "gstyle.toml", "[risk]\npdg_change='low'\n")
	write(t, root, "source.go", "package sample\nfunc Count() int { return 1 }\n")
	base := commit(t, root)
	git(t, root, "update-index", "--add", "--cacheinfo", "160000,"+base+",docs/module.md")
	git(t, root, "commit", "-m", "add uninitialized gitlink")
	head := git(t, root, "rev-parse", "HEAD")
	t.Chdir(root)
	var output bytes.Buffer
	err := Run(context.Background(), []string{"--base", base, "--head", head, "--check", "--format", "json"}, &output)
	if !errors.Is(err, ErrReviewRequired) || !strings.Contains(output.String(), "submodule paths need review") {
		t.Fatalf("uninitialized submodule passed the gate: %v\n%s", err, output.String())
	}
}
