package bindingflow

import "testing"

func TestValueFactBoundary(t *testing.T) {
	for _, raw := range []string{`{"non_nil":false}`, `{"non_nil":null}`, `{"non_nil":"true"}`, `null`, `[]`, `{}`, `{"nil":false}`, `{"nil":null}`, `{"boolean":null}`, `{"boolean":"false"}`, `{"conversion":true}`, `{"conversion":"other"}`, `{"nil":true,"nil":true}`, `{"boolean":true,"boolean":false}`, `{"nil":true} {}`, `{"extra":true}`, `{"nil":{}}`} {
		t.Run(raw, func(t *testing.T) {
			if _, err := decodeValueFact(raw); err == nil {
				t.Fatal("accepted invalid fact")
			}
		})
	}
	for _, raw := range []string{`{"non_nil":true}`, `{"nil":true}`, `{"boolean":false}`, `{"boolean":true}`, `{"conversion":"identity"}`, `{"conversion":"nil"}`, `{"conversion":"boxed"}`, `{"conversion":"nilness"}`, `{"conversion":"unknown"}`} {
		t.Run(raw, func(t *testing.T) {
			if _, err := decodeValueFact(raw); err != nil {
				t.Fatal(err)
			}
		})
	}
}
