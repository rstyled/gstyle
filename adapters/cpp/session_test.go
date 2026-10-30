package cpp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func sessionFixture(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "export.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCPPPendingScanKeepsItsSnapshotAcrossRefresh(t *testing.T) {
	path := sessionFixture(t)
	analyzer := Analyzer{}.Session()
	if _, err := analyzer.Analyze([]string{path}); err != nil {
		t.Fatal(err)
	}
	load := analyzer.PatternScan(context.Background(), []string{path}, patterns.DefaultConfig())
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := analyzer.Analyze([]string{path}); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("refresh unexpectedly succeeds", err)
	}
	if _, err := load(); err != nil {
		t.Fatal("refresh replaces the already prepared scan's snapshot", err)
	}
	if _, err := analyzer.Patterns(context.Background(), []string{path}); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a new scan receives the previous report's snapshot", err)
	}
}

func TestCPPSessionUsesOneSnapshotAndRefreshReadsAgain(t *testing.T) {
	path := sessionFixture(t)
	analyzer := Analyzer{}.Session()
	report, err := analyzer.Analyze([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	patterns, err := analyzer.Patterns(context.Background(), []string{path})
	if err != nil || report.Functions == 0 || len(patterns.AnalysisCoverage) == 0 {
		t.Fatalf("patterns reread the export instead of using the verified snapshot: %v", err)
	}
	if _, err := analyzer.Patterns(context.Background(), []string{path}); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the consumed snapshot remains cached: %v", err)
	}
	if _, err := analyzer.Analyze([]string{path}); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("refresh does not read the current export: %v", err)
	}
}

func TestCPPSessionHonorsRootAndCancellation(t *testing.T) {
	path := sessionFixture(t)
	analyzer := Analyzer{}.Session()
	if _, err := analyzer.Analyze([]string{path}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := analyzer.Patterns(ctx, []string{path}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cached extraction ignores cancellation: %v", err)
	}
	other := analyzer
	other.SourceRoot = t.TempDir()
	if _, err := other.Patterns(context.Background(), []string{path}); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the snapshot crosses source roots: %v", err)
	}
	if _, err := analyzer.Patterns(context.Background(), []string{path}); err != nil {
		t.Fatalf("a canceled or unrelated request consumed the snapshot: %v", err)
	}
}
