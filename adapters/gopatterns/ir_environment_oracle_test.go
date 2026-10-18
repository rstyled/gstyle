package gopatterns

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/callflow"
	"strings"
	"testing"
)

func TestEscapedCellsAgainstTwoObjectTraceOracle(t *testing.T) {
	var source strings.Builder
	source.WriteString(`package example;
 func factory(flag bool)(func()bool,func(bool)){return func()bool{return flag},func(v bool){flag=v}};
 func identity(cb func()bool)func()bool{return cb};func store(cb func(bool),v bool){cb(v)};
 func execute(a,b,d0,v0,d1,v1,wantA,wantB bool){readA,setA:=factory(a);readB,setB:=factory(b);aliasA:=identity(readA);if d0{store(setA,v0)}else{store(setB,v0)};if d1{store(setA,v1)}else{store(setB,v1)};if aliasA()!=wantA||readB()!=wantB{panic("mismatch")}};`)
	want := map[string]callflow.Outcomes{}
	for code := 0; code < 256; code++ {
		values := make([]bool, 8)
		args := make([]string, 8)
		for bit := range values {
			values[bit] = code&(1<<bit) != 0
			args[bit] = fmt.Sprint(values[bit])
		}
		a, b := values[0], values[1]
		for _, offset := range []int{2, 4} {
			if values[offset] {
				a = values[offset+1]
			} else {
				b = values[offset+1]
			}
		}
		outcome := callflow.Returns
		if a != values[6] || b != values[7] {
			outcome = callflow.Panics
		}
		name := fmt.Sprintf("f%d", code)
		fmt.Fprintf(&source, "func %s(){execute(%s)};", name, strings.Join(args, ","))
		want["example."+name] = outcome
	}
	checkCallableSourceOracle(t, source.String(), want)
	if len(want) != 256 {
		t.Fatal("incomplete escaped-cell trace oracle")
	}
}
