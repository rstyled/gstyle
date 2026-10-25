package changerisk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/changerisk"
)

func TestRiskReportsCommittedGraphsUsingBasePolicy(t *testing.T) {
	root := repository(t)
	write(t, root, "gstyle.toml", "[risk]\npdg_change='medium'\ndependency_change='high'\n[[risk.calls]]\ncallee='os.WriteFile'\nlevel='high'\n")
	write(t, root, "store.go", "package sample\nfunc Save(path string, data []byte) error { return nil }\n")
	base := commit(t, root)
	write(t, root, "store.go", "package sample\nimport \"os\"\nfunc Save(path string, data []byte) error { return os.WriteFile(path, data, 0600) }\n")
	write(t, root, "gstyle.toml", "")
	head := commit(t, root)
	write(t, root, "store.go", "this is deliberately invalid uncommitted Go")
	t.Chdir(root)
	var output bytes.Buffer
	if err := Run(context.Background(), []string{"--base", base, "--head", head, "--format", "json"}, &output); err != nil {
		t.Fatal(err)
	}
	var report changerisk.Report
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Risk != changerisk.High || report.Base != base || report.Head != head || report.ChangedFunctions != 1 {
		t.Fatalf("unexpected report: %s", output.String())
	}
	matched := false
	for _, reason := range report.Reasons {
		if reason.Rule == "added_call" && reason.Level == changerisk.High && reason.Path == "store.go" && reason.Line == 3 {
			matched = true
		}
	}
	if !matched {
		t.Fatalf("new callee was not measured: %s", output.String())
	}
	if !report.DependenciesChanged || len(report.Unmeasured) != 3 {
		t.Fatalf("missing measured or unavailable evidence: %s", output.String())
	}
	output.Reset()
	if err := Run(context.Background(), []string{"--base", base, "--head", head}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "```mermaid") || !strings.Contains(output.String(), "os.WriteFile") || !strings.Contains(output.String(), "<summary>Graph changes</summary>") || strings.Contains(output.String(), "**Checks:**") {
		t.Fatalf("missing Markdown evidence: %s", output.String())
	}
	for _, removed := range []string{"Analysis details", "Not assessed:", "Compared "} {
		if strings.Contains(output.String(), removed) {
			t.Errorf("removed analysis text %q remains in CLI output: %s", removed, output.String())
		}
	}
	first := output.String()
	output.Reset()
	if err := Run(context.Background(), []string{"--base", base, "--head", head}, &output); err != nil || output.String() != first {
		t.Fatalf("report is not deterministic: %v\n%s", err, output.String())
	}
	if worktrees := git(t, root, "worktree", "list", "--porcelain"); strings.Contains(worktrees, "gstyle-risk-") {
		t.Fatalf("temporary worktree leaked: %s", worktrees)
	}
}

func TestRiskIgnoresSourcePositionsAndReportsMissingMeasurements(t *testing.T) {
	root := repository(t)
	write(t, root, "gstyle.toml", "")
	write(t, root, "source.go", "package sample\nfunc Count(value int) int { return value+1 }\n")
	base := commit(t, root)
	write(t, root, "source.go", "package sample\n\n\n// existing function moves\nfunc Count(value int) int { return value+1 }\n")
	head := commit(t, root)
	t.Chdir(root)
	var output bytes.Buffer
	if err := Run(context.Background(), []string{"--base", base, "--head", head, "--format", "json"}, &output); err != nil {
		t.Fatal(err)
	}
	var report changerisk.Report
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.ChangedFunctions != 0 || report.Risk != changerisk.Unrated || report.DependenciesChanged {
		t.Fatalf("positions or missing measurements became a verdict: %s", output.String())
	}
}

func TestRiskRejectsInvalidInputs(t *testing.T) {
	root := repository(t)
	write(t, root, "gstyle.toml", "")
	commit(t, root)
	t.Chdir(root)
	for _, args := range [][]string{{"--base", "missing"}, {"--head", "missing"}, {"--format", "text"}, {"--config", "../outside"}, {"--config", "/absolute"}, {"--config", "."}, {"extra.go"}} {
		if err := Run(context.Background(), args, &bytes.Buffer{}); err == nil {
			t.Errorf("invalid arguments accepted: %v", args)
		}
	}
}

func TestSnapshotCleanupAfterCancellation(t *testing.T) {
	root := repository(t)
	write(t, root, "gstyle.toml", "")
	head := commit(t, root)
	ctx, cancel := context.WithCancel(context.Background())
	err := withSnapshot(ctx, root, head, func(string) error { cancel(); return context.Canceled })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	if worktrees := git(t, root, "worktree", "list", "--porcelain"); strings.Contains(worktrees, "gstyle-risk-") {
		t.Fatalf("temporary worktree leaked: %s", worktrees)
	}
}

func TestDefaultBaseUsesRemoteAndSelectedHead(t *testing.T) {
	root := repository(t)
	write(t, root, "gstyle.toml", "")
	first := commit(t, root)
	write(t, root, "note.txt", "second\n")
	second := commit(t, root)
	git(t, root, "checkout", "-b", "feature")
	git(t, root, "branch", "-f", "master", first)
	git(t, root, "update-ref", "refs/remotes/origin/master", second)
	write(t, root, "note.txt", "third\n")
	third := commit(t, root)
	for head, want := range map[string]string{third: second, first: first} {
		got, err := baseRevision(context.Background(), root, options{}, head)
		if err != nil || got != want {
			t.Fatalf("head=%s base=%s want=%s error=%v", head, got, want, err)
		}
	}
}

func TestRiskDoesNotRenderASelectedPackageTypeError(t *testing.T) {
	root := repository(t)
	write(t, root, "gstyle.toml", "")
	write(t, root, "source.go", "package sample\nfunc Valid() {}\n")
	base := commit(t, root)
	write(t, root, "source.go", "package sample\nfunc Invalid() { missing() }\n")
	head := commit(t, root)
	t.Chdir(root)
	var output bytes.Buffer
	if err := Run(context.Background(), []string{"--base", base, "--head", head}, &output); err == nil || output.Len() != 0 {
		t.Fatalf("type error produced a report: %v %s", err, output.String())
	}
	if worktrees := git(t, root, "worktree", "list", "--porcelain"); strings.Contains(worktrees, "gstyle-risk-") {
		t.Fatalf("temporary worktree leaked: %s", worktrees)
	}
}

func TestSnapshotChangesAndEscapingPolicyAreRejected(t *testing.T) {
	root := repository(t)
	write(t, root, "gstyle.toml", "")
	head := commit(t, root)
	err := withSnapshot(context.Background(), root, head, func(dir string) error {
		write(t, dir, "go.mod", "module changed.example\n\ngo 1.26.0\n")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "snapshot changed") {
		t.Fatalf("changed snapshot accepted: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "gstyle.toml")); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "policy.toml")
	if err := os.WriteFile(outside, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "gstyle.toml")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	head = commit(t, root)
	t.Chdir(root)
	if err := Run(context.Background(), []string{"--base", head}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "symlinks") {
		t.Fatalf("escaping policy accepted: %v", err)
	}
}

func repository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	git(t, root, "init")
	git(t, root, "config", "user.name", "Test")
	git(t, root, "config", "user.email", "test@example.com")
	write(t, root, "go.mod", "module example.com/sample\n\ngo 1.26.0\n")
	return root
}

func write(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func commit(t *testing.T, root string) string {
	t.Helper()
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "fixture")
	return git(t, root, "rev-parse", "HEAD")
}
