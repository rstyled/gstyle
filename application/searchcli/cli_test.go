package searchcli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/application"
)

func searchModule(t *testing.T, source string) string {
	t.Helper()
	dir := t.TempDir()
	for name, text := range map[string]string{"go.mod": "module example\n\ngo 1.26\n", "sample.go": source} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestRunSearchesSourceAndFindings(t *testing.T) {
	dir := searchModule(t, "package sample\nimport \"fmt\"\nfunc Save() { fmt.Println(\"saved\"); fmt.Println(\"again\"); fmt.Println(\"done\") }\nfunc Calm() {}\n")
	var output bytes.Buffer
	err := Run(context.Background(), []string{"--query", "SAVE", "--rule", "side_effect_density", dir}, &output)
	if err != nil {
		t.Fatal(err)
	}
	var result application.SearchResult
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || result.Matches[0].Name != "Save" || result.Quality.Status != "ready" {
		t.Fatalf("result = %s", &output)
	}
	output.Reset()
	if err := Run(context.Background(), []string{"--format", "text", "--query", "Calm", dir}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Calm CC=1 COG=0") {
		t.Fatalf("text = %s", &output)
	}
}

func TestRunReturnsPartialResultsAndErrorForTypeFailure(t *testing.T) {
	dir := searchModule(t, "package sample\nfunc Broken() { missing() }\n")
	var output bytes.Buffer
	err := Run(context.Background(), []string{dir}, &output)
	if err == nil || !strings.Contains(err.Error(), "quality analysis failed") {
		t.Fatalf("error = %v", err)
	}
	var result application.SearchResult
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || result.Quality.Status != "error" {
		t.Fatalf("partial result = %s", &output)
	}
}

func TestRunRejectsBadOptionsBeforeAnalysis(t *testing.T) {
	for _, args := range [][]string{{"--limit", "0"}, {"--offset", "-1"}, {"--format", "xml"}, {"--min-complexity", "-1"}, {"--config", "missing.toml"}} {
		if err := Run(context.Background(), args, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, errors.New("closed") }

func TestSearchPropagatesOutputFailure(t *testing.T) {
	for _, format := range []string{"json", "text"} {
		if err := writeResult(failedWriter{}, application.SearchResult{}, format); err == nil {
			t.Fatal("lost writer error")
		}
	}
}
