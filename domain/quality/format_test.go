package quality

import "testing"

func TestPrivateScalarScanStorage(t *testing.T) {
	local := Storage{Provenance: Local}
	external := Storage{Provenance: External}
	unknown := Storage{Provenance: Unknown}
	for _, test := range []struct {
		name   string
		callee string
		memory bool
		args   []Storage
		want   Kind
	}{
		{"sscan", "fmt.Sscan", true, []Storage{external, local}, None},
		{"sscanln", "fmt.Sscanln", true, []Storage{external, local}, None},
		{"sscanf", "fmt.Sscanf", true, []Storage{external, external, local, local}, None},
		{"shared", "fmt.Sscanf", true, []Storage{local, local, external}, MutationEffect},
		{"mixed", "fmt.Sscan", true, []Storage{local, local, external}, MutationEffect},
		{"unknown", "fmt.Sscan", true, []Storage{local, unknown}, MutationEffect},
		{"legacy", "fmt.Sscan", true, nil, MutationEffect},
		{"no destinations", "fmt.Sscanf", true, []Storage{local, local}, MutationEffect},
		{"no scalar proof", "fmt.Sscan", false, []Storage{local, local}, IO},
		{"memory writer", "fmt.Fprint", true, []Storage{local, local}, MutationEffect},
		{"JSON scalar", "encoding/json.Unmarshal", true, []Storage{external, local}, MutationEffect},
		{"JSON shared scalar", "encoding/json.Unmarshal", true, []Storage{external, external}, MutationEffect},
		{"JSON unknown scalar", "encoding/json.Unmarshal", true, []Storage{external, unknown}, MutationEffect},
		{"JSON legacy scalar", "encoding/json.Unmarshal", true, nil, MutationEffect},
		{"JSON unproved type", "encoding/json.Unmarshal", false, []Storage{external, local}, UnknownEffect},
	} {
		t.Run(test.name, func(t *testing.T) {
			call := Call{Callee: test.callee, Line: 2, MemoryWriter: test.memory, StorageArgs: test.args, Args: make([]ArgSource, len(test.args))}
			kind, configured := call.policyKind(DefaultConfig().Prefixes)
			if !configured || kind != test.want {
				t.Fatalf("got %s/%t, want %s: %+v", kind, configured, test.want, call)
			}
			for _, override := range []Kind{IO, MutationEffect, UnknownEffect} {
				prefixes := append(DefaultConfig().Prefixes, Prefix{Path: test.callee, Kind: override})
				if kind, _ := call.policyKind(prefixes); kind != override {
					t.Errorf("private scan hides explicit %s policy: %s", override, kind)
				}
			}
			call.Args = append(call.Args, ArgSource{Kind: ArgOther})
			if kind, _ := call.policyKind(DefaultConfig().Prefixes); kind == None {
				t.Error("partial storage facts hide an unproved destination")
			}
			call.Args = nil
			if kind, _ := call.policyKind(DefaultConfig().Prefixes); kind == None {
				t.Error("absent argument facts prove private destinations")
			}
		})
	}
}

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
		{Callee: "encoding/json.Unmarshal", Line: 2, Dynamic: true, MemoryWriter: true},
	} {
		f := fact("caller", 1)
		f.Calls = []Call{call}
		if _, err := Evaluate([]Function{f}, DefaultConfig()); err == nil {
			t.Fatalf("invalid writer evidence accepted: %+v", call)
		}
	}
}
