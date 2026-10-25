package sarif_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/fredbi/uri"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type document struct {
	Version string `json:"version"`
	Runs    []run  `json:"runs"`
}

type run struct {
	Tool struct {
		Driver struct{ Name string } `json:"driver"`
	} `json:"tool"`
	AutomationDetails  struct{ ID string }             `json:"automationDetails"`
	OriginalURIBaseIDs map[string]struct{ URI string } `json:"originalUriBaseIds"`
	Results            []result                        `json:"results"`
}

type result struct {
	RuleID    string                `json:"ruleId"`
	Message   struct{ Text string } `json:"message"`
	Locations []struct {
		PhysicalLocation struct {
			Region           struct{ StartLine int } `json:"region"`
			ArtifactLocation struct {
				URI, URIBaseID string
			} `json:"artifactLocation"`
		} `json:"physicalLocation"`
	} `json:"locations"`
}

func TestSARIFSchemaAndCLI(t *testing.T) {
	schema := loadSchema(t)
	root := t.TempDir()
	binary := filepath.Join(root, "gstyle")
	command(t, "../..", "go", []string{"build", "-buildvcs=false", "-o", binary, "."}, 0)
	writeFile(t, filepath.Join(root, "go.mod"), []byte("module example.com/sariftest\n\ngo 1.26.0\n"))
	source := filepath.Join(root, "input #é.go")
	writeFile(t, source, []byte("package sample\nfunc Clean() {}\n"))
	for _, name := range []string{"check", "patterns"} {
		if got := cliReport(t, schema, binary, root, []string{name, "--format", "sarif", "."}, 0); len(got.Results) != 0 {
			t.Fatalf("clean %s: %+v", name, got.Results)
		}
	}
	writeFile(t, source, []byte("package sample\nfunc Bad(err error) error {\nif err != nil { return nil }; return err\n}\nfunc Many(a,b,c,d,e,f,g,h int) int { return a+b+c+d+e+f+g+h }\n"))
	quality := cliReport(t, schema, binary, root, []string{"check", "--format", "sarif", "."}, 1)
	patterns := cliReport(t, schema, binary, root, []string{"patterns", "--format", "sarif", "."}, 0)
	requireRule(t, quality, "fn_params")
	requireRule(t, patterns, "nilerr")
	for _, item := range append(slices.Clone(quality.Results), patterns.Results...) {
		uri := item.Locations[0].PhysicalLocation.ArtifactLocation.URI
		if !strings.Contains(uri, "%20%23%C3%A9.go") {
			t.Fatalf("unescaped source URI: %s", uri)
		}
	}
	facts := command(t, root, binary, []string{"check", "--format", "facts", "."}, 0)
	writeFile(t, filepath.Join(root, "facts.json"), facts)
	replay := cliReport(t, schema, binary, root, []string{"check", "--facts-in", "facts.json", "--format", "sarif"}, 1)
	if !reflect.DeepEqual(quality.Results, replay.Results) {
		t.Fatal("saved-fact SARIF differs from direct results")
	}
	command(t, root, "git", []string{"init"}, 0)
	writeFile(t, filepath.Join(root, "gstyle.toml"), []byte("[guard.comments]\nmode='all'\n[[guard.files]]\nid='python'\nextensions=['.py']\n"))
	if got := cliReport(t, schema, binary, root, []string{"guard", "--format", "sarif"}, 0); len(got.Results) != 0 {
		t.Fatalf("clean guard: %+v", got.Results)
	}
	writeFile(t, filepath.Join(root, "tool.py"), []byte("pass\n"))
	requireRule(t, cliReport(t, schema, binary, root, []string{"guard", "--format", "sarif"}, 1), "python")
}

func loadSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	data, err := os.ReadFile("schema.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != "c3b4bb2d6093897483348925aaa73af03b3e3f4bd4ca38cef26dcb4212a2682e" {
		t.Fatalf("OASIS schema changed: %s", got)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft4)
	compiler.AssertFormat()
	compiler.RegisterFormat(&jsonschema.Format{Name: "uri-reference", Validate: func(value any) error {
		text, ok := value.(string)
		if !ok {
			return nil
		}
		_, err := uri.ParseReference(text)
		return err
	}})
	compiler.RegisterFormat(&jsonschema.Format{Name: "uri", Validate: func(value any) error {
		text, ok := value.(string)
		if !ok {
			return nil
		}
		_, err := uri.Parse(text)
		return err
	}})
	schema, err := compiler.Compile("schema.json")
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

func cliReport(t *testing.T, schema *jsonschema.Schema, binary, root string, args []string, expectedExit int) run {
	t.Helper()
	data := command(t, root, binary, args, expectedExit)
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(instance); err != nil {
		t.Fatalf("invalid SARIF: %v", err)
	}
	var report document
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Version != "2.1.0" || len(report.Runs) != 1 {
		t.Fatalf("unexpected SARIF report: %s", data)
	}
	got := report.Runs[0]
	if got.Tool.Driver.Name != "gstyle" || got.AutomationDetails.ID != "gstyle/"+args[0]+"/" {
		t.Fatalf("unexpected driver/category: %+v", got)
	}
	uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(root) + "/"}).String()
	if got.OriginalURIBaseIDs["%SRCROOT%"].URI != uri {
		t.Fatalf("source-root URI: %+v", got.OriginalURIBaseIDs)
	}
	for _, item := range got.Results {
		checkResult(t, item)
	}
	return got
}

func checkResult(t *testing.T, item result) {
	t.Helper()
	if item.RuleID == "" || strings.TrimSpace(item.Message.Text) == "" || len(item.Locations) != 1 {
		t.Fatalf("missing finding identity/message/location: %+v", item)
	}
	physical := item.Locations[0].PhysicalLocation
	if physical.Region.StartLine <= 0 {
		t.Fatalf("nonpositive source line: %+v", physical)
	}
	artifact := physical.ArtifactLocation
	if artifact.URIBaseID != "%SRCROOT%" && !strings.HasPrefix(artifact.URI, "file://") {
		t.Fatalf("missing artifact base: %+v", artifact)
	}
}

func requireRule(t *testing.T, report run, id string) {
	t.Helper()
	if !slices.ContainsFunc(report.Results, func(item result) bool { return item.RuleID == id }) {
		t.Fatalf("missing %s: %+v", id, report.Results)
	}
}

func command(t *testing.T, root, name string, args []string, expectedExit int) []byte {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != expectedExit {
		t.Fatalf("%s %v: %v; want exit %d\n%s", name, args, err, expectedExit, stderr.String())
	}
	return output
}

func writeFile(t *testing.T, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(name, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestSchemaRejectsInvalidReports(t *testing.T) {
	schema := loadSchema(t)
	for _, control := range []struct{ input, before, after string }{
		{`{"version":"2.0","runs":[]}`, `"2.0"`, `"2.1.0"`},
		{`{"version":"2.1.0","runs":[{"tool":{}}]}`, `"tool":{}`, `"tool":{"driver":{"name":"gstyle"}}`},
		{`{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"gstyle"}},"originalUriBaseIds":{"%SRCROOT%":{"uri":"not a URI"}}}]}`, "not a URI", "file:///tmp/"},
		{`{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"gstyle"}},"results":[{"message":{"text":"bad"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"file.go"},"region":{"startLine":0}}}]}]}]}`, `"startLine":0`, `"startLine":1`},
	} {
		instance, err := jsonschema.UnmarshalJSON(strings.NewReader(control.input))
		if err != nil {
			t.Fatal(err)
		}
		if err := schema.Validate(instance); err == nil {
			t.Fatalf("invalid report accepted: %s", control.input)
		}
		valid, err := jsonschema.UnmarshalJSON(strings.NewReader(strings.Replace(control.input, control.before, control.after, 1)))
		if err != nil {
			t.Fatal(err)
		}
		if err := schema.Validate(valid); err != nil {
			t.Fatalf("repaired control is still invalid: %v", err)
		}
	}
}
