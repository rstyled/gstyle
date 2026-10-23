package main

import (
	"fmt"
	"strconv"
	"strings"
)

func certificate(cases []observation, selected []suite, source string) string {
	var out strings.Builder
	out.WriteString("import Gstyle.Proof.Programs.Contract\nopen Perennial.GstylePrograms\nset_option maxRecDepth 100000\nset_option maxHeartbeats 0\nnamespace GeneratedPrograms\n")
	for _, c := range cases {
		writeCase(&out, c)
	}
	for index, s := range selected {
		writeCoverage(&out, cases, s, index)
	}
	writeSource(&out, cases, source)
	out.WriteString("end GeneratedPrograms\n")
	return out.String()
}

func writeCase(out *strings.Builder, c observation) {
	name := c.Name
	fmt.Fprintf(out, "def p%s : Program := %s\n", name, c.Program.lean())
	fmt.Fprintf(out, "def r%s : Reported := ⟨(%d), (%d), %t, %t⟩\n", name, c.Range.Lower, c.Range.Upper, c.Range.HasLower, c.Range.HasUpper)
	fmt.Fprintf(out, "theorem sound%s : Sound p%s r%s := by\n", name, name, name)
	fmt.Fprintf(out, "  intro x hx\n  simp only [p%s, r%s, Program.eval, Expression.eval, Reported.contains, Comparison.holds]\n", name, name)
	out.WriteString("  all_goals simp_all [Signed64]\n  all_goals repeat' (split <;> simp_all [Signed64])\n  all_goals try omega\n")
	fmt.Fprintf(out, "theorem precision%s : PreciseLiteral p%s r%s := by\n", name, name, name)
	out.WriteString("  intro n h\n  cases h\n  all_goals decide\n")
	fmt.Fprintf(out, "theorem source%s : p%s.source %s = %s := by rfl\n", name, name, strconv.Quote(name), strconv.Quote(c.Source))
	fmt.Fprintf(out, "def c%s : Certificate := ⟨%s, p%s, r%s, %s, source%s, by decide, sound%s, precision%s⟩\n", name, strconv.Quote(name), name, name, strconv.Quote(c.Source), name, name, name)
}

func writeCoverage(out *strings.Builder, cases []observation, s suite, index int) {
	var names []string
	for _, c := range cases {
		if c.Suite == s.Name {
			names = append(names, "c"+c.Name)
		}
	}
	fmt.Fprintf(out, "def certificates%d : List Certificate := [%s]\n", index, strings.Join(names, ", "))
	fmt.Fprintf(out, "theorem coverage%d : certificates%d.map Certificate.program = %s := by rfl\n", index, index, s.leanPrograms())
	fmt.Fprintf(out, "theorem complete%d (p : Program) (hp : Admitted %s p) :\n", index, strings.TrimPrefix(s.leanPrograms(), "enumerate "))
	out.WriteString("    ∃ c ∈ certificates" + strconv.Itoa(index) + ", c.program = p ∧ Sound p c.report ∧ PreciseLiteral p c.report := by\n")
	fmt.Fprintf(out, "  exact certified_admitted certificates%d _ _ _ coverage%d p hp\n", index, index)
	fmt.Fprintf(out, "#print axioms GeneratedPrograms.complete%d\n", index)
}

func writeSource(out *strings.Builder, cases []observation, source string) {
	var names []string
	for _, c := range cases {
		names = append(names, "c"+c.Name)
	}
	fmt.Fprintf(out, "theorem source_file : \"package generated\\nfunc observe(value int64) {}\\n\" ++ String.join ([%s].map Certificate.source) = %s := by rfl\n", strings.Join(names, ", "), strconv.Quote(source))
	out.WriteString("#print axioms GeneratedPrograms.source_file\n")
}
