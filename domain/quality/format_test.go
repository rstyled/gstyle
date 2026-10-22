package quality

import "testing"

func TestMemoryWriterEvidenceRejectsOtherCalls(t *testing.T) {
	for _, call := range []Call{
		{Callee: "fmt.Fprint", Line: 2, Dynamic: true, MemoryWriter: true},
		{Callee: "fmt.Sprintf", Line: 2, MemoryWriter: true},
		{Callee: "other.Fprint", Line: 2, MemoryWriter: true},
		{Callee: "fmt.Sscan", Line: 2, Dynamic: true, MemoryWriter: true},
		{Callee: "fmt.Fscan", Line: 2, MemoryWriter: true},
		{Callee: "io.WriteString", Line: 2, Dynamic: true, MemoryWriter: true},
		{Callee: "encoding/xml.Decoder.Decode", Line: 2, Dynamic: true, MemoryWriter: true},
		{Callee: "encoding/xml.Unmarshal", Line: 2, MemoryWriter: true},
		{Callee: "encoding/json.Decoder.Decode", Line: 2, Dynamic: true, MemoryWriter: true},
		{Callee: "encoding/json.Unmarshal", Line: 2, MemoryWriter: true},
	} {
		f := fact("caller", 1)
		f.Calls = []Call{call}
		if _, err := Evaluate([]Function{f}, DefaultConfig()); err == nil {
			t.Fatalf("invalid writer evidence accepted: %+v", call)
		}
	}
}
