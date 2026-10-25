package gorisk

import (
	"context"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/changerisk"
)

func TestTypedAPIComparison(t *testing.T) {
	cases := []struct {
		name, before, after string
		want                changerisk.APIStatus
	}{
		{"body only", "func Count(v int) int { return v+1 }", "func Count(v int) int { return v+2 }", changerisk.APIUnchanged},
		{"parameter name", "func Count(first int) int { return first }", "func Count(second int) int { return second }", changerisk.APIUnchanged},
		{"updated callers", "func Save(v int) {}\nfunc caller() { Save(1) }", "func Save(v string) {}\nfunc caller() { Save(\"one\") }", changerisk.APIBreaking},
		{"struct field", "type Options struct { Value int }", "type Options struct { Value string }", changerisk.APIBreaking},
		{"interface method", "type Reader interface { Read() error }", "type Reader interface { Read() error; Close() error }", changerisk.APIBreaking},
		{"method signature", "type Handler struct{}\nfunc (Handler) Run(v int) {}", "type Handler struct{}\nfunc (Handler) Run(v string) {}", changerisk.APIBreaking},
		{"constant", "const Size = 1", "const Size = 2", changerisk.APIBreaking},
		{"removed variable", "var Setting bool", "var setting bool", changerisk.APIBreaking},
		{"addition", "func Count() {}", "func Count() {}\nfunc Added() {}", changerisk.APICompatible},
		{"type alias", "type Options = struct { Value int }", "type Options = struct { Value string }", changerisk.APIBreaking},
		{"generic constraint narrowed", "func Apply[T any](v T) T { return v }", "func Apply[T ~int](v T) T { return v }", changerisk.APIPartial},
		{"generic constraint widened", "func Apply[T ~int](v T) T { return v }", "func Apply[T any](v T) T { return v }", changerisk.APIPartial},
		{"generic body only", "func Apply[T any](v T) T { return v }", "func Apply[T any](v T) T { next:=v; return next }", changerisk.APIUnchanged},
		{"generic parameter names", "func Apply[T any](v T) T { return v }", "func Apply[U any](value U) U { return value }", changerisk.APIUnchanged},
		{"generic type constraint", "type Box[T any] struct { Value T }", "type Box[T ~int] struct { Value T }", changerisk.APIPartial},
		{"constraint interface", "type Number interface { ~int }", "type Number interface { ~int | ~string }", changerisk.APIPartial},
		{"private constraint", "type number interface { ~int }; func Apply[T number](v T) T {return v}", "type number interface { ~int | ~string }; func Apply[T number](v T) T {return v}", changerisk.APIPartial},
		{"embedded private constraint", "type inner interface { ~int }; type number interface {inner}; func Apply[T number](v T) T {return v}", "type inner interface { ~int | ~string }; type number interface {inner}; func Apply[T number](v T) T {return v}", changerisk.APIPartial},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			base := checkedAPI(t, "package sample\n"+tc.before)
			head := checkedAPI(t, "package sample\n"+tc.after)
			report := CompareAPI(base, head)
			if report.Status != tc.want || report.PackagesBefore != 1 || report.PackagesAfter != 1 || report.Scope != changerisk.APIScope {
				t.Fatalf("API comparison: %+v", report)
			}
			if repeat := CompareAPI(base, head); changerisk.Markdown(changerisk.Report{GoAPI: repeat}) != changerisk.Markdown(changerisk.Report{GoAPI: report}) {
				t.Fatal("API rendering is not deterministic")
			}
		})
	}
}

func TestPackageNameChangesAreBreaking(t *testing.T) {
	base := checkedAPI(t, "package before\nfunc Exported() {}")
	head := checkedAPI(t, "package after\nfunc Exported() {}")
	report := CompareAPI(base, head)
	if report.Status != changerisk.APIBreaking || len(report.Changes) != 1 || !strings.Contains(report.Changes[0].Message, "package name changed") {
		t.Fatalf("default import name break disappeared: %+v", report)
	}
}

func checkedAPI(t *testing.T, source string) API {
	t.Helper()
	root := sourceModule(t, map[string]string{"api.go": source})
	api := ExtractAPI(context.Background(), root)
	if api.Error != "" {
		t.Fatalf("API type load: %s", api.Error)
	}
	return api
}

