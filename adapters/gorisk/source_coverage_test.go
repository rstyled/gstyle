package gorisk

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/changerisk"
)

func TestDeclarationFingerprintSeparatesBodyAndDeclarationChanges(t *testing.T) {
	for _, test := range []struct {
		name, before, after string
		unchanged           bool
	}{
		{"body", "package p; func value() int { return 1 }", "package p; func value() int { return 2 }", true},
		{"positions", "package p\nfunc value() int { return 1 }\n", "\npackage p\n\nfunc value( ) int {return 1}\n", true},
		{"commented body", "package p\n// Value adjusts input.\nfunc value(n int) int {\n// Keep arithmetic here.\nreturn n+1\n}", "package p\n// Value adjusts input.\nfunc value(n int) int {\n// Keep arithmetic here.\nreturn n+2\n}", true},
		{"literal", "package p; func value() string { return `//go:embed old` }", "package p; func value() string { return `//go:embed new` }", true},
		{"private variable", "package p; var enabled = true", "package p; var enabled = false", false},
		{"constant", "package p; const enabled = true", "package p; const enabled = false", false},
		{"initializer body", "package p; var enabled = func() bool { return true }()", "package p; var enabled = func() bool { return false }()", false},
		{"package", "package p", "package q", false},
		{"signature", "package p; func value() int { return 1 }", "package p; func value(n int) int { return n }", false},
		{"added declaration", "package p", "package p; func init() {}", false},
		{"Go directive", "package p\n//go:noinline\nfunc value() {}", "package p\n//go:nosplit\nfunc value() {}", false},
		{"moved Go directive", "package p\n//go:noinline\nfunc first() {}\nfunc second() {}", "package p\nfunc first() {}\n//go:noinline\nfunc second() {}", false},
		{"moved embed directive", "package p\nvar (\n//go:embed first\nfirst string\nsecond string\n)", "package p\nvar (\nfirst string\n//go:embed first\nsecond string\n)", false},
		{"line directive", "package p\n//line old.go:10\nfunc value() {}", "package p\n//line new.go:10\nfunc value() {}", false},
		{"cgo", "package p\n/* #define VALUE 1 */\nimport \"C\"", "package p\n/* #define VALUE 2 */\nimport \"C\"", false},
		{"body directive", "package p\nfunc value() {\n//line old.go:10\n}", "package p\nfunc value() {\n//line new.go:10\n}", false},
		{"raw carriage return", "package p\n/* a\rb */\nfunc value() {}", "package p\n/* ab */\nfunc value() {}", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			before, _, err := declarationFacts("source.go", test.before, nil)
			if err != nil {
				t.Fatal(err)
			}
			after, _, err := declarationFacts("source.go", test.after, nil)
			if err != nil {
				t.Fatal(err)
			}
			if (before == after) != test.unchanged {
				t.Fatalf("unchanged=%v, want %v", before == after, test.unchanged)
			}
		})
	}
	if _, _, err := declarationFacts("broken.go", "package", nil); err == nil {
		t.Fatal("invalid source received a fingerprint")
	}
}

func TestSourceCoverageProtectsChangesOutsideFunctionBodies(t *testing.T) {
	const source = "package sample\nimport \"strings\"\nvar enabled = true\nfunc ready() bool { return enabled }\nfunc clean(value string) string { return strings.TrimSpace(value) }\n// Count adjusts input.\nfunc Count(value int) int {\n// Keep arithmetic here.\nreturn value+1\n}\n"
	for _, test := range []struct {
		name, after string
		want        changerisk.Level
	}{
		{"private variable", strings.Replace(source, "enabled = true", "enabled = false", 1), changerisk.Unknown},
		{"existing helper initializer", source + "var startup = ready()\n", changerisk.Unknown},
		{"existing import initializer", source + "var startup = strings.TrimSpace(\" sample \")\n", changerisk.Unknown},
		{"documented body", strings.Replace(source, "return value+1", "return value+2", 1), changerisk.Low},
		{"removed declaration", strings.Replace(source, "func ready() bool { return enabled }\n", "", 1), changerisk.Unknown},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := sourceModule(t, map[string]string{"source.go": source})
			base := coverageSnapshot(t, root, []string{"source.go"})
			coverageWrite(t, root, "source.go", test.after)
			head := coverageSnapshot(t, root, []string{"source.go"})
			report := changerisk.Evaluate(base, head, changerisk.Config{PDGChange: changerisk.Low}, changerisk.APIReport{})
			if report.Risk != test.want {
				t.Fatalf("source coverage rating: %+v", report)
			}
			if test.want == changerisk.Low {
				if report.ChangedFunctions != 1 || base.Declarations["source.go"] != head.Declarations["source.go"] {
					t.Fatalf("body change did not use the PDG policy: %+v", report)
				}
				return
			}
			if !slices.ContainsFunc(report.Reasons, func(reason changerisk.Reason) bool {
				return reason.Rule == "analysis_unavailable" && reason.Path == "source.go"
			}) {
				t.Fatalf("missing path-specific coverage evidence: %+v", report.Reasons)
			}
		})
	}
}

