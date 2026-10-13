// Package graphmatch exposes imported projected-stage and validated native-source
// matching. Native PDG JSON claims alone are not source validation proof.
package graphmatch

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rstyled/gstyle/domain/ccgraph"
)

type inputFile struct {
	Profile    string                  `json:"profile"`
	InputScope string                  `json:"input_scope"`
	Functions  []ccgraph.StageFunction `json:"functions"`
}
type record struct {
	Type  string        `json:"type"`
	Pair  *ccgraph.Pair `json:"pair,omitempty"`
	Run   *ccgraph.Run  `json:"run,omitempty"`
	Error string        `json:"error,omitempty"`
}

// Run writes JSON Lines pair traces followed by a complete/incomplete summary.
func Run(ctx context.Context, args []string, output io.Writer) error {
	fs := flag.NewFlagSet("match-graphs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("input", "", "imported projected-stage input JSON")
	source := fs.String("source", "", "Go source root for independently validated native matching")
	contextPath := fs.String("context-output", "", "optional paired PDG/CFG conformance JSON Lines output")
	configPath := fs.String("config", "", "strict matching overrides JSON")
	timeout := fs.Duration("timeout", 0, "explicit resource deadline; zero is unlimited")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := inputOptions(*path, *source, *contextPath, fs.NArg()); err != nil {
		return err
	}
	return runDeadline(ctx, options{input: *path, config: *configPath, timeout: *timeout, source: *source, contextOutput: *contextPath}, output)
}

type options struct {
	input, config         string
	timeout               time.Duration
	source, contextOutput string
}

func runDeadline(ctx context.Context, opts options, output io.Writer) error {
	timeout := opts.timeout
	if timeout < 0 {
		return fmt.Errorf("timeout must be nonnegative")
	}
	if timeout == 0 {
		return runInput(ctx, opts, output)
	}
	limited, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return runInput(limited, opts, output)
}

func inputOptions(input, source, contextPath string, args int) error {
	if args != 0 || (input == "") == (source == "") {
		return fmt.Errorf("usage: gstyle match-graphs --input graphs.json | --source ROOT [--config thresholds.json] [--context-output context.jsonl]")
	}
	if contextPath != "" && source == "" {
		return fmt.Errorf("context-output requires native source input")
	}
	return nil
}

func runInput(ctx context.Context, opts options, output io.Writer) error {
	if opts.source != "" {
		return runNative(ctx, opts, output)
	}
	return runFiles(ctx, opts.input, opts.config, output)
}

func runFiles(ctx context.Context, path, configPath string, output io.Writer) error {
	input, err := loadInput(path)
	if err != nil {
		return err
	}
	config, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	run, matchErr := ccgraph.Evaluate(ctx, input.Functions, config, func(pair ccgraph.Pair) error {
		return encoder.Encode(record{Type: "pair", Pair: &pair})
	})
	return finish(encoder, run, matchErr)
}

func loadInput(path string) (inputFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return inputFile{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var input inputFile
	if err := decoder.Decode(&input); err != nil {
		return inputFile{}, err
	}
	if err := endOfJSON(decoder); err != nil {
		return inputFile{}, err
	}
	if input.Profile != ccgraph.Profile || input.InputScope != "projected_stage_inputs" {
		return inputFile{}, fmt.Errorf("require %s projected_stage_inputs; native PDG JSON claims are not validation proof, use --source", ccgraph.Profile)
	}
	return input, nil
}

func endOfJSON(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("trailing JSON value")
}

func loadConfig(path string) (ccgraph.Config, error) {
	if path == "" {
		return ccgraph.DefaultConfig(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ccgraph.Config{}, err
	}
	return ccgraph.ParseConfig(data)
}

func finish(encoder *json.Encoder, run ccgraph.Run, matchErr error) error {
	final := record{Type: "summary", Run: &run}
	if matchErr != nil {
		final.Error = matchErr.Error()
	}
	if err := encoder.Encode(final); err != nil {
		return err
	}
	return matchErr
}
