package gopatterns

import "testing"

func TestAssertionWarningExpression(t *testing.T) {
	for _, test := range []struct {
		name, body string
		want       []string
	}{
		{"simple", `_ = v.(string)`, []string{`v.(string)`}},
		{"nested", `_ = v.(any).(string)`, []string{`v.(any).(string)`, `v.(any)`}},
		{"function literal", `_ = func() any { return v }().(string)`, []string{"func() any {\n\treturn v\n}().(string)"}},
		{"call operand", `_ = wrap(v).(map[string][]int)`, []string{`wrap(v).(map[string][]int)`}},
		{"multiline type", "_ = v.(\nmap[string][]int)", []string{`v.(map[string][]int)`}},
		{"literal whitespace", "_ = map[string]any{\"a  b\": v}[\"a  b\"].(string)", []string{`map[string]any{"a  b": v}["a  b"].(string)`}},
		{"raw literal newline", "_ = map[string]any{`a  b\nc`: v}[`a  b\nc`].(string)", []string{"map[string]any{`a  b\nc`: v}[`a  b\nc`].(string)"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			fn, fs, _ := parseNilErrFunc(t, `package p;func f(v any){`+test.body+`};func wrap(v any)any{return v}`)
			hits := findForceTypeAssertHits(fn, fs)
			if len(hits) != len(test.want) {
				t.Fatalf("got %+v, want expressions %v", hits, test.want)
			}
			for i, want := range test.want {
				if hits[i].ExprText != want {
					t.Errorf("assertion %d: got %q, want %q", i, hits[i].ExprText, want)
				}
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
