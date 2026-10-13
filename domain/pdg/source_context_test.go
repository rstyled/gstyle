package pdg

import (
	"crypto/sha256"
	"encoding/json"
	"strings"
	"testing"
)

func TestSourceContextOwnsOneSnapshot(t *testing.T) {
	source := []byte("// doc\npackage p\n")
	comments := []ByteRange{{0, 6}}
	context := NewSourceContext("/repo", []string{"."})
	ref := context.Add("/repo/p.go", source, comments)
	if context.Add("/repo/p.go", source, comments) != ref {
		t.Fatal("duplicate source gets another reference")
	}
	source[0] = 'x'
	comments[0].End = 1
	snapshot, ok := context.Snapshot(ref)
	if !ok || snapshot.Identity != sha256.Sum256([]byte(snapshot.Content)) {
		t.Fatal("snapshot does not own original bytes")
	}
	snapshot.Comments[0].End = 2
	again, _ := context.Snapshot(ref)
	if again.Comments[0].End != 6 {
		t.Fatal("snapshot comments can be changed by consumer")
	}
	if context.Check("/repo/p.go", source) == nil {
		t.Fatal("stale source accepted")
	}
	if context.Check("missing", source) == nil {
		t.Fatal("unknown source accepted")
	}
	if _, ok := context.Snapshot(0); ok {
		t.Fatal("absent reference accepted")
	}
}

func TestSourceEditsPreserveUntouchedComments(t *testing.T) {
	source := SourceSnapshot{Path: "p.go", Content: "// doc\nx + y // tail\n", Comments: []ByteRange{{0, 6}, {13, 20}}}
	out, err := ApplySourceEdits(source, []SourceEdit{{ByteRange{7, 8}, "first"}, {ByteRange{11, 12}, "second"}})
	if err != nil {
		t.Fatal(err)
	}
	if out != "// doc\nfirst + second // tail\n" {
		t.Fatalf("wrong edit result: %q", out)
	}
	if !source.HasComment(ByteRange{0, 2}) || source.HasComment(ByteRange{7, 12}) {
		t.Fatal("wrong comment intersection")
	}
}

func TestSourceEditsRejectInvalidRanges(t *testing.T) {
	source := SourceSnapshot{Content: "abcdef"}
	cases := [][]SourceEdit{
		{{Range: ByteRange{2, 7}}},
		{{Range: ByteRange{3, 2}}},
		{{Range: ByteRange{0, 3}}, {Range: ByteRange{2, 4}}},
		{{Range: ByteRange{2, 2}}, {Range: ByteRange{2, 2}}},
	}
	for _, edits := range cases {
		if _, err := ApplySourceEdits(source, edits); err == nil {
			t.Fatal("invalid edits accepted")
		}
	}
}

func TestSourceContextDoesNotExpandGraphJSON(t *testing.T) {
	context := NewSourceContext("source-context-marker", []string{"."})
	ref := context.Add("p.go", []byte("package p"), nil)
	value := struct {
		Tables *Tables
		Source Source
	}{Tables: &Tables{Sources: context}, Source: Source{Snapshot: ref}}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "source-context-marker") || strings.Contains(string(data), "Snapshot") {
		t.Fatal("runtime source context leaks into graph JSON")
	}
}
