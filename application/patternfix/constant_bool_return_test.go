package patternfix

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/application/nextscore"
)

func TestConstantBoolReturnRemainsAdvisory(t *testing.T) {
	source := "package example\nfunc predicate(enabled bool) bool { if enabled { return false }; return false }\n"
	path := writeTempGo(t, source)
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "go.mod"), []byte("module example\n\ngo 1.26\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Dir(path))
	var output bytes.Buffer
	if err := nextscore.Run(context.Background(), []string{path}, &output); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"constant_bool_return", "no automatic fix"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("next output lacks %q: %s", want, output.String())
		}
	}
	output.Reset()
	if err := nextscore.Score(context.Background(), []string{path}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Score: 92/100") {
		t.Errorf("score omits constant return warning: %s", output.String())
	}
	output.Reset()
	if err := Run(context.Background(), []string{"--apply", path}, &output); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != source {
		t.Fatalf("advisory finding rewrites source: %s", actual)
	}
}
