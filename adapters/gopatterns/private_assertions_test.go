package gopatterns

import (
	"strings"
	"testing"
)

func TestPrivateHelperAssertions(t *testing.T) {
	for _, test := range []struct {
		name, source string
		want         int
	}{
		{"guarded caller", `func leaf(v any){_=v.(string)};func caller(v any){if _,ok:=v.(string);ok{leaf(v)}}`, 0},
		{"concrete caller", `func leaf(v any){_=v.(string)};func caller(){leaf("safe")}`, 0},
		{"returned guard", `func leaf(v any){_=v.(string)};func caller(v any){if _,ok:=v.(string);!ok{return};leaf(v)}`, 0},
		{"type case caller", `func leaf(v any){_=v.(string)};func caller(v any){switch v.(type){case string:leaf(v)}}`, 0},
		{"alias parameter", `func leaf(v any){copy:=v;_=copy.(string)};func caller(){leaf("safe")}`, 0},
		{"unguarded caller", `func leaf(v any){_=v.(string)};func caller(v any){leaf(v)}`, 1},
		{"mixed callers", `func leaf(v any){_=v.(string)};func good(){leaf("safe")};func bad(v any){leaf(v)}`, 1},
		{"wrong caller type", `func leaf(v any){_=v.(string)};func caller(){leaf(7)}`, 1},
		{"nil caller", `func leaf(v any){_=v.(string)};func caller(){leaf(nil)}`, 1},
		{"function escape", `func leaf(v any){_=v.(string)};var escaped=leaf;func caller(){leaf("safe")}`, 1},
		{"no caller", `func leaf(v any){_=v.(string)}`, 1},
		{"exported function", `func Leaf(v any){_=v.(string)};func caller(){Leaf("safe")}`, 1},
		{"method", `type box struct{};func(box)leaf(v any){_=v.(string)};func caller(){box{}.leaf("safe")}`, 1},
		{"overwrite parameter", `func leaf(v any){v=7;_=v.(string)};func caller(){leaf("safe")}`, 1},
		{"conditional overwrite", `func leaf(v any,flag bool){if flag{v=7};_=v.(string)};func caller(flag bool){leaf("safe",flag)}`, 1},
		{"captured parameter", `func leaf(v any){func(){v=7}();_=v.(string)};func caller(){leaf("safe")}`, 1},
		{"escaped parameter", `func leaf(v any){replace(&v);_=v.(string)};func caller(){leaf("safe")};func replace(p *any){*p=7}`, 1},
		{"wrong guard", `func leaf(v any){_=v.(string)};func caller(v,other any){if _,ok:=other.(string);ok{leaf(v)}}`, 1},
		{"stale guard", `func leaf(v any){_=v.(string)};func caller(v any){if _,ok:=v.(string);!ok{return};v=7;leaf(v)}`, 1},
		{"guard bypass", `func leaf(v any){_=v.(string)};func caller(v any,flag bool){if flag{goto use};if _,ok:=v.(string);!ok{return};use:leaf(v)}`, 1},
		{"recursion", `func leaf(v any){_=v.(string);leaf(v)};func caller(){leaf("safe")}`, 1},
		{"generic", `func leaf[T any](v any){_=v.(string)};func caller(){leaf[int]("safe")}`, 1},
		{"closure caller", `func leaf(v any){_=v.(string)};func caller(){func(){leaf("safe")}()}`, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, "package example;"+test.source)
			seen := false
			for _, fn := range extraction.Funcs {
				if !strings.HasSuffix(strings.ToLower(fn.Name), ".leaf") {
					continue
				}
				seen = true
				if len(fn.ForceTypeAssertHits) != test.want {
					t.Fatalf("got %+v, want %d warnings", fn.ForceTypeAssertHits, test.want)
				}
			}
			if !seen {
				t.Fatal("missing helper function")
			}
		})
	}
}