func TestSourceCoverageRejectsBuildExcludedFiles(t *testing.T) {
	root := sourceModule(t, map[string]string{
		"source.go": "package sample\nfunc active() {}\n",
		"tagged.go": "//go:build gstyle_source_coverage_missing\n\npackage sample\nvar enabled = true\n",
	})
	snapshot := coverageSnapshot(t, root, []string{"source.go", "tagged.go"})
	if snapshot.Declarations["source.go"] == "" || snapshot.Declarations["tagged.go"] != "" || !strings.Contains(snapshot.AnalysisError, "tagged.go") {
		t.Fatalf("excluded source received coverage: %+v", snapshot)
	}
	report := changerisk.Evaluate(snapshot, snapshot, changerisk.Config{PDGChange: changerisk.Low}, changerisk.APIReport{})
	if report.Risk != changerisk.Unknown {
		t.Fatalf("build-excluded source received a low rating: %+v", report)
	}
}

func TestSourceCoverageKeepsRemovedAndRenamedFiles(t *testing.T) {
	for _, rename := range []bool{false, true} {
		root := sourceModule(t, map[string]string{"source.go": "package sample\nfunc value() int { return 1 }\n"})
		paths := []string{"source.go", "renamed.go"}
		base := coverageSnapshot(t, root, paths)
		if rename {
			if err := os.Rename(filepath.Join(root, "source.go"), filepath.Join(root, "renamed.go")); err != nil {
				t.Fatal(err)
			}
		} else if err := os.Remove(filepath.Join(root, "source.go")); err != nil {
			t.Fatal(err)
		}
		head := coverageSnapshot(t, root, paths)
		report := changerisk.Evaluate(base, head, changerisk.Config{PDGChange: changerisk.Low}, changerisk.APIReport{})
		if report.Risk != changerisk.Unknown {
			t.Fatalf("removed or renamed source disappeared: %+v", report)
		}
	}
}

func TestSourceCoverageLimitsInertDocumentation(t *testing.T) {
	for _, test := range []struct {
		path string
		low  bool
	}{
		{"README.md", true}, {"docs/guide.md", true}, {"docs/nested/guide.md", true},
		{"notes.md", false}, {".github/workflows/check.yml", false}, {"gstyle.toml", false},
		{"quality-baseline.json", false}, {"script.sh", false}, {"asset.svg", false}, {"input.s", false},
		{"AGENTS.md", false}, {"docs/AGENTS.md", false}, {"docs/SKILL.md", false}, {"docs/CLAUDE.md", false},
		{"docs/skills/helper.md", false}, {"docs/.hidden/guide.md", false}, {"docs/.guide.md", false},
		{"source_test.go", false}, {"nested/source.go", false},
	} {
		t.Run(test.path, func(t *testing.T) {
			root := t.TempDir()
			coverageWrite(t, root, test.path, "content\n")
			if test.path == "nested/source.go" {
				coverageWrite(t, root, "nested/go.mod", "module nested\n")
			}
			base := coverageSnapshot(t, root, []string{test.path})
			if err := os.Remove(filepath.Join(root, test.path)); err != nil {
				t.Fatal(err)
			}
			head := coverageSnapshot(t, root, []string{test.path})
			report := changerisk.Evaluate(base, head, changerisk.Config{PDGChange: changerisk.Low}, changerisk.APIReport{})
			want := changerisk.Unknown
			if test.low {
				want = changerisk.Low
			}
			if report.Risk != want || (!test.low && !strings.Contains(base.AnalysisError, test.path)) {
				t.Fatalf("file scope rating: %+v", report)
			}
			if report := changerisk.Evaluate(base, head, changerisk.Config{}, changerisk.APIReport{}); report.Risk != changerisk.Unrated {
				t.Fatalf("unconfigured file change received a rating: %+v", report)
			}
		})
	}
}

func TestSourceCoverageRejectsUnsafeDocumentationModes(t *testing.T) {
	for _, kind := range []string{"symlink", "executable", "directory"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "README.md")
			switch kind {
			case "symlink":
				coverageWrite(t, root, "target", "content\n")
				if err := os.Symlink("target", path); err != nil {
					t.Fatal(err)
				}
			case "executable":
				coverageWrite(t, root, "README.md", "content\n")
				if err := os.Chmod(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			base := coverageSnapshot(t, root, []string{"README.md"})
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			head := coverageSnapshot(t, root, []string{"README.md"})
			for _, pair := range [][2]changerisk.Snapshot{{base, head}, {head, base}} {
				report := changerisk.Evaluate(pair[0], pair[1], changerisk.Config{PDGChange: changerisk.Low}, changerisk.APIReport{})
				if report.Risk != changerisk.Unknown {
					t.Fatalf("unsafe documentation type received a low rating: %+v", report)
				}
			}
		})
	}
}

func coverageSnapshot(t *testing.T, root string, changed []string) changerisk.Snapshot {
	t.Helper()
	paths, message, err := ExistingPaths(root, changed)
	if err != nil {
		t.Fatal(err)
	}
	source, err := ExtractSource(context.Background(), root, paths)
	if err != nil {
		t.Fatal(err)
	}
	source.AnalysisError = strings.TrimSpace(strings.Join([]string{source.AnalysisError, message}, "\n"))
	return source
}

func coverageWrite(t *testing.T, root, path, content string) {
	t.Helper()
	name := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
