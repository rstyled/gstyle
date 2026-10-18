package bindingflow

import "testing"

func TestCallShapeDecoderRejectsIncompleteOrAmbiguousRecords(t *testing.T) {
	for _, raw := range []string{
		`null`, `[]`, `{}`, `{"fixed":0,"variadic":false,"spread":false}`,
		`{"fixed":0,"variadic":false,"spread":false,"tuple":0,"fixed":0}`,
		`{"fixed":0,"variadic":false,"spread":false,"tuple":0,"other":0}`,
		`{"fixed":null,"variadic":false,"spread":false,"tuple":0}`,
		`{"fixed":-1,"variadic":false,"spread":false,"tuple":0}`,
		`{"fixed":1.0,"variadic":false,"spread":false,"tuple":0}`,
		`{"fixed":4294967296,"variadic":false,"spread":false,"tuple":0}`,
		`{"fixed":0,"variadic":"true","spread":false,"tuple":0}`,
		`{"fixed":0,"variadic":false,"spread":null,"tuple":0}`,
		`{"fixed":0,"variadic":false,"spread":true,"tuple":0}`,
		`{"fixed":0,"variadic":true,"spread":true,"tuple":2}`,
		`{"fixed":1,"variadic":false,"spread":false,"tuple":1}`,
		`{"fixed":0,"variadic":false,"spread":false,"tuple":0} {}`,
	} {
		t.Run(raw, func(t *testing.T) {
			if _, err := decodeCallShape(raw); err == nil {
				t.Fatal("accepted invalid call shape")
			}
		})
	}
	for _, raw := range []string{
		`{"fixed":0,"variadic":false,"spread":false,"tuple":0}`,
		`{"fixed":2,"variadic":false,"spread":false,"tuple":2}`,
		`{"fixed":1,"variadic":true,"spread":false,"tuple":0}`,
		`{"fixed":0,"variadic":true,"spread":true,"tuple":0}`,
	} {
		if _, err := decodeCallShape(raw); err != nil {
			t.Fatal(err)
		}
	}
}
func TestTupleArityRequiresOneJSONInteger(t *testing.T) {
	for _, raw := range []string{`null`, `"2"`, `02`, `2.0`, `2 3`, `-2`, `4294967296`, `true`, `[]`, `{}`} {
		if _, err := decodeTupleArity(raw); err == nil {
			t.Fatalf("accepted arity %s", raw)
		}
	}
	if n, err := decodeTupleArity("2"); err != nil || n != 2 {
		t.Fatalf("arity=%d %v", n, err)
	}
}
