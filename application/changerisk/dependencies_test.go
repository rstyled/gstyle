package changerisk

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDependencyFailureRetainsItsCauseWithoutCheckoutPaths(t *testing.T) {
	var previous string
	for range 2 {
		root := repository(t)
		write(t, root, "source.go", "package sample\nimport _ \"example.com/missing\"\n")
		git(t, root, "add", ".")
		snapshot, err := populateSnapshot(context.Background(), root, nil)
		if err != nil {
			t.Fatal(err)
		}
		message := snapshot.Source.DependencyError
		if !strings.Contains(message, "example.com/missing") || strings.Contains(message, root) {
			t.Fatalf("missing dependency cause or leaked checkout path: %s", message)
		}
		if previous != "" && previous != message {
			t.Fatalf("checkout paths changed the error: %q != %q", previous, message)
		}
		previous = message
	}
}

func TestModuleRecordsKeepReplacementsAndDiscardCheckoutPaths(t *testing.T) {
	a := []byte(`{"Path":"example.com/module","Version":"v1.0.0","GoVersion":"1.26","Dir":"/temporary/base","Replace":{"Path":"../replacement","Dir":"/temporary/base/replacement"}}`)
	b := []byte(`{"Path":"example.com/module","Version":"v1.0.0","GoVersion":"1.26","Dir":"/temporary/head","Replace":{"Path":"../replacement","Dir":"/temporary/head/replacement"}}`)
	left, err := moduleRecords(a)
	if err != nil {
		t.Fatal(err)
	}
	right, err := moduleRecords(b)
	if err != nil || !reflect.DeepEqual(left, right) {
		t.Fatalf("checkout paths changed dependencies: %v %v %v", left, right, err)
	}
	changed, err := moduleRecords([]byte(`{"Path":"example.com/module","Version":"v1.0.0","GoVersion":"1.26","Replace":{"Path":"../different"}}`))
	if err != nil || reflect.DeepEqual(left, changed) {
		t.Fatal("replacement change disappeared")
	}
	if _, err := moduleRecords([]byte(`{"Path":`)); err == nil {
		t.Fatal("malformed dependency output accepted")
	}
	for _, input := range []string{"", `{}`, `{"Path":"module","Replace":{}}`} {
		if _, err := moduleRecords([]byte(input)); err == nil {
			t.Errorf("incomplete dependency output accepted: %s", input)
		}
	}
}

func TestExistingReflectImportInAnotherPackageIsNotANewDependency(t *testing.T) {
	root := repository(t)
	if err := os.Mkdir(filepath.Join(root, "other"), 0700); err != nil {
		t.Fatal(err)
	}
	write(t, root, "gstyle.toml", "[risk]\npdg_change='medium'\npdg_low_risk_max_functions=3\ndependency_change='high'\napi_breaking='high'\n")
	write(t, root, "other/source.go", "package other\nimport \"reflect\"\nfunc Type(v any) reflect.Type { return reflect.TypeOf(v) }\n")
	write(t, root, "source.go", "package sample\nfunc Count(n int) int { return n+1 }\n")
	base := commit(t, root)
	write(t, root, "source.go", "package sample\nimport \"reflect\"\nfunc Count(n int) int { _=reflect.TypeOf(n); return n+1 }\n")
	head := commit(t, root)
	report, err := compareRevisions(context.Background(), root, options{base: base, head: head, config: "gstyle.toml"})
	if err != nil {
		t.Fatal(err)
	}
	if report.DependenciesChanged || report.DependencyRisk != "low" || len(report.DependenciesAdded)+len(report.DependenciesRemoved) != 0 {
		t.Fatalf("existing import usage became a new dependency: %+v", report)
	}
}
