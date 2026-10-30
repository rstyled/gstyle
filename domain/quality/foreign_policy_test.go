package quality

import "testing"

func TestForeignPolicyDoesNotOverrideSourceBody(t *testing.T) {
	config := DefaultConfig()
	config.Prefixes = []Prefix{{Path: "cpp::printf:", Kind: IO, ForeignOnly: true}}
	f := fact("cpp::caller:void()", 1)
	f.Helpers = []Helper{{Name: "cpp::printf:void()"}}
	f.Calls = []Call{{Callee: "cpp::printf:void()", Line: 1, Local: true}}
	result := evaluate(t, f, config).Functions[0]
	if !result.Complete || len(result.Effects) != 0 {
		t.Fatalf("foreign API policy overrides a source body: %+v", result)
	}
	f.Calls[0].Local = false
	result = evaluate(t, f, config).Functions[0]
	if !result.Complete || len(result.Effects) != 1 || result.Effects[0].Kind != IO {
		t.Fatalf("foreign API policy is ignored: %+v", result)
	}
}
