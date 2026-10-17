package callflow

import "testing"

func TestOperationEffectsRejectMissingOrUnknownSchema(t *testing.T) {
	for _, value := range []string{`null`, `{"panik":true}`, `{} {}`, `[]`, `{"panic":1}`, `{"panic":null}`, `{"nil_input":null}`, `{"panic":true,"panic":false}`, `{"nil_input":true,"nil_input":false}`} {
		if _, err := decodeOperationEffect(value); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
	effect, err := decodeOperationEffect(`{"panic":true,"block":true}`)
	if err != nil || !effect.Panic || !effect.Block || effect.Unknown {
		t.Fatalf("effect %+v %v", effect, err)
	}
}
