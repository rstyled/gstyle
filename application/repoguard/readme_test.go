package repoguard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const readmePolicy = "[[guard.files]]\nid='nested_readme'\nmode='added'\nnames=['README.md']\nallow_paths=['README.md']\n"

func TestNestedReadmeRejectsNewPathsAndKeepsExistingEdits(t *testing.T) {
	root := testRepository(t)
	writeFile(t, root, "gstyle.toml", readmePolicy)
	writeFile(t, root, "README.md", "root\n")
	writeFile(t, root, "existing/README.md", "existing\n")
	writeFile(t, root, "old/README.md", "moved\n")
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "base")
	writeFile(t, root, "README.md", "edited root\n")
	writeFile(t, root, "existing/README.md", "edited existing\n")
	writeFile(t, root, "new/README.md", "new\n")
	writeFile(t, root, "deep/nested/README.md", "staged\n")
	gitTest(t, root, "add", "deep")
	gitTest(t, root, "mv", "old", "renamed")
	t.Chdir(root)
	want := []string{"deep/nested/README.md", "new/README.md", "renamed/README.md"}
	assertReadmeFindings(t, "HEAD", want)
	gitTest(t, root, "add", ".")
	gitTest(t, root, "commit", "-m", "new paths")
	assertReadmeFindings(t, "HEAD^", want)
	for _, name := range want {
		if err := os.Remove(filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := Run(context.Background(), []string{"--base", "HEAD^"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("existing edits or deleted paths rejected: %v", err)
	}
}

func TestNestedReadmeBeforeFirstCommitAndWithDisabledPolicy(t *testing.T) {
	root := testRepository(t)
	writeFile(t, root, "gstyle.toml", readmePolicy)
	writeFile(t, root, "README.md", "root\n")
	writeFile(t, root, "docs/README.md", "nested\n")
	gitTest(t, root, "add", ".")
	t.Chdir(root)
	assertReadmeFindings(t, "", []string{"docs/README.md"})
	if err := Run(context.Background(), []string{"--base", "missing"}, &bytes.Buffer{}); err == nil || errors.Is(err, ErrFindings) {
		t.Fatalf("invalid base accepted: %v", err)
	}
	writeFile(t, root, "gstyle.toml", "")
	if err := Run(context.Background(), nil, &bytes.Buffer{}); err != nil {
		t.Fatalf("disabled policy restricts README files: %v", err)
	}
}

func assertReadmeFindings(t *testing.T, base string, want []string) {
	t.Helper()
	args := []string{"--format", "json"}
	if base != "" {
		args = append(args, "--base", base)
	}
	var output bytes.Buffer
	err := Run(context.Background(), args, &output)
	if !errors.Is(err, ErrFindings) {
		t.Fatalf("README findings: %v %s", err, output.String())
	}
	var got report
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, finding := range got.Findings {
		if finding.RuleID != "nested_readme" {
			t.Fatalf("unexpected rule: %+v", finding)
		}
		paths = append(paths, finding.Path)
	}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths=%v, want %v", paths, want)
	}
}
