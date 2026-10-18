package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func closureEnvironmentCases() []unwindValueCase {
	return []unwindValueCase{
		{"returned false environment", `read:=factory(false);if read(){panic("bad")}`, `func factory(flag bool)func()bool{return func()bool{return flag}}`, callflow.Returns},
		{"returned true environment", `read:=factory(true);if !read(){panic("bad")}`, `func factory(flag bool)func()bool{return func()bool{return flag}}`, callflow.Returns},
		{"shared reader writer cell", `read,write:=factory(false);write(true);if !read(){panic("bad")}`, `func factory(flag bool)(func()bool,func(bool)){return func()bool{return flag},func(value bool){flag=value}}`, callflow.Returns},
		{"distinct factory activations", `read,write:=factory(false);other,_:=factory(false);write(true);if !read()||other(){panic("bad")}`, `func factory(flag bool)(func()bool,func(bool)){return func()bool{return flag},func(value bool){flag=value}}`, callflow.Returns},
		{"live store before factory return", `read:=factory();if !read(){panic("bad")}`, `func factory()func()bool{flag:=false;read:=func()bool{return flag};flag=true;return read}`, callflow.Returns},
		{"deferred store before factory return", `read:=factory();if !read(){panic("bad")}`, `func factory()func()bool{flag:=false;defer func(){flag=true}();return func()bool{return flag}}`, callflow.Returns},
		{"captured callable environment", `read:=factory(no);if read(){panic("bad")}`, `func no()bool{return false};func factory(cb func()bool)func()bool{return func()bool{return cb()}}`, callflow.Returns},
		{"captured callable changes before return", `read:=factory();if read(){panic("bad")}`, `func no()bool{return false};func yes()bool{return true};func factory()func()bool{cb:=yes;read:=func()bool{return cb()};cb=no;return read}`, callflow.Returns},
		{"forwarded aliases keep identity", `read,write:=factory(false);again:=identity(read);write(true);if !again(){panic("bad")}`, `func identity(cb func()bool)func()bool{return cb};func factory(flag bool)(func()bool,func(bool)){return func()bool{return flag},func(v bool){flag=v}}`, callflow.Returns},
		{"same code distinct environments terminate", `first:=factory(stop);second:=factory(first);second()`, `func stop(){panic("end")};func factory(cb func())func(){return func(){cb()}}`, callflow.Panics},
		{"same code target alternatives retain both environments", `read:=pick(factory(false),factory(true),choose);if read(){panic("possible")}`, `func pick(a,b func()bool,v bool)func()bool{if v{return a};return b};func factory(flag bool)func()bool{return func()bool{return flag}}`, callflow.Returns | callflow.Panics},
		{"callback writes escaped cell", `read,write:=factory(false);apply(write,true);if !read(){panic("bad")}`, `func apply(cb func(bool),v bool){cb(v)};func factory(flag bool)(func()bool,func(bool)){return func()bool{return flag},func(v bool){flag=v}}`, callflow.Returns},
		{"callable cell updates", `read,write:=factory(yes);write(no);if read(){panic("bad")}`, `func no()bool{return false};func yes()bool{return true};func factory(cb func()bool)(func()bool,func(func()bool)){return func()bool{return cb()},func(v func()bool){cb=v}}`, callflow.Returns},
		{"captured assignment exports new environment", `var read func()bool;set:=func(){flag:=false;read=func()bool{return flag};flag=true};set();if !read(){panic("bad")}`, "", callflow.Returns},
		{"relay environment outlives both owners", `make:=outer(false);read,write:=make();write(true);if !read(){panic("bad")}`, `func outer(flag bool)func()(func()bool,func(bool)){return func()(func()bool,func(bool)){return func()bool{return flag},func(v bool){flag=v}}}`, callflow.Returns},
		{"panic prefix publishes live cell", `read,write:=factory(false);defer func(){recover();if !read(){panic("bad")}}();write()`, `func factory(flag bool)(func()bool,func()){return func()bool{return flag},func(){flag=true;panic("caught")}}`, callflow.Returns},
		{"repeated allocation site uses weak updates", `again:=true;var first,second func()bool;var set func(bool);for{read,write:=factory(false);if again{first=read;set=write;again=false;continue};second=read;break};set(true);if !first()||second(){panic("possible")}`, `func factory(flag bool)(func()bool,func(bool)){return func()bool{return flag},func(v bool){flag=v}}`, callflow.Returns | callflow.Panics},
		{"alternative writers retain untouched cells", `read,write:=factory(false);_,other:=factory(false);write=pick(write,other,choose);write(true);if read(){panic("possible")}`, `func pick(a,b func(bool),v bool)func(bool){if v{return a};return b};func factory(flag bool)(func()bool,func(bool)){return func()bool{return flag},func(v bool){flag=v}}`, callflow.Returns | callflow.Panics},
		{"writer or noop retains untouched cell", `read,write:=factory(false);write=pick(write,noop,choose);write(true);if read(){panic("possible")}`, `func noop(bool){};func pick(a,b func(bool),v bool)func(bool){if v{return a};return b};func factory(flag bool)(func()bool,func(bool)){return func()bool{return flag},func(v bool){flag=v}}`, callflow.Returns | callflow.Panics},
		{"alternative factories retain branch allocations", `make:=pick(left,right,choose);read:=make();if read(){panic("bad")}`, `func pick(a,b func()func()bool,v bool)func()func()bool{if v{return a};return b};func left()func()bool{flag:=false;return func()bool{return flag}};func right()func()bool{flag:=false;return func()bool{return flag}}`, callflow.Returns},
		{"immediate writer shares callback argument cell", `flag:=false;read:=func()bool{return flag};func(cb func()bool){flag=true;_ = cb()}(read);if !read(){panic("bad")}`, "", callflow.Returns},
		{"guarded captured callable keeps identity", `read:=factory(no);if read(){panic("bad")}`, `func no()bool{return false};func factory(cb func()bool)func()bool{return func()bool{if cb!=nil{return cb()};return false}}`, callflow.Returns},
	}
}
func TestClosureEnvironmentsCrossFactoryReturns(t *testing.T) {
	for _, tc := range closureEnvironmentCases() {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, `package example;var choose bool;func f(){`+tc.body+`};`+tc.helper)
			graphs := map[string]*pdg.Graph{}
			for _, graph := range extraction.Callables {
				graphs[graph.Tables.Text(graph.Function.ID)] = graph
			}
			summaries, err := callflow.ExitEffects(graphs)
			if err != nil {
				t.Fatal(err)
			}
			if got := summaries["example.f"].Ordinary; got != tc.want {
				t.Fatalf("exit=%v want %v", got, tc.want)
			}
		})
	}
}
func TestNativeClosureEnvironments(t *testing.T) {
	verifyNativeValueCases(t, closureEnvironmentCases())
}
