package changerisk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/changerisk"
)

func TestRiskCheckRejectsDocumentationEmbeddedInExcludedBuilds(t *testing.T) {
	for _, tc := range []struct{ name, file, constraint string }{
		{"windows", "assets_windows.go", ""},
		{"build tag", "assets.go", "//go:build embedcoverageexcluded\n\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GOOS", "linux")
			root := repository(t)
			write(t, root, "gstyle.toml", "[risk]\npdg_change='low'\ndependency_change='low'\n")
			write(t, root, "source.go", "package sample\nfunc Count() int { return 1 }\n")
			write(t, root, tc.file, tc.constraint+"package sample\nimport _ \"embed\"\n//go:embed docs/policy.md\nvar Policy string\n")
			writeEmbedFixture(t, root, "docs/policy.md", "before\n")
			ignored, err := command(context.Background(), root, "go", "list", "-mod=readonly", "-f", "{{.IgnoredGoFiles}}", ".")
			if err != nil || !strings.Contains(string(ignored), tc.file) {
				t.Fatalf("fixture source is not excluded from the current build: %s error=%v", ignored, err)
			}
			base := commit(t, root)
			writeEmbedFixture(t, root, "docs/policy.md", "after\n")
			head := commit(t, root)
			t.Chdir(root)
			var output bytes.Buffer
			err = Run(context.Background(), []string{"--base", base, "--head", head, "--check", "--format", "json"}, &output)
			if !errors.Is(err, ErrReviewRequired) {
				t.Fatalf("excluded embed passed the gate: %v\n%s", err, output.String())
			}
			var report changerisk.Report
			if err := json.Unmarshal(output.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if report.Risk != changerisk.Unknown || !strings.Contains(output.String(), "changed embedded resources need review across Go build configurations: docs/policy.md") {
				t.Fatalf("excluded embed evidence is missing: %s", output.String())
			}
			if report.DependenciesChanged {
				t.Fatalf("excluded embed invented a dependency change: %s", output.String())
			}
		})
	}
}

func TestRiskCheckAllowsDocumentationWithUnrelatedEmbeds(t *testing.T) {
	root := repository(t)
	write(t, root, "gstyle.toml", "[risk]\npdg_change='low'\ndependency_change='low'\n")
	write(t, root, "source.go", "package sample\nfunc Count() int { return 1 }\n")
	writeEmbedFixture(t, root, "application/assets.go", "package application\nimport _ \"embed\"\n//go:embed openrpc.json\nvar Schema string\n")
	writeEmbedFixture(t, root, "application/openrpc.json", "{}\n")
	write(t, root, "assets_windows.go", "package sample\nimport _ \"embed\"\n//go:embed docs/policy.md\nvar Policy string\n")
	writeEmbedFixture(t, root, "docs/policy.md", "unchanged\n")
	writeEmbedFixture(t, root, "docs/guide.md", "before\n")
	write(t, root, "README.md", "before\n")
	base := commit(t, root)
	writeEmbedFixture(t, root, "docs/guide.md", "after\n")
	write(t, root, "README.md", "after\n")
	head := commit(t, root)
	t.Chdir(root)
	var output bytes.Buffer
	if err := Run(context.Background(), []string{"--base", base, "--head", head, "--check", "--format", "json"}, &output); err != nil {
		t.Fatalf("unrelated embeds blocked documentation: %v\n%s", err, output.String())
	}
	var report changerisk.Report
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Risk != changerisk.Low || strings.Contains(output.String(), "embedded resources") {
		t.Fatalf("unrelated embeds affected documentation: %s", output.String())
	}
}

