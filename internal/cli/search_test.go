package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/application"
)

func TestSearchDispatchUsesTheParsedOptions(t *testing.T) {
	dir := t.TempDir()
	for name, source := range map[string]string{
		"go.mod":    "module sample\n\ngo 1.26\n",
		"sample.go": "package sample\nfunc Save() {}\nfunc Calm() {}\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"Save", "Calm"} {
		var output bytes.Buffer
		if err := run([]string{"search", "--query", name, "--limit", "1", dir}, &output); err != nil {
			t.Fatal(err)
		}
		var result application.SearchResult
		if err := json.Unmarshal(output.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Total != 1 || len(result.Matches) != 1 || result.Matches[0].Name != name {
			t.Fatalf("query=%s result=%s", name, &output)
		}
	}
}

func TestSearchOperationalFailureExitCode(t *testing.T) {
	if got := commandExitCode([]string{"search"}, os.ErrInvalid); got != 2 {
		t.Fatalf("search failure exit code = %d, want 2", got)
	}
}
