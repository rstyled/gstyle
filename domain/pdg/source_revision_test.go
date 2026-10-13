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

func TestSourceSyntaxFollowsRevisionLifetime(t *testing.T) {
	source := NewSourceContext("/module", nil)
	ref := source.Add("/module/code.go", []byte("before"), nil)
	if err := source.SetSyntax(ref, "profile", "compact"); err != nil {
		t.Fatal(err)
	}
	evidence, _ := source.EvidenceOnly().Snapshot(ref)
	if evidence.Syntax != "" {
		t.Fatal("disk proof retains obsolete syntax")
	}
	unchanged, _ := source.NextRevision(nil).Snapshot(ref)
	if unchanged.Syntax != "compact" {
		t.Fatal("unchanged revision loses syntax")
	}
	updated := source.NextRevision(map[string]bool{"/module/code.go": true})
	updated.Add("/module/code.go", []byte("after"), nil)
	snapshot, _ := updated.Snapshot(ref)
	if snapshot.Syntax != "" {
		t.Fatal("changed revision retains stale syntax")
	}
	if err := updated.SetSyntax(ref, "profile", "new compact"); err != nil {
		t.Fatal(err)
	}
	original, _ := source.Snapshot(ref)
	if original.Syntax != "compact" {
		t.Fatal("new revision mutates original syntax")
	}
}
