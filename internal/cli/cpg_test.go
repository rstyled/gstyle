package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCPGCommands(t *testing.T) {
	fixture := "../../adapters/cpp/testdata/sample.graphson.json"
	for _, args := range [][]string{
		{"search", "--cpg", "--format", "json", fixture},
		{"patterns", "--cpg", "--format", "json", fixture},
		{"patterns", "--cpg", "--format", "pdg-json", fixture},
		{"patterns", "--cpg", "--format", "sarif", fixture},
	} {
		var output bytes.Buffer
		if err := run(args, &output); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if !json.Valid(output.Bytes()) {
			t.Fatalf("%v: invalid JSON", args)
		}
		if strings.Contains(output.String(), `"conformant"`) {
			t.Fatal("CPG import claims source conformance")
		}
		if args[0] == "patterns" && args[3] == "json" && !strings.Contains(output.String(), `"group": "general"`) {
			t.Fatal("C++ dependency findings lack their general pattern group")
		}
	}
	var output bytes.Buffer
	if err := run([]string{"search", "--cpg", "--format", "text", fixture}, &output); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"CC=", "COG="} {
		if !strings.Contains(output.String(), text) {
			t.Fatalf("missing %q", text)
		}
	}
}

func TestCPGDirectoryAndInvalidOptions(t *testing.T) {
	data, err := os.ReadFile("../../adapters/cpp/testdata/sample.graphson.json")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "export.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run([]string{"search", "--cpg", dir}, &output); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"search", "--cpg-source", dir, dir},
		{"patterns", "--cpg-source", dir, dir},
		{"search", "--cpg", "--cpg-source", filepath.Join(dir, "missing"), dir},
		{"patterns", "--cpg", "--cpg-source", filepath.Join(dir, "missing"), dir},
		{"search", "--cpg", "--tags", "test", dir},
		{"search", "--cpg", "--tests=false", dir},
		{"search", "--cpg", "--config", "missing", dir},
		{"search", "--cpg", dir, dir},
	} {
		if err := run(args, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if _, err := parseRunOptions([]string{"--cpg", "--config", "policy.toml", dir}); err != nil {
		t.Fatal("CPG TUI rejects a shared quality policy")
	}
	if _, err := parseRunOptions([]string{"--cpg-source", dir, dir}); err == nil {
		t.Fatal("TUI accepts a CPG source root without CPG input")
	}
	for _, command := range []string{"search", "patterns"} {
		if err := run([]string{command, "--cpg", "--cpg-source", dir, dir}, &bytes.Buffer{}); err != nil {
			t.Fatalf("%s source-root option: %v", command, err)
		}
	}
}
