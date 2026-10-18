package callflow

import "testing"

func TestCallBoundaryRejectsInvalidMetadata(t *testing.T) {
	for _, value := range []string{`null`, `1`, `""`, `"nil-safe"`, `{}`} {
		var boundary callBoundary
		if boundary.entry(value) == nil {
			t.Fatalf("invalid entry accepted: %s", value)
		}
	}
	for _, value := range []string{`null`, `[0]`, `[1,1]`, `[-1]`, `[1.5]`, `"one"`} {
		var boundary callBoundary
		if boundary.resources(value) == nil {
			t.Fatalf("invalid resource inventory accepted: %s", value)
		}
	}
	var boundary callBoundary
	if err := boundary.entry(`"function"`); err != nil {
		t.Fatal(err)
	}
	if boundary.entry(`"function"`) == nil {
		t.Fatal("duplicate entry accepted")
	}
	if err := boundary.resources(`[]`); err != nil {
		t.Fatal(err)
	}
	if boundary.resources(`[]`) == nil {
		t.Fatal("duplicate empty inventory accepted")
	}
}
