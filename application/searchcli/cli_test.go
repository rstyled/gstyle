package searchcli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
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
	err := runCommand(context.Background(), []string{"--query", "SAVE", "--rule", "side_effect_density", dir}, &output)
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
	if err := runCommand(context.Background(), []string{"--format", "text", "--query", "Calm", dir}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Calm CC=1 COG=0") {
		t.Fatalf("text = %s", &output)
	}
}

func TestRunReturnsPartialResultsAndErrorForTypeFailure(t *testing.T) {
	dir := searchModule(t, "package sample\nfunc Broken() { missing() }\n")
	var output bytes.Buffer
	err := runCommand(context.Background(), []string{dir}, &output)
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
		if err := runCommand(context.Background(), args, &bytes.Buffer{}); err == nil {
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

type shortWriter struct{}

func (shortWriter) Write(data []byte) (int, error) { return len(data) - 1, nil }

func TestSearchRejectsShortWrites(t *testing.T) {
	for _, format := range []string{"json", "text"} {
		err := writeResult(shortWriter{}, application.SearchResult{Matches: []application.SearchMatch{}}, format)
		if !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("%s error = %v, want short write", format, err)
		}
	}
}

func TestSearchEncodesCompletePages(t *testing.T) {
	next := 2
	result := application.SearchResult{Total: 3, Offset: 1, Limit: 1, NextOffset: &next,
		Matches: []application.SearchMatch{{Path: "sample.go", Name: "Save", Line: 4, Column: 1, Complexity: 2,
			Findings: []application.SearchFinding{{RuleID: "fn_length", Message: "long body"}},
		}},
	}
	var text bytes.Buffer
	if err := writeResult(&text, result, "text"); err != nil {
		t.Fatal(err)
	}
	want := "sample.go:4:1 Save CC=2 COG=0\n  fn_length: long body\n3 matching functions; offset 1; returned 1\nnext offset: 2\n"
	if text.String() != want {
		t.Fatalf("text = %q, want %q", text.String(), want)
	}
	var output bytes.Buffer
	if err := writeResult(&output, result, "json"); err != nil {
		t.Fatal(err)
	}
	var decoded application.SearchResult
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, result) || !bytes.HasSuffix(output.Bytes(), []byte("\n")) {
		t.Fatalf("JSON page = %s", &output)
	}
}

// Match the CLI boundary: parse the flags used for help, then execute once.
func runCommand(ctx context.Context, args []string, output io.Writer) error {
	flags, execute := NewCommand()
	if err := flags.Parse(args); err != nil {
		return err
	}
	return execute(ctx, args, output)
}

func TestCommandExecutesTheParsedFlagSetOnce(t *testing.T) {
	dir := searchModule(t, "package sample\nfunc Save() {}\nfunc Calm() {}\n")
	flags, execute := NewCommand()
	parses := 0
	flags.Func("probe", "count parses", func(string) error {
		parses++
		return nil
	})
	args := []string{"--probe", "value", "--query", "Calm", dir}
	if err := flags.Parse(args); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := execute(context.Background(), args, &output); err != nil {
		t.Fatal(err)
	}
	var result application.SearchResult
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if parses != 1 || result.Total != 1 || result.Matches[0].Name != "Calm" {
		t.Fatalf("parses=%d result=%s", parses, &output)
	}
}

func TestCommandRequiresParsedFlags(t *testing.T) {
	_, execute := NewCommand()
	if err := execute(context.Background(), nil, &bytes.Buffer{}); err == nil {
		t.Fatal("accepted an unparsed flag set")
	}
}

func TestCommandChecksCancellationBeforeLoadingConfig(t *testing.T) {
	flags, execute := NewCommand()
	if err := flags.Parse([]string{"--config", "missing.toml"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := execute(ctx, nil, &bytes.Buffer{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want cancellation before config IO", err)
	}
}
