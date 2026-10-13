package graphmatch

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/domain/ccgraph"
)

func TestNativeCLIReportsVerifiedPairsAndContext(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{"go.mod": "module example.com/native\n\ngo 1.26\n", "example.go": "package native;func First(x int)int{return x};func Second(y int)int{return y}"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(t.TempDir(), "context.jsonl")
	var output bytes.Buffer
	if err := Run(context.Background(), []string{"--source", root, "--context-output", path}, &output); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&output)
	counts := map[string]int{}
	var summary ccgraph.Run
	for {
		var row struct {
			Type string       `json:"type"`
			Run  *ccgraph.Run `json:"run"`
		}
		err := decoder.Decode(&row)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		counts[row.Type]++
		if row.Run != nil {
			summary = *row.Run
		}
	}
	if counts["native-manifest"] != 1 || counts["pair"] != 1 || counts["summary"] != 1 {
		t.Fatal(counts)
	}
	if !summary.Complete || summary.InputScope != "validated_native_source" || summary.Pairs != 1 {
		t.Fatal(summary)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), []string{"--source", root, "--context-output", path}, io.Discard); err == nil {
		t.Fatal("existing context output is overwritten")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("existing output changes", err)
	}
}

func TestNativeCLIRejectsMixedInputModes(t *testing.T) {
	if err := Run(context.Background(), []string{"--source", ".", "--input", inputPath(t)}, io.Discard); err == nil {
		t.Fatal("mixed modes are accepted")
	}
	if err := Run(context.Background(), []string{"--input", inputPath(t), "--context-output", "unused"}, io.Discard); err == nil {
		t.Fatal("imported inputs request native context")
	}
}
