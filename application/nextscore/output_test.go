package nextscore

import (
	"context"
	"errors"
	"github.com/rstyled/gstyle/domain/patterns"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type failedOutput struct{ err error }

type failedStringOutput struct{ err error }

func (w failedStringOutput) Write(data []byte) (int, error)  { return len(data), nil }
func (w failedStringOutput) WriteString(string) (int, error) { return 0, w.err }

type partialOutput struct{ err error }

func (w partialOutput) Write(data []byte) (int, error) { return min(1, len(data)), w.err }

func (w failedOutput) Write([]byte) (int, error) { return 0, w.err }

func TestCommandsReturnOutputErrors(t *testing.T) {
	for _, content := range []string{"package p\n", "package p;func At(s []int,i int)int{if i<len(s){return s[i]};return 0}"} {
		root := t.TempDir()
		for name, source := range map[string]string{"go.mod": "module example\n\ngo 1.26\n", "p.go": content} {
			if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
		}
		t.Chdir(root)
		for name, run := range map[string]func(context.Context, []string, io.Writer) error{"next": Run, "score": Score} {
			t.Run(name+content, func(t *testing.T) {
				want := errors.New("output unavailable")
				for _, writer := range []io.Writer{failedOutput{want}, failedStringOutput{want}, partialOutput{want}} {
					err := run(context.Background(), []string{"."}, writer)
					if !errors.Is(err, want) {
						t.Fatalf("writer %T: got %v, want output error", writer, err)
					}
				}
			})
		}
	}
}

func TestCommandTextPreservesAllSections(t *testing.T) {
	report := &patterns.PatternsReport{
		Candidates:       []patterns.Candidate{{Kind: patterns.UncheckedOutput, ScoreMilli: 800, Observation: "write dropped", Inference: "return the error", Sites: []patterns.Site{{Path: "p.go", Line: 9, Name: "Report"}}}},
		AnalysisCoverage: []patterns.AnalysisCoverage{{Kind: patterns.TaintFlowKind, Partial: 1}},
		Suppressed:       []patterns.Suppressed{{}},
	}
	text := nextText(report)
	for _, part := range []string{"coverage is incomplete", "Next: unchecked_output (score 0.80)", "write dropped", "return the error", "p.go:9 Report", "no automatic fix"} {
		if !strings.Contains(text, part) {
			t.Fatalf("missing %q: %s", part, text)
		}
	}
	score := scoreText(report)
	for _, part := range []string{"Score: 92/100", "1 active candidates, 1 suppressed", "coverage is incomplete"} {
		if !strings.Contains(score, part) {
			t.Fatalf("missing %q: %s", part, score)
		}
	}
}
