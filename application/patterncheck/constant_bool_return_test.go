package patterncheck

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/adapters/goquality"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/quality"
)

const constantBoolCases = `package example
type Flag bool
const disabled = false
const enabled Flag = true
var calls int
func choose() bool { calls++; return calls % 2 == 0 }
func observe() { calls++ }
func stop() { panic("stop") }
func alwaysTrue() bool { return true }
func alwaysFalse() bool { return false }
type features interface { IsEnabledGlobally(string) bool }
type reader struct { features features }
func enableSqlExpressions(h *reader) bool {
	enabled := !h.features.IsEnabledGlobally("sql")
	if enabled { return false }
	return false
}
func EqualTrue(c bool) bool { if c { return true }; return true }
func ExplicitElse(c bool) bool { if c { return false } else { return false } }
func Aliases(c bool) bool { if c { return (disabled) }; return false }
func NamedBoolean(c bool) Flag { if c { return enabled }; return true }
func Generic[T any](c bool) bool { if c { return false }; return false }
func ConstantExpressions(c bool) bool { if c { return 1 == 1 }; return 2 > 0 }
func Effects() bool { observe(); if choose() { return false }; return false }
func Init(c bool) bool { if observe(); c { return false }; return false }
func ClosurePrelude(c bool) bool { _ = func() bool { return true }; if c { return false }; return false }
func Fixed(c bool) bool { if c { return false }; return true }
func Stub() bool { return false }
func OneReturn(c bool) bool { if c { observe() }; return false }
func Variable(c bool) bool { value := false; if c { return value }; return false }
func CallResult(c bool) bool { if c { return choose() }; return false }
func Shadowed(c, false bool) bool { if c { return false }; return false }
func NonBoolean(c bool) int { if c { return 1 }; return 1 }
func GenericResult[T ~bool](c bool) T { if c { return false }; return false }
func Multiple(c bool) (bool, error) { if c { return false, nil }; return false, nil }
func Named(c bool) (result bool) { if c { return false }; return false }
func Naked(c bool) (result bool) { if c { return }; return }
func Deferred(c bool) bool { defer observe(); if c { return false }; return false }
func Recovered(c bool) (result bool) { defer func() { result = true; recover() }(); if c { return false }; return false }
func ArmEffects(c bool) bool { if c { observe(); return false }; return false }
func PrefixBranch(c bool) bool { if c { observe() }; if choose() { return false }; return false }
func ConstantCondition() bool { if true { return false }; return false }
func DeadShortCircuit(c bool) bool { if c && false { return false }; return false }
func DeadTail(c bool) bool { return true; if c { return false }; return false }
func DeadPanic(c bool) bool { panic("stop"); if c { return false }; return false }
func DeadHelper(c bool) bool { stop(); if c { return false }; return false }
func OneLiveArmTrue() bool { if alwaysTrue() { return false }; return false }
func OneLiveArmFalse() bool { if alwaysFalse() { return false }; return false }
func PanicArm(c bool) bool { if c { panic("stop") }; return false }
func Infinite(c bool) bool { for {}; if c { return false }; return false }
func LoopArm(c bool) bool { if c { for {} }; return false }
func Switch(c bool) bool { switch c { case true: return false; default: return false } }
func Goto(c bool) bool { goto tail; tail: if c { return false }; return false }
func Concurrent(c bool) bool { go observe(); if c { return false }; return false }
func ClosureResult(c bool) bool { if c { return func() bool { return false }() }; return false }
func DeadExtraReturn(c bool) bool { if c { return false }; return false; return false }
func ElseWithTail(c bool) bool { if c { return false } else { return false }; return false }
func BlockTail(c bool) bool { if c { return false }; { return false } }
// gstyle-allow(constant_bool_return): intentionally disabled by policy
func Suppressed(c bool) bool { if c { return false }; return false }
//lint:ignore constant_bool_return intentionally disabled by policy
func LintSuppressed(c bool) bool { if c { return false }; return false }
`

func TestConstantBoolReturnBehavior(t *testing.T) {
	directory := writeConstantBoolModule(t)
	t.Chdir(directory)
	report, err := GetReport(context.Background(), []string{"sample.go"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"enableSqlExpressions": "false", "EqualTrue": "true", "ExplicitElse": "false",
		"Aliases": "false", "NamedBoolean": "true", "Generic": "false",
		"ConstantExpressions": "true", "Effects": "false", "Init": "false", "ClosurePrelude": "false",
	}
	for _, candidate := range report.Candidates {
		if candidate.Kind != patterns.ConstantBoolReturn {
			continue
		}
		name := strings.TrimPrefix(candidate.Sites[0].Name, "example.")
		value, ok := want[name]
		if !ok {
			t.Errorf("unexpected finding for %s: %+v", name, candidate)
			continue
		}
		delete(want, name)
		if !strings.HasSuffix(candidate.Observation, "return "+value) || candidate.FixSpec != nil {
			t.Errorf("incorrect value or automatic fix for %s: %+v", name, candidate)
		}
		if name == "enableSqlExpressions" {
			line := strings.Count(constantBoolCases[:strings.Index(constantBoolCases, "\tif enabled")], "\n") + 1
			site := candidate.Sites[0]
			if site.Path != "sample.go" || site.Line != line || site.EndLine != line+1 {
				t.Errorf("incorrect source location: %+v", site)
			}
		}
	}
	if len(want) != 0 {
		t.Errorf("missing findings: %v", want)
	}
	assertConstantBoolSuppression(t)
}

func TestConstantBoolReturnSARIF(t *testing.T) {
	t.Chdir(writeConstantBoolModule(t))
	var output bytes.Buffer
	if err := Run(context.Background(), []string{"--format", "sarif", "sample.go"}, &output); err != nil {
		t.Fatal(err)
	}
	var document struct {
		Runs []struct {
			Results []struct {
				RuleID string `json:"ruleId"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(output.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, result := range document.Runs[0].Results {
		if result.RuleID == "constant_bool_return" {
			count++
		}
	}
	if count != 10 {
		t.Fatalf("want 10 SARIF warnings, got %d", count)
	}
}

func writeConstantBoolModule(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	for name, content := range map[string]string{"go.mod": "module example\n\ngo 1.26\n", "sample.go": constantBoolCases} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return directory
}

func assertConstantBoolSuppression(t *testing.T) {
	t.Helper()
	facts, err := (goquality.Analyzer{}).Extract(context.Background(), []string{"sample.go"})
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.RuleID == "invalid_suppression" {
			t.Errorf("registered pattern suppression is invalid: %+v", diagnostic)
		}
	}
}
