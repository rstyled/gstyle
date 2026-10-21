package gopatterns

import "testing"

func TestAssertionWarningExpression(t *testing.T) {
	for _, test := range []struct{ name, body, want string }{
		{"simple", `_ = v.(string)`, `v.(string)`},
		{"nested", `_ = v.(any).(string)`, `v.(any).(string)`},
		{"function literal", `_ = func() any { return v }().(string)`, "func() any {\n\treturn v\n}().(string)"},
		{"call operand", `_ = wrap(v).(map[string][]int)`, `wrap(v).(map[string][]int)`},
		{"multiline type", "_ = v.(\nmap[string][]int)", `v.(map[string][]int)`},
		{"literal whitespace", "_ = map[string]any{\"a  b\": v}[\"a  b\"].(string)", `map[string]any{"a  b": v}["a  b"].(string)`},
		{"raw literal newline", "_ = map[string]any{`a  b\nc`: v}[`a  b\nc`].(string)", "map[string]any{`a  b\nc`: v}[`a  b\nc`].(string)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fn, fs, info := parseNilErrFunc(t, `package p;func f(v any){`+test.body+`};func wrap(v any)any{return v}`)
			hits := findForceTypeAssertHits(fn, fs, info)
			if len(hits) != 1 || hits[0].ExprText != test.want {
				t.Fatalf("got %+v, want one assertion with expression %q", hits, test.want)
			}
		})
	}
}

func TestNilErrorWarningExpression(t *testing.T) {
	for _, test := range []struct{ name, condition, want string }{
		{"simple", `err != nil`, `err != nil`},
		{"reversed", `nil != err`, `nil != err`},
		{"parenthesized multiline", "(err !=\nnil)", `(err != nil)`},
	} {
		t.Run(test.name, func(t *testing.T) {
			fn, fs, info := parseNilErrFunc(t, `package p;func f(err error)error{if `+test.condition+`{return nil};return err}`)
			hits := findNilErrHits(fn, fs, info)
			if len(hits) != 1 || hits[0].CondText != test.want {
				t.Fatalf("got %+v, want one nil-error finding with condition %q", hits, test.want)
			}
		})
	}
}
