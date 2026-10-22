package gopatterns

import (
	"strings"
	"testing"
)

func TestFreshFieldAssertions(t *testing.T) {
	for _, test := range []struct {
		name, body string
		want       int
	}{
		{"fresh pointer field", `box:=&Box{Value:&Target{}};_=box.Value.(*Target)`, 0},
		{"scalar field", `box:=&Box{Value:"safe"};_=box.Value.(string)`, 0},
		{"typed nil field", `box:=&Box{Value:(*Target)(nil)};_=box.Value.(*Target)`, 0},
		{"returned after assertion", `box:=&Box{Value:&Target{}};_=box.Value.(*Target);return box`, 0},
		{"mutate child only", `box:=&Box{Value:&Target{}};box.Value.(*Target).Count=1;return box`, 0},
		{"wrong target", `box:=&Box{Value:&Other{}};_=box.Value.(*Target)`, 1},
		{"nil field", `box:=&Box{Value:nil};_=box.Value.(*Target)`, 1},
		{"missing field", `box:=&Box{};_=box.Value.(*Target)`, 1},
		{"unknown field", `box:=&Box{Value:input};_=box.Value.(*Target)`, 1},
		{"direct overwrite", `box:=&Box{Value:&Target{}};box.Value=&Other{};_=box.Value.(*Target)`, 1},
		{"conditional overwrite", `box:=&Box{Value:&Target{}};if flag{box.Value=&Other{}};_=box.Value.(*Target)`, 1},
		{"alias write", `box:=&Box{Value:&Target{}};alias:=box;alias.Value=&Other{};_=box.Value.(*Target)`, 1},
		{"argument escape", `box:=&Box{Value:&Target{}};change(box);_=box.Value.(*Target)`, 1},
		{"field address", `box:=&Box{Value:&Target{}};replace(&box.Value);_=box.Value.(*Target)`, 1},
		{"captured write", `box:=&Box{Value:&Target{}};func(){box.Value=&Other{}}();_=box.Value.(*Target)`, 1},
		{"address root", `box:=&Box{Value:&Target{}};replaceBox(&box);_=box.Value.(*Target)`, 1},
		{"method call", `box:=&Box{Value:&Target{}};box.change();_=box.Value.(*Target)`, 1},
		{"reassigned root", `box:=&Box{Value:&Target{}};box=&Box{Value:&Other{}};_=box.Value.(*Target)`, 1},
		{"global publication", `box:=&Box{Value:&Target{}};global=box;_=box.Value.(*Target)`, 1},
		{"parameter field", `_=passed.Value.(*Target)`, 1},
		{"factory field", `box:=newBox(input);_=box.Value.(*Target)`, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			end := ";return nil"
			if strings.Contains(test.body, "return box") {
				end = ""
			}
			extraction, _ := proofExtraction(t, `package example
  type Target struct{Count int};type Other struct{};type Box struct{Value any}
  var global *Box
  func change(box *Box){box.Value=&Other{}}
  func(box *Box)change(){box.Value=&Other{}}
  func replace(value *any){*value=&Other{}}
  func replaceBox(box **Box){*box=&Box{Value:&Other{}}}
  func newBox(value any)*Box{return &Box{Value:value}}
  func check(input any,flag bool,passed *Box)*Box{`+test.body+end+`}`)
			for _, fn := range extraction.Funcs {
				if strings.HasSuffix(fn.Name, ".check") {
					if len(fn.ForceTypeAssertHits) != test.want {
						t.Fatalf("got %+v, want %d warnings", fn.ForceTypeAssertHits, test.want)
					}
					return
				}
			}
			t.Fatal("missing check function")
		})
	}
}
