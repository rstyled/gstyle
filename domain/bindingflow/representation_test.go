package bindingflow

import (
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestOriginRepresentationFollowsCopies(t *testing.T) {
	q := originCycle(t)
	q.representations[2][1] = BoxedRepresentation
	got := q.OriginsOf(4, Facts{})
	if !got.Known || len(got.Sources) != 1 || got.Sources[0].Representation != BoxedRepresentation {
		t.Fatalf("boxing lost across cyclic copies: %+v", got)
	}
	delete(q.representations, 2)
	got = q.OriginsOf(4, Facts{})
	if !got.Known || len(got.Sources) != 1 || got.Sources[0].Representation != UnknownRepresentation {
		t.Fatalf("missing representation assumes identity: %+v", got)
	}
}

func TestRepresentationInventoryRejectsInvalidEvidence(t *testing.T) {
	for _, data := range []string{"null", "{", `[{"position":0,"kind":"identity"}]`, `[{"position":1,"kind":"invalid"}]`, `[{"position":1,"kind":"identity"},{"position":1,"kind":"boxed"}]`} {
		t.Run(data, func(t *testing.T) {
			q := valueQuery(t, "copy")
			ext := pdg.Extension{Value: testText(q, data)}
			if err := q.representationFact(1, ext); err == nil {
				t.Fatal("invalid inventory accepted")
			}
		})
	}
	q := valueQuery(t, "copy")
	ext := pdg.Extension{Value: testText(q, `[{"position":1,"kind":"identity"}]`)}
	if err := q.representationFact(1, ext); err != nil {
		t.Fatal(err)
	}
	if err := q.representationFact(1, ext); err == nil {
		t.Fatal("duplicate inventory accepted")
	}
}
