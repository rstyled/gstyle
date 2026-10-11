package graphmatch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shanejonas/cyclo/domain/ccgraph"
)

func inputPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "graphs.json")
	data := `{"profile":"compat.ccgraph.strict/0.1.0","input_scope":"projected_stage_inputs","functions":[
	{"Key":"b","Name":"same","References":0,"Inputs":[],"Outputs":[{"Role":"return","Type":"unit","Variadic":false}],"Graph":{"Nodes":[{"ID":"n","Label":"OTHER"}],"Edges":[]}},
	{"Key":"a","Name":"same","References":0,"Inputs":[],"Outputs":[{"Role":"return","Type":"unit","Variadic":false}],"Graph":{"Nodes":[{"ID":"x","Label":"OTHER"}],"Edges":[]}}
	]}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCLIEmitsPairAndCompleteSummary(t *testing.T) {
	var output bytes.Buffer
	if err := Run(context.Background(), []string{"--input", inputPath(t)}, &output); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&output)
	var pair, summary record
	if err := decoder.Decode(&pair); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(&summary); err != nil {
		t.Fatal(err)
	}
	if pair.Type != "pair" || pair.Pair.Left != "a" || !pair.Pair.Accepted {
		t.Fatalf("pair=%+v", pair)
	}
	if summary.Type != "summary" || !summary.Run.Complete || summary.Run.Pairs != 1 || summary.Run.InputScope != "projected_stage_inputs" {
		t.Fatalf("summary=%+v", summary)
	}
	if err := endOfJSON(decoder); err != nil {
		t.Fatal(err)
	}
}

func TestCLIFailures(t *testing.T) {
	for _, args := range [][]string{nil, {"--input", "missing"}, {"--unknown"}, {"--input", inputPath(t), "extra"}} {
		if err := Run(context.Background(), args, &bytes.Buffer{}); err == nil {
			t.Fatal(args)
		}
	}
	path := inputPath(t)
	for _, data := range []string{`{"profile":"legacy","input_scope":"projected_stage_inputs"}`, `{"unknown":true}`, `{} {}`, `{"profile":"compat.ccgraph.strict/0.1.0","input_scope":"native_pdg"}`} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if err := Run(context.Background(), []string{"--input", path}, &bytes.Buffer{}); err == nil {
			t.Fatal(data)
		}
	}
}

func TestMissingVariadicCannotBecomeKnownFalse(t *testing.T) {
	path := inputPath(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.ReplaceAll(data, []byte(`,"Variadic":false`), nil)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), []string{"--input", path}, &bytes.Buffer{}); err == nil {
		t.Fatal("missing variadic became known false")
	}
}

func TestCLICancelledRunDisclosesIncomplete(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var output bytes.Buffer
	err := Run(ctx, []string{"--input", inputPath(t)}, &output)
	if !errors.Is(err, context.Canceled) || !strings.Contains(output.String(), `"Complete":false`) {
		t.Fatalf("%s %v", output.String(), err)
	}
}

func TestCLIResourceDeadline(t *testing.T) {
	var output bytes.Buffer
	err := Run(context.Background(), []string{"--input", inputPath(t), "--timeout", "1ns"}, &output)
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(output.String(), `"Complete":false`) {
		t.Fatalf("%s %v", output.String(), err)
	}
	if err := Run(context.Background(), []string{"--input", inputPath(t), "--timeout", "-1s"}, &bytes.Buffer{}); err == nil {
		t.Fatal("negative timeout accepted")
	}
}

func TestCLIConfigChangesDecisionWithoutRounding(t *testing.T) {
	path := inputPath(t)
	var input inputFile
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	input.Functions[0].Name = "different"
	data, err = json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(config, []byte(`{"string_threshold":"1"}`), 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := Run(context.Background(), []string{"--input", path, "--config", config}, &output); err != nil {
		t.Fatal(err)
	}
	var first record
	if err := json.NewDecoder(&output).Decode(&first); err != nil {
		t.Fatal(err)
	}
	if first.Pair.Accepted || first.Pair.Strings != ccgraph.Rejected || first.Pair.Final != ccgraph.NotEvaluated {
		t.Fatal(first.Pair)
	}
}
