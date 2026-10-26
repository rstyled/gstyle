package gorisk

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/changerisk"
)

func TestAuxiliaryTestCoverageIsAdditiveAndHasNoStartupHooks(t *testing.T) {
	for _, test := range []struct {
		name, source string
		accepted     bool
	}{
		{"ordinary", "package sample; import \"testing\"; func TestValue(t *testing.T) { t.Fatal(1) }", true},
		{"helpers", "package sample; func helper() int { return 1 }; func BenchmarkValue() {}", true},
		{"syntax", "package", false},
		{"startup", "package sample; func init() {}", false},
		{"test main", "package sample; func TestMain() {}", false},
		{"initializer", "package sample; var value = run()", false},
		{"empty variable", "package sample; var value int", false},
		{"compiler directive", "package sample\n//go:linkname value other\nfunc value() {}", false},
		{"build tag", "//go:build windows\n\npackage sample", false},
		{"line directive", "package sample\n//line file.go:10\nfunc helper() {}", false},
		{"blank import", "package sample; import _ \"os\"", false},
		{"dot import", "package sample; import . \"os\"", false},
		{"cgo", "package sample; import \"C\"", false},
		{"duplicate", "package sample; func helper() {}; func helper() {}", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			accepted := testFile("value_test.go", []byte(test.source))
			if accepted != test.accepted {
				t.Fatalf("accepted=%v, want %v", accepted, test.accepted)
			}
		})
	}
}

func TestAuxiliaryRejectsUnsafePathsAndModes(t *testing.T) {
	for _, kind := range []string{"executable", "symlink", "parent symlink", "nested module", "directory"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			name := "value_test.go"
			target := filepath.Join(root, name)
			coverageWrite(t, root, name, "package sample; func TestValue() {}")
			switch kind {
			case "executable":
				if err := os.Chmod(target, 0700); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Rename(target, target+".source"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target+".source", target); err != nil {
					t.Fatal(err)
				}
			case "parent symlink":
				if err := os.Symlink(root, filepath.Join(root, "linked")); err != nil {
					t.Fatal(err)
				}
				name = "linked/" + name
			case "nested module":
				coverageWrite(t, root, "nested/go.mod", "module nested\n")
				coverageWrite(t, root, "nested/"+name, "package nested")
				name = "nested/" + name
			case "directory":
				if err := os.Remove(target); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(target, 0700); err != nil {
					t.Fatal(err)
				}
			}
			result, uncovered := extractAuxiliary(root, []string{name})
			if len(result.PreservedDeclarations) != 0 || len(uncovered) != 1 {
				t.Fatalf("unsafe path covered: %+v", result)
			}
		})
	}
	for _, path := range []string{"../value_test.go", "/value_test.go", "a\\value_test.go"} {
		if _, ok := readAuxiliary(t.TempDir(), path); ok {
			t.Fatalf("unsafe path %q covered", path)
		}
	}
}

func TestQualityBaselineCoverageValidatesSchemaAndRates(t *testing.T) {
	baseline := changerisk.QualityBaseline{Version: 1, Functions: 100, Tolerance: 0.05,
		Rules: map[string]changerisk.BaselineRule{"density": {Limit: 500, Excess: 10, Rate: 100}}}
	content, err := json.Marshal(baseline)
	if err != nil {
		t.Fatal(err)
	}
	if qualityBaseline(content) == nil {
		t.Fatal("valid baseline rejected")
	}
	for _, invalid := range []string{
		strings.Replace(string(content), `"version":1`, `"version":2`, 1),
		strings.Replace(string(content), `"functions":100`, `"functions":0`, 1),
		strings.Replace(string(content), `"tolerance":0.05`, `"tolerance":-1`, 1),
		strings.Replace(string(content), `"excess":10`, `"excess":-10`, 1),
		strings.Replace(string(content), `"limit":500`, `"limit":-1`, 1),
		strings.Replace(string(content), `"per_1000_functions":100`, `"per_1000_functions":99`, 1),
		strings.Replace(string(content), `"version":1`, `"extra":0,"version":1`, 1),
		string(content) + ` {}`, `{}`, `null`,
	} {
		if qualityBaseline([]byte(invalid)) != nil {
			t.Fatalf("invalid baseline accepted: %s", invalid)
		}
	}
}

func TestExistingTestFileEditsRequireReview(t *testing.T) {
	const source = "package sample\nfunc ExampleValue() {\nprintln(1)\n// Output: 1\n}\n"
	root := t.TempDir()
	coverageWrite(t, root, "value_test.go", source)
	before, _ := extractAuxiliary(root, []string{"value_test.go"})
	for _, after := range []string{
		strings.Replace(source, "Output: 1", "Output: 2", 1),
		strings.Replace(source, "println(1)", "println(2)", 1),
		source + "func TestExtra() {}\n",
	} {
		coverageWrite(t, root, "value_test.go", after)
		head, _ := extractAuxiliary(root, []string{"value_test.go"})
		report := changerisk.Evaluate(before, head, changerisk.Config{PDGChange: changerisk.Low}, changerisk.APIReport{})
		if report.Risk != changerisk.Unknown {
			t.Fatal("existing test edit received a low rating")
		}
	}
}
