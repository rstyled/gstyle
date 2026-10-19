package bindingflow

import (
	"go/constant"
	"go/token"
	"strings"
	"testing"
)

func concatenatedScalar() constant.Value {
	value := constant.MakeString("")
	piece := constant.MakeString(strings.Repeat("x", 1024))
	for range 1024 {
		value = constant.BinaryOp(value, token.ADD, piece)
	}
	return value
}

func TestBooleanIgnoresLargeConcatenatedString(t *testing.T) {
	value := Value{Scalar: concatenatedScalar()}
	for range 2 {
		if flag, known := value.Bool(); flag || known {
			t.Fatalf("string becomes Boolean: %t %t", flag, known)
		}
	}
}

func BenchmarkBooleanScalar(b *testing.B) {
	for name, scalar := range map[string]constant.Value{"bool": constant.MakeBool(true), "string": constant.MakeString("text"), "flattened_1MiB": concatenatedScalar()} {
		// The first conversion flattens a lazy string once, outside the steady-state measurement.
		value := Value{Scalar: scalar}
		value.Bool()
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				value.Bool()
			}
		})
	}
	b.Run("fresh_1MiB", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			b.StopTimer()
			value := Value{Scalar: concatenatedScalar()}
			b.StartTimer()
			value.Bool()
		}
	})
}
