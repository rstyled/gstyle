package changerisk

import (
	"reflect"
	"testing"
)

func TestModuleRecordsKeepReplacementsAndDiscardCheckoutPaths(t *testing.T) {
	a := []byte(`{"Path":"example.com/module","Version":"v1.0.0","GoVersion":"1.26","Dir":"/temporary/base","Replace":{"Path":"../replacement","Dir":"/temporary/base/replacement"}}`)
	b := []byte(`{"Path":"example.com/module","Version":"v1.0.0","GoVersion":"1.26","Dir":"/temporary/head","Replace":{"Path":"../replacement","Dir":"/temporary/head/replacement"}}`)
	left, err := moduleRecords(a)
	if err != nil {
		t.Fatal(err)
	}
	right, err := moduleRecords(b)
	if err != nil || !reflect.DeepEqual(left, right) {
		t.Fatalf("checkout paths changed dependencies: %v %v %v", left, right, err)
	}
	changed, err := moduleRecords([]byte(`{"Path":"example.com/module","Version":"v1.0.0","GoVersion":"1.26","Replace":{"Path":"../different"}}`))
	if err != nil || reflect.DeepEqual(left, changed) {
		t.Fatal("replacement change disappeared")
	}
	if _, err := moduleRecords([]byte(`{"Path":`)); err == nil {
		t.Fatal("malformed dependency output accepted")
	}
	for _, input := range []string{"", `{}`, `{"Path":"module","Replace":{}}`} {
		if _, err := moduleRecords([]byte(input)); err == nil {
			t.Errorf("incomplete dependency output accepted: %s", input)
		}
	}
}
