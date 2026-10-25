package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMutationAssuranceRejectsUnsafeAnalyzers(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	baseline := buildAssuranceAnalyzer(t, root, "", "", "")
	checkAssuranceAnalyzer(t, root, baseline, false)
	for _, control := range []struct{ name, file, before, after string }{
		{"external-mutations-ignored", "domain/quality/evaluate.go",
			"if mutation.Provenance == Local {\n\t\t\tcontinue\n\t\t}",
			"if mutation.Provenance == Local || mutation.Provenance == External {\n\t\t\tcontinue\n\t\t}"},
		{"local-elements-owned", "adapters/goquality/mutation.go",
			"func referencedOrigin(input quality.Provenance) quality.Provenance {\n\tif input == quality.External {\n\t\treturn quality.External\n\t}\n\treturn quality.Unknown\n}",
			"func referencedOrigin(input quality.Provenance) quality.Provenance {\n\tif input == quality.External {\n\t\treturn quality.External\n\t}\n\treturn quality.Local\n}"},
		{"mixed-bindings-owned", "adapters/goquality/mutation.go",
			"if result != x.bindingValueOrigin(value, seen) {\n\t\t\treturn quality.Unknown\n\t\t}",
			"if result != x.bindingValueOrigin(value, seen) {\n\t\t\treturn quality.Local\n\t\t}"},
	} {
		t.Run(control.name, func(t *testing.T) {
			binary := buildAssuranceAnalyzer(t, root, control.file, control.before, control.after)
			checkAssuranceAnalyzer(t, root, binary, true)
		})
	}
}

func buildAssuranceAnalyzer(t *testing.T, root, name, before, after string) string {
	t.Helper()
	work := t.TempDir()
	args := []string{"build", "-buildvcs=false", "-o", filepath.Join(work, "gstyle")}
	if name != "" {
		source := filepath.Join(root, name)
		data, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Count(data, []byte(before)) != 1 {
			t.Fatalf("control must match one source site: %s", name)
		}
		replacement := filepath.Join(work, "replacement.go.txt")
		assuranceWrite(t, work, "replacement.go.txt", bytes.Replace(data, []byte(before), []byte(after), 1))
		overlay, err := json.Marshal(map[string]any{"Replace": map[string]string{source: replacement}})
		if err != nil {
			t.Fatal(err)
		}
		assuranceWrite(t, work, "overlay.json", overlay)
		args = append(args, "-overlay", filepath.Join(work, "overlay.json"))
	}
	command := exec.Command("go", append(args, ".")...)
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("control does not compile: %v\n%s", err, output)
	}
	return filepath.Join(work, "gstyle")
}

func checkAssuranceAnalyzer(t *testing.T, root, binary string, unsafe bool) {
	t.Helper()
	work := t.TempDir()
	command := exec.Command("go", "test", "-buildvcs=false", "./examples/quality-hunt", "-run", "^TestMutationAssurance$", "-count=1", "-v")
	command.Dir = root
	command.Env = append(os.Environ(), "GSTYLE_ASSURANCE_BINARY="+binary, "GSTYLE_ASSURANCE_OUTPUT="+work, "GSTYLE_ASSURANCE_CASE=")
	output, runErr := command.CombinedOutput()
	data, err := os.ReadFile(filepath.Join(work, "assessment.json"))
	if err != nil {
		t.Fatalf("missing semantic assessment: %v\n%s", err, output)
	}
	var assessment assuranceAssessment
	if err := json.Unmarshal(data, &assessment); err != nil {
		t.Fatal(err)
	}
	if assessment.Executions != 291 {
		t.Fatalf("control lost corpus coverage: %d", assessment.Executions)
	}
	if !unsafe {
		if runErr != nil || assessment.Missing != 0 {
			t.Fatalf("current analyzer fails runtime oracle: %v\n%s", runErr, output)
		}
		return
	}
	var exit *exec.ExitError
	if !errors.As(runErr, &exit) || exit.ExitCode() != 1 || assessment.Missing == 0 || !strings.Contains(string(output), "caller write missing:") {
		t.Fatalf("unsafe control survived or failed outside the write contract: %v\n%s", runErr, output)
	}
	t.Logf("unsafe analyzer rejected: %d missing write witnesses", assessment.Missing)
}
