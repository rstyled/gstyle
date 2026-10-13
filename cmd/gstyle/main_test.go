package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestImportedGraphCommandDispatch(t *testing.T) {
	var output bytes.Buffer
	err := run([]string{"match-graphs", "--input", "../../domain/ccgraph/testdata/imported-graphs.json"}, &output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"type":"pair"`) || !strings.Contains(output.String(), `"Complete":true`) {
		t.Fatal(output.String())
	}
	if code := commandExitCode([]string{"match-graphs"}, os.ErrNotExist); code != 2 {
		t.Fatalf("hard failure exit code=%d", code)
	}
}

func TestSkillFlagPrintsAnAgentSkillWithoutStartingTheTUI(t *testing.T) {
	var output bytes.Buffer
	err := run([]string{"--skill"}, &output)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(output.String(), "---\nname: gstyle\n") {
		t.Fatalf("skill output has no gstyle frontmatter:\n%s", output.String())
	}
	if strings.Contains(output.String(), "\x1b") {
		t.Fatalf("skill output contains TUI escape sequences: %q", output.String())
	}
}

func TestBinaryLaunchesGstyle(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("XDG_STATE_HOME", directory)
	source := filepath.Join(directory, "sample.go")
	err := os.WriteFile(source, []byte("package sample\n\nfunc Simple() {}\n"), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	// Cold CI builds need their own deadline; compilation is not TUI startup.
	buildContext, cancelBuild := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelBuild()
	binary := filepath.Join(t.TempDir(), "gstyle")
	build := exec.CommandContext(buildContext, "go", "build", "-buildvcs=false", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build gstyle: %v\n%s", err, output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, binary, "--control-port", "0", source)
	command.Stdin = strings.NewReader("q")
	command.Env = append(os.Environ(), "TERM=xterm-256color")
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("gstyle did not exit after q: %v\n%s", ctx.Err(), output)
	}
	if err != nil {
		t.Fatalf("gstyle failed: %v\n%s", err, output)
	}
}

func TestParseControlPortAndPaths(t *testing.T) {
	options, err := parseRunOptions([]string{"--control-port", "9000", "one.go", "two.go"})
	if err != nil {
		t.Fatal(err)
	}
	if options.controlPort != 9000 || strings.Join(options.paths, ",") != "one.go,two.go" {
		t.Fatalf("options = %+v", options)
	}
}

func TestDefaultControlPort(t *testing.T) {
	options, err := parseRunOptions(nil)
	if err != nil {
		t.Fatal(err)
	}
	if options.controlPort != 8197 {
		t.Fatalf("control port = %d, want 8197", options.controlPort)
	}
}

func TestQualityCheckHelpDoesNotStartTUI(t *testing.T) {
	var output bytes.Buffer
	if err := run([]string{"check", "--help"}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Usage: gstyle check") {
		t.Fatal(output.String())
	}
}

func TestRejectInvalidControlPort(t *testing.T) {
	_, err := parseRunOptions([]string{"--control-port", "70000"})
	if err == nil {
		t.Fatal("invalid control port succeeded")
	}
}

func TestBugReducerHelpDoesNotStartTUI(t *testing.T) {
	var output bytes.Buffer
	err := run([]string{"bug-reducer", "--help"}, &output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Minimize an input") {
		t.Fatalf("help = %q", output.String())
	}
}
