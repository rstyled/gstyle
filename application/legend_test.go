package application

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestSourceLegendNamesBackgroundsAndTextSeparately(t *testing.T) {
	legend := strings.Join(sourceColorLegend(120), "\n")
	plain := ansi.Strip(legend)
	for _, label := range []string{
		"Source BG:", "Q quality (e)", "▲ pattern (p)", "cognitive", "◆ note", "selected",
		"Source text:", "cyclomatic", "error return", "success return",
	} {
		if !strings.Contains(plain, label) {
			t.Fatalf("legend hides %q: %s", label, plain)
		}
	}
	for _, background := range []string{qualityBackground, patternBackground, annotationBackground, selectionBackground, "48;2;43;33;64"} {
		if !strings.Contains(legend, background) {
			t.Fatalf("legend omits source background %q", background)
		}
	}
}

func TestLegendFitsWithoutHidingSourceCursorOrKeys(t *testing.T) {
	for _, size := range [][2]int{{40, 18}, {80, 18}, {120, 30}, {160, 48}} {
		m := qualityModel()
		m.width, m.height, m.focus, m.sourceCursor = size[0], size[1], detailsPane, 4
		m = m.keepSourceCursorVisible()
		plain := ansi.Strip(m.View().Content)
		lines := strings.Split(plain, "\n")
		if len(lines) > m.height || !containsAll(plain, "Source BG:", "Source text:", "›   14 │") {
			t.Fatalf("legend hides source or exceeds height at %v:\n%s", size, plain)
		}
		if lines[len(lines)-1] != ansi.Strip(m.footer(m.width)) {
			t.Fatalf("legend hides key hints at %v", size)
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > m.width {
				t.Fatalf("legend exceeds width at %v: %q", size, line)
			}
		}
	}
}
