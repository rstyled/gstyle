package bindingflow

import (
	"go/constant"
	"testing"
)

func TestBooleanScalarValues(t *testing.T) {
	for _, tc := range []struct {
		name         string
		scalar       constant.Value
		value, known bool
	}{
		{"nil", nil, false, false},
		{"true", constant.MakeBool(true), true, true},
		{"false", constant.MakeBool(false), false, true},
		{"integer", constant.MakeInt64(1), false, false},
		{"string", constant.MakeString("true"), false, false},
		{"unknown", constant.MakeUnknown(), false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, known := (Value{Scalar: tc.scalar}).Bool()
			if value != tc.value || known != tc.known {
				t.Fatalf("got (%t,%t), want (%t,%t)", value, known, tc.value, tc.known)
			}
		})
	}
}
