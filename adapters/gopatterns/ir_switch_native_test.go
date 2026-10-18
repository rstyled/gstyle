package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"testing"
)

func TestNativeSyntheticSwitchEffectEvidence(t *testing.T) {
	verifyNativeValueCases(t, []unwindValueCase{
		{"uncomparable interface case panics", `value:=any([]int{});switch value{case value:}`, "", callflow.Panics},
		{"nil interface comparison returns", `value:=any([]int{});switch value{case nil:panic("bad")}`, "", callflow.Returns},
		{"type decision returns", `value:=any([]int{});switch value.(type){case []int:default:panic("bad")}`, "", callflow.Returns},
	})
}
