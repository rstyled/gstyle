package application

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func sourceColorLegend(width int) []string {
	backgrounds := []string{
		withBackground(blue.Render("Q quality (e)"), qualityBackground),
		withBackground(amber.Render("▲ pattern (p)"), patternBackground),
		text.Background(cognitiveBackground).Render("cognitive"),
		withBackground(text.Render("◆ note"), annotationBackground),
		withBackground(text.Render("selected"), selectionBackground),
	}
	foregrounds := []string{amber.Render("cyclomatic"), danger.Render("error return"), green.Render("success return")}
	labels := []string{
		muted.Render("Source BG: ") + strings.Join(backgrounds, muted.Render(" · ")),
		muted.Render("Source text: ") + strings.Join(foregrounds, muted.Render(" · ")),
	}
	return strings.Split(ansi.Wrap(strings.Join(labels, "\n"), max(width, 1), ""), "\n")
}

func (m Model) footerLines(width int) []string {
	return append(sourceColorLegend(width), m.footer(width))
}

func (m Model) footerHeight() int {
	return len(sourceColorLegend(m.terminalWidth())) + 1
}

func (m Model) narrowPaneHeight() int {
	if m.height == 0 {
		return 0
	}
	return max(m.height-1-m.footerHeight(), 2)
}
