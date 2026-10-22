package quality

import "testing"

func TestLocalMemoryReadKeepsPolicyAndValidation(t *testing.T) {
	f := fact("caller", 1)
	f.Calls = []Call{
		{Callee: "io.ReadAll", Line: 2, LocalMemoryRead: true},
		{Callee: "io.Closer.Close", Line: 3, Dynamic: true, LocalMemoryRead: true},
		{Callee: "encoding/json.Decoder.Token", Line: 4, LocalMemoryRead: true},
		{Callee: "encoding/json.Decoder.More", Line: 5, LocalMemoryRead: true},
	}
	got := evaluate(t, f, DefaultConfig()).Functions[0]
	if len(got.Effects) != 0 || !got.Complete {
		t.Fatalf("closed memory reads: %+v", got)
	}
	config := DefaultConfig()
	config.Prefixes = append(config.Prefixes,
		Prefix{Path: "io.ReadAll", Kind: IO}, Prefix{Path: "io.Closer.Close", Kind: IO},
		Prefix{Path: "encoding/json.Decoder.Token", Kind: IO}, Prefix{Path: "encoding/json.Decoder.More", Kind: IO})
	if got := evaluate(t, f, config).Functions[0]; len(got.Effects) != 4 {
		t.Fatalf("reader proof hid override: %+v", got)
	}
	for _, call := range []Call{
		{Callee: "other", Line: 2, LocalMemoryRead: true},
		{Callee: "io.ReadAll", Line: 2, Dynamic: true, LocalMemoryRead: true},
		{Callee: "encoding/json.Decoder.Token", Line: 2, Dynamic: true, LocalMemoryRead: true},
		{Callee: "encoding/json.Decoder.More", Line: 2, Dynamic: true, LocalMemoryRead: true},
	} {
		f.Calls = []Call{call}
		if _, err := Evaluate([]Function{f}, DefaultConfig()); err == nil {
			t.Fatalf("invalid memory proof accepted: %+v", call)
		}
	}
}