func TestAPIPackageRemovalAndCommandScope(t *testing.T) {
	root := sourceModule(t, map[string]string{"api.go": "package sample\nfunc Exported() {}\n"})
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "api.go"), []byte("package child\ntype Options struct{Value int}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	base := ExtractAPI(context.Background(), root)
	if err := os.RemoveAll(child); err != nil {
		t.Fatal(err)
	}
	head := ExtractAPI(context.Background(), root)
	report := CompareAPI(base, head)
	if report.Status != changerisk.APIBreaking || len(report.Changes) != 1 || report.Changes[0].Message != "package removed" {
		t.Fatalf("removed package escaped API comparison: %+v", report)
	}
	reverse := CompareAPI(head, base)
	if reverse.Status != changerisk.APICompatible || len(reverse.Changes) != 1 || !reverse.Changes[0].Compatible {
		t.Fatalf("added package reported as breaking: %+v", reverse)
	}
	command := checkedAPI(t, "package main\nfunc main() {}\nfunc Exported() {}")
	if len(command.Packages) != 0 {
		t.Fatal("command package treated as an importable API")
	}
}

func TestAPILoadFailuresCannotBecomeCompatibilityPasses(t *testing.T) {
	valid := checkedAPI(t, "package sample\nfunc Exported() {}")
	root := sourceModule(t, map[string]string{"api.go": "package sample\nfunc Broken() { missing() }"})
	invalid := ExtractAPI(context.Background(), root)
	if invalid.Error == "" {
		t.Fatal("type error disappeared")
	}
	for _, pair := range [][2]API{{valid, invalid}, {invalid, valid}, {{}, valid}} {
		report := CompareAPI(pair[0], pair[1])
		if report.Status != changerisk.APIUnavailable || report.Error == "" || strings.Contains(report.Error, root) {
			t.Fatalf("incomplete inventory became a pass or leaked its checkout path: %+v", report)
		}
	}
}

func TestAPIChangesRetainTypedDeclarationsAndPhysicalSources(t *testing.T) {
	base := checkedAPI(t, "package sample\nfunc Save(value int) {}\ntype Options struct { Value int }\ntype Handler struct{}\nfunc (*Handler) Run(value int) {}\nconst Limit = 1\n")
	head := checkedAPI(t, "package sample\nfunc Save(value string) {}\ntype Options struct { Value string }\ntype Handler struct{}\nfunc (*Handler) Run(value string) {}\n")
	report := CompareAPI(base, head)
	for _, change := range report.Changes {
		if change.Before == nil || change.Before.Path != "api.go" || change.Before.Line < 2 || change.Before.Declaration == "" {
			t.Errorf("missing original declaration: %+v", change)
		}
		if strings.HasPrefix(change.Message, "Limit:") {
			if change.After != nil || change.Before.Line != 6 {
				t.Errorf("removed constant has the wrong source: %+v", change)
			}
			continue
		}
		if change.After == nil || change.After.Path != "api.go" || change.After.Line < 2 || !strings.Contains(change.After.Declaration, "string") {
			t.Errorf("missing changed declaration: %+v", change)
		}
	}
	if len(report.Changes) != 4 {
		t.Fatalf("expected function, field, method and constant changes: %+v", report)
	}
}

func TestNewAPIPackageIncludesDirectoryAndExportCount(t *testing.T) {
	head := checkedAPI(t, "package sample\nfunc Added() {}\ntype Options struct{}\nvar hidden bool\n")
	report := CompareAPI(API{Packages: map[string]*types.Package{}}, head)
	change := report.Changes[0]
	if change.Before != nil || change.After == nil || change.After.Path != "." || change.After.Exports != 2 {
		t.Fatalf("new package lacks useful source evidence: %+v", change)
	}
}

func TestAPILinksIgnoreLogicalLineDirectives(t *testing.T) {
	base := checkedAPI(t, "package sample\n//line spoofed.go:900\nfunc Save(value int) {}\n")
	head := checkedAPI(t, "package sample\n//line spoofed.go:900\nfunc Save(value string) {}\n")
	change := CompareAPI(base, head).Changes[0]
	for _, declaration := range []*changerisk.APIDeclaration{change.Before, change.After} {
		if declaration == nil || declaration.Path != "api.go" || declaration.Line != 3 {
			t.Fatalf("logical source position leaked into a review link: %+v", declaration)
		}
	}
}
