package pdg

import "testing"

func TestSourceRevisionKeepsReferencesAndOriginalEvidence(t *testing.T) {
	source := NewSourceContext("/module", []string{"/module/..."})
	original := source.Add("/module/code.go", []byte("before"), []ByteRange{{Start: 0, End: 3}})
	source.Functions = []SourceFunction{{ID: "function", Source: original}}
	source.References = []SourceReference{{Source: original}}
	evidence := source.EvidenceOnly()
	if len(evidence.Functions) != 0 || len(evidence.References) != 0 {
		t.Fatal("disk proof retains old typed inventories")
	}
	next := source.NextRevision(map[string]bool{"/module/code.go": true})
	updated := next.Add("/module/code.go", []byte("after"), nil)
	if updated != original {
		t.Fatal("source snapshot reference changed")
	}
	if err := evidence.Check("/module/code.go", []byte("before")); err != nil {
		t.Fatal("original evidence mutated")
	}
	if err := next.Check("/module/code.go", []byte("after")); err != nil {
		t.Fatal("revision lost updated evidence")
	}
}
