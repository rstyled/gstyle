package gopatterns

import (
	"go/types"
	"strconv"
	"testing"
)

func TestSignedStepWidthFacts(t *testing.T) {
	graphs := runtimeTestGraphs(t, `package p; type Index int64; func F(s []int, a int64, b int, c int32, d uint64, e Index) { a++; a--; b++; c++; d--; e--; _ = s[a] }`)
	seen := 0
	for _, function := range graphs {
		g := &function.Pdg
		for _, node := range g.Nodes {
			attrs := g.NodeAttributes(node)
			form := g.Tables.Text(attrs.WriteForm)
			if form != "++" && form != "--" {
				continue
			}
			if len(attrs.Writes) != 1 {
				t.Fatal("step has no unique write")
			}
			name := g.Tables.Text(g.Symbols[attrs.Writes[0].Symbol-1].Name.Value)
			want := ""
			if name == "a" || name == "e" || name == "b" {
				want = strconv.Quote(form)
			}
			got := ""
			for _, extension := range attrs.Extensions {
				if g.Tables.Text(extension.Name) == "gstyle.runtime:signed-step64" {
					got = g.Tables.Text(extension.Value)
				}
			}
			if got != want {
				t.Fatalf("name=%s form=%s got=%q want=%q", name, form, got, want)
			}
			seen++
		}
	}
	if seen != 6 {
		t.Fatalf("checked %d steps, want 6", seen)
	}
}

func TestSignedStepNegativeIndexIntegration(t *testing.T) {
	cases := []struct {
		name, source string
		warning      bool
	}{
		{"bounded increment retains lower", `func At(s []int, i int64) int { if i >= 0 && i < 7 { i++; if i < int64(len(s)) { return s[i] } }; return 0 }`, false},
		{"unbounded increment can wrap", `func At(s []int, i int64) int { if i >= 0 { i++; if i < int64(len(s)) { return s[i] } }; return 0 }`, true},
		{"decrement crosses zero", `func At(s []int, i int64) int { if i >= 0 && i < 7 { i--; if i < int64(len(s)) { return s[i] } }; return 0 }`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := negativeIndexTestCandidates(t, "package p; "+tc.source)
			if (len(got) > 0) != tc.warning {
				t.Fatalf("findings=%+v want warning=%v", got, tc.warning)
			}
		})
	}
}

func TestSignedStepTargetSizes(t *testing.T) {
	cases := []struct {
		name  string
		kind  types.BasicKind
		sizes types.Sizes
		want  bool
	}{
		{"native 32", types.Int, types.SizesFor("gc", "386"), false},
		{"native 64", types.Int, types.SizesFor("gc", "amd64"), true},
		{"native unknown", types.Int, nil, false},
		{"int64 without target", types.Int64, nil, true},
		{"int32 on 64", types.Int32, types.SizesFor("gc", "amd64"), false},
		{"unsigned 64", types.Uint64, types.SizesFor("gc", "amd64"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			builder := irBuilder{sizes: tc.sizes}
			if got := builder.signedStep64Type(types.Typ[tc.kind]); got != tc.want {
				t.Fatalf("got=%v want=%v", got, tc.want)
			}
		})
	}
}
