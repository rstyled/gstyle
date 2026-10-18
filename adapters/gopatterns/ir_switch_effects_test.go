package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"testing"
)

func TestSyntheticSwitchComparisonsHaveTypedEffects(t *testing.T) {
	cases := []struct {
		name, source string
		want         callflow.Outcomes
	}{
		{"integer", `package example;func f(v int){switch v{case 0:return;case 1:return};return}`, callflow.Returns},
		{"interface comparison", `package example;func f(x,y any){switch x{case y:return};return}`, callflow.Returns | callflow.Panics},
		{"nil interface comparison", `package example;func f(x any){switch x{case nil:return};return}`, callflow.Returns},
		{"struct with interface", `package example;type S struct{value any};func f(x,y S){switch x{case y:return};return}`, callflow.Returns | callflow.Panics},
		{"type decision", `package example;func f(x any){switch x.(type){case int:return;case string:return};return}`, callflow.Returns},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			checkCallableSourceOracle(t, tc.source, map[string]callflow.Outcomes{"example.f": tc.want})
		})
	}
}