func TestEmbeddedDocumentationCoverageScansBothSnapshots(t *testing.T) {
	for _, change := range []string{"add", "remove", "rename"} {
		t.Run(change, func(t *testing.T) {
			root := repository(t)
			writeEmbedFixture(t, root, "docs/policy.md", "before\n")
			source := "package sample\nimport _ \"embed\"\n//go:embed docs/policy.md\nvar Policy string\n"
			if change != "add" {
				write(t, root, "assets_windows.go", source)
			}
			base := commit(t, root)
			if change == "add" {
				write(t, root, "assets_windows.go", source)
			} else if err := os.Remove(filepath.Join(root, "assets_windows.go")); err != nil {
				t.Fatal(err)
			}
			if change == "rename" {
				writeEmbedFixture(t, root, "other/assets_windows.go", source)
			}
			writeEmbedFixture(t, root, "docs/policy.md", "after\n")
			head := commit(t, root)
			for _, revision := range []struct {
				commit string
				match  bool
			}{{base, change != "add"}, {head, change == "add"}} {
				err := withSnapshot(context.Background(), root, revision.commit, func(snapshot string) error {
					message, err := embeddedDocumentationCoverage(context.Background(), snapshot, []string{"docs/policy.md"})
					if err != nil || (message != "") != revision.match {
						t.Fatalf("snapshot embed scope: message=%q error=%v wantMatch=%t", message, err, revision.match)
					}
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestEmbedPatternCoverage(t *testing.T) {
	for _, tc := range []struct {
		name, source, patterns, document string
		want                             bool
	}{
		{"file", "assets_windows.go", "docs/policy.md", "docs/policy.md", true},
		{"directory", "assets_windows.go", "docs", "docs/nested/policy.md", true},
		{"directory wildcard", "assets_windows.go", "doc*", "docs/nested/policy.md", true},
		{"file wildcard", "assets_windows.go", "docs/*.md", "docs/policy.md", true},
		{"wildcard depth", "assets_windows.go", "docs/*.md", "docs/nested/policy.md", false},
		{"matched directory suffix", "assets_windows.go", "docs/*.md", "docs/archive.md/policy.md", true},
		{"double quote", "assets_windows.go", "\"docs/policy notes.md\"", "docs/policy notes.md", true},
		{"raw quote", "assets_windows.go", "`docs/policy notes.md`", "docs/policy notes.md", true},
		{"escaped space", "assets_windows.go", "\"docs/policy\\u0020notes.md\"", "docs/policy notes.md", true},
		{"all prefix", "assets_windows.go", "all:docs", "docs/.private/_policy.md", true},
		{"all quoted", "assets_windows.go", "\"all:docs/policy notes.md\"", "docs/policy notes.md", true},
		{"multiple patterns", "assets_windows.go", "README.md\t\"docs/policy notes.md\"\u2003docs/policy.md", "docs/policy.md", true},
		{"relative owner", "pkg/assets_windows.go", "docs", "pkg/docs/policy.md", true},
		{"outside owner", "pkg/assets_windows.go", "docs", "docs/policy.md", false},
		{"owner with trailing dot", "pkg./assets_windows.go", "docs", "pkg./docs/policy.md", true},
		{"literal owner glob", "pkg[1]/assets_windows.go", "docs", "pkg[1]/docs/policy.md", true},
		{"unrelated embed", "application/assets.go", "openrpc.json", "README.md", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			patterns, err := scanEmbedPatterns(tc.source, []byte("package sample\n//go:embed\t"+tc.patterns+"\nvar Text string\n"))
			if err != nil {
				t.Fatal(err)
			}
			matches, err := matchingEmbeddedDocuments(tc.source, patterns, []string{tc.document})
			if err != nil || (len(matches) != 0) != tc.want {
				t.Fatalf("matches=%v error=%v want=%t", matches, err, tc.want)
			}
		})
	}
}

func TestScanEmbedPatternsIgnoresCommentLikeLiteralsAndInvalidSyntax(t *testing.T) {
	for _, source := range []string{
		"package sample\nvar Text = `//go:embed docs/policy.md`\n",
		"package sample\nvar Text = `\n//go:embed docs/policy.md\n`\n",
		"package sample\nvar Text = \"//go:embed docs/policy.md\"\n",
		"package sample\n// ordinary //go:embed docs/policy.md\n",
		"package sample\n// go:embed docs/policy.md\n",
		"package sample\n/*go:embed docs/policy.md*/\n",
		"package sample\n//go:embedder docs/policy.md\n",
		"package sample\nfunc invalid( {\n",
	} {
		patterns, err := scanEmbedPatterns("testdata/source.go", []byte(source))
		if err != nil || len(patterns) != 0 {
			t.Errorf("non-directive became embed coverage: %q patterns=%v error=%v", source, patterns, err)
		}
	}
}

func TestEmbeddedDocumentationCoverageAllowsUnrelatedInvalidTestdata(t *testing.T) {
	root := repository(t)
	writeEmbedFixture(t, root, "testdata/invalid.go", "package sample\nfunc invalid( {\n")
	write(t, root, "source.go", "package sample\nvar Text = `//go:embed README.md`\n")
	git(t, root, "add", ".")
	message, err := embeddedDocumentationCoverage(context.Background(), root, []string{"README.md", "docs/guide.md"})
	if err != nil || message != "" {
		t.Fatalf("unrelated invalid syntax requires documentation review: message=%q error=%v", message, err)
	}
}

func TestScanEmbedPatternsRejectsInvalidDirectives(t *testing.T) {
	for _, arguments := range []string{
		"", " ", ".", "../policy.md", "/docs/policy.md", "docs//policy.md", "docs/./policy.md", "docs/../policy.md",
		"docs/[", "all:", "all:../docs", "\"\"", "\"docs/policy.md", "`docs/policy.md", "\"docs/policy.md\"suffix",
		"`docs/policy.md`suffix", "\"docs\\\\policy.md\"", "\"docs/\\qpolicy.md\"",
	} {
		if _, err := scanEmbedPatterns("assets_windows.go", []byte("package sample\n//go:embed "+arguments+"\nvar Text string\n")); err == nil {
			t.Errorf("invalid embed pattern accepted: %q", arguments)
		}
	}
	if _, err := scanEmbedPatterns("assets_windows.go", []byte("package sample\nvar Text = `unterminated")); err == nil {
		t.Fatal("unknown token coverage is accepted")
	}
}

func TestEmbeddedDocumentationCoverageFailsClosed(t *testing.T) {
	root := repository(t)
	write(t, root, "assets_windows.go", "package sample\n//go:embed docs/[\nvar Policy string\n")
	git(t, root, "add", ".")
	message := embeddedDocumentationScope(context.Background(), root, []string{"README.md"})
	if !strings.Contains(message, "Go embed coverage is unavailable") {
		t.Fatalf("invalid excluded directive did not require review: %q", message)
	}
	if err := os.Remove(filepath.Join(root, "assets_windows.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := embeddedDocumentationCoverage(context.Background(), root, []string{"README.md"}); err == nil {
		t.Fatal("missing inventoried Go source is accepted")
	}
	if _, err := embeddedDocumentationCoverage(context.Background(), t.TempDir(), []string{"README.md"}); err == nil {
		t.Fatal("unavailable Git inventory is accepted")
	}
	if message, err := embeddedDocumentationCoverage(context.Background(), t.TempDir(), []string{"source.go"}); err != nil || message != "" {
		t.Fatalf("non-documentation requires an embed inventory: message=%q error=%v", message, err)
	}
	if _, err := embeddedDocumentationCoverage(context.Background(), root, []string{"../policy.md"}); err == nil {
		t.Fatal("escaping documentation path is accepted")
	}
}

func TestReadEmbedSourceRejectsUnsafePathsAndModes(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"../outside.go", "/outside.go", "pkg\\source.go", "pkg/../source.go", "source.txt"} {
		if _, err := readEmbedSource(root, name); err == nil {
			t.Errorf("unsafe Go inventory path is accepted: %q", name)
		}
	}
	write(t, root, "source.go", "package sample\n")
	if err := os.Symlink(filepath.Join(root, "source.go"), filepath.Join(root, "link.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := readEmbedSource(root, "link.go"); err == nil {
		t.Fatal("symlinked Go inventory source is accepted")
	}
	if err := os.Symlink(root, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	if _, err := readEmbedSource(root, "linked/source.go"); err == nil {
		t.Fatal("symlinked Go inventory parent is accepted")
	}
	if err := os.Mkdir(filepath.Join(root, "directory.go"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := readEmbedSource(root, "directory.go"); err == nil {
		t.Fatal("directory Go inventory source is accepted")
	}
}

func TestEmbeddedDocumentationMessageIsDeterministic(t *testing.T) {
	message := embeddedDocumentationMessage([]string{"docs/z.md", "README.md", "docs/z.md"})
	if message != "changed embedded resources need review across Go build configurations: README.md, docs/z.md" {
		t.Fatalf("unexpected embed scope: %q", message)
	}
}

func writeEmbedFixture(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(name)), 0700); err != nil {
		t.Fatal(err)
	}
	write(t, root, name, content)
}
