package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"testing"
)

func TestExitArgumentsUseVariadicAndTupleInputs(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         callflow.Outcomes
	}{
		{"empty pack is nil", `func f(){stop()};func stop(values ...int){if values!=nil{panic("bad")}}`, callflow.Returns},
		{"nonempty pack is nonnil", `func f(){stop(0)};func stop(values ...int){if values==nil{panic("bad")}}`, callflow.Returns},
		{"nil element does not make nil pack", `func f(){stop(nil)};func stop(values ...*int){if values==nil{panic("bad")}}`, callflow.Returns},
		{"nil spread stays nil", `func f(){var values []int;stop(values...)};func stop(values ...int){if values!=nil{panic("bad")}}`, callflow.Returns},
		{"fixed prefix with empty pack", `func f(){stop(false)};func stop(flag bool,values ...int){if flag||values!=nil{panic("bad")}}`, callflow.Returns},
		{"fixed prefix with nonempty pack", `func f(){stop(false,0)};func stop(flag bool,values ...int){if flag||values==nil{panic("bad")}}`, callflow.Returns},
		{"empty literal spread is nonnil", `func f(){values:=[]int{};stop(values...)};func stop(values ...int){if values==nil{panic("bad")}}`, callflow.Returns},
		{"saved nonnil empty spread", `func f(){values:=[]int{};defer stop(values...);values=nil;_=values};func stop(values ...int){if values==nil{panic("bad")}}`, callflow.Returns},
		{"unknown spread stays unknown", `func f(values []int){stop(values...)};func stop(values ...int){if values==nil{panic("bad")}}`, callflow.Returns | callflow.Panics},
		{"pack-state recursion terminates", `func f(){walk(0)};func walk(values ...int){if values==nil{return};walk()}`, callflow.Returns},
		{"nonempty pack recursion diverges", `func f(){walk(0)};func walk(values ...int){if values!=nil{walk(0)}}`, callflow.Diverges},
		{"tuple boxes both results", `func f(){stop(pair())};func pair()(*int,*int){return nil,nil};func stop(first,second any){if first==nil||second==nil{panic("bad")}}`, callflow.Returns},
		{"tuple values keep their slot types", `func f(){stop(pair())};func pair()(any,*int){return nil,nil};func stop(first,second any){if second==nil{panic("bad")};_=first}`, callflow.Returns},
		{"tuple position-specific boxing", `func f(){stop(pair())};func pair()(*int,*int){return nil,nil};func stop(first *int,second any){if second==nil{panic("bad")};_=first}`, callflow.Returns},
		{"tuple nil interface result is known", `func f(){stop(pair())};func pair()(any,*int){return nil,nil};func stop(first,second any){if first==nil{panic("unknown")};_=second}`, callflow.Panics},
		{"tuple fills variadic pack", `func f(){stop(pair())};func pair()(int,int){return 0,0};func stop(values ...int){if values==nil{panic("bad")}}`, callflow.Returns},
		{"tuple exactly fills fixed prefix", `func f(){stop(pair())};func pair()(*int,*int){return nil,nil};func stop(first,second any,values ...int){if first==nil||second==nil||values!=nil{panic("bad")}}`, callflow.Returns},
		{"saved nil spread", `func f(){var values []int;defer stop(values...);values=[]int{0};_=values};func stop(values ...int){if values!=nil{panic("bad")}}`, callflow.Returns},
		{"saved tuple pack", `func f(){defer stop(pair())};func pair()(int,int){return 0,0};func stop(values ...int){if values==nil{panic("bad")}}`, callflow.Returns},
		{"tuple fault happens before receiving call", `func f(){stop(pair())};func pair()(*int,*int){panic("fault")};func stop(first,second any){}`, callflow.Panics},
		{"constructor value preserves fault outcomes", `import "os";func f(){stop(make([]int,0)...)};func stop(values ...int){if values==nil{os.Exit(1)}}`, callflow.Returns | callflow.Panics},
		{"saved nonempty pack", `func f(){defer stop(0)};func stop(values ...int){if values==nil{panic("bad")}}`, callflow.Returns},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graphs, name := argumentExitGraphs(t, tc.source)
			summaries, err := callflow.ExitEffects(graphs)
			if err != nil {
				t.Fatal(err)
			}
			if got := summaries[name].Ordinary; got != tc.want {
				t.Fatalf("exit=%v want %v", got, tc.want)
			}
		})
	}
}

func nativePackShape(values ...*int) (bool, int) { return values == nil, len(values) }
func nativePackPair() (*int, *int)               { return nil, nil }
func TestNativePackShapeUsesArityAndSpread(t *testing.T) {
	var empty []*int
	if nilValue, count := nativePackShape(); !nilValue || count != 0 {
		t.Fatal("zero arguments must produce nil slice")
	}
	if nilValue, count := nativePackShape(nil); nilValue || count != 1 {
		t.Fatal("nil element must remain in nonnil pack")
	}
	if nilValue, count := nativePackShape(empty...); !nilValue || count != 0 {
		t.Fatal("spread preserves nil slice")
	}
	if nilValue, count := nativePackShape([]*int{}...); nilValue || count != 0 {
		t.Fatal("spread preserves nonnil empty slice")
	}
	if nilValue, count := nativePackShape(nativePackPair()); nilValue || count != 2 {
		t.Fatal("tuple results form two packed arguments")
	}
}
