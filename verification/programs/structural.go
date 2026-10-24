package main

import (
	"fmt"
	"strconv"
	"strings"
)

func structuralCertificate(cases []observation, selected []suite, source string) string {
	var out strings.Builder
	out.WriteString("import Gstyle.Proof.Programs.Structural\nopen Perennial.GstylePrograms\nset_option maxRecDepth 100000\nnamespace GeneratedPrograms\n")
	for _, c := range cases {
		writeStructuralCase(&out, c)
	}
	for index, s := range selected {
		writeCoverage(&out, cases, s, index)
	}
	writeSource(&out, cases, source)
	out.WriteString("end GeneratedPrograms\n")
	return out.String()
}

func writeStructuralCase(out *strings.Builder, c observation) {
	name := c.Name
	fmt.Fprintf(out, "def p%s : Program := %s\n", name, c.Program.lean())
	fmt.Fprintf(out, "def r%s : Reported := ⟨(%d), (%d), %t, %t⟩\n", name, c.Range.Lower, c.Range.Upper, c.Range.HasLower, c.Range.HasUpper)
	fmt.Fprintf(out, "theorem checked%s : checkReport p%s r%s = true := by decide\n", name, name, name)
	fmt.Fprintf(out, "theorem source%s : p%s.source %s = %s := by rfl\n", name, name, strconv.Quote(name), strconv.Quote(c.Source))
	fmt.Fprintf(out, "def c%s : Certificate := certificateOfCheck %s p%s r%s %s source%s (by decide) checked%s\n", name, strconv.Quote(name), name, name, strconv.Quote(c.Source), name, name)
	fmt.Fprintf(out, "#print axioms GeneratedPrograms.checked%s\n#print axioms GeneratedPrograms.source%s\n", name, name)
}
