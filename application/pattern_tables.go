package application

import (
	"fmt"

	"github.com/rstyled/gstyle/domain"
)

func (m Model) patternCountCell(count int) string {
	if !m.showPatterns {
		return muted.Render("   -")
	}
	if m.refreshing || m.patternsLoading {
		return amber.Render("   …")
	}
	if m.patternsError != nil {
		return danger.Render("   !")
	}
	return amber.Render(fmt.Sprintf(" %3d", count))
}

func (m Model) patternSourceBadge(function domain.Function) string {
	if !m.showPatterns {
		return ""
	}
	if m.patternsLoading {
		return muted.Render(" · ") + amber.Render("▲ patterns loading…")
	}
	if m.patternsError != nil {
		return muted.Render(" · ") + danger.Render("patterns failed: "+m.patternsError.Error())
	}
	if count := m.functionPatternCount(m.selectedFilePath(), function); count > 0 {
		return muted.Render(" · ") + amber.Bold(true).Render(fmt.Sprintf("▲ %d patterns", count))
	}
	return ""
}

func (m Model) filePatternCount(path string) int {
	if !m.showPatterns {
		return 0
	}
	return m.patternFile(path).count
}

func (m Model) functionPatternCount(path string, function domain.Function) int {
	if !m.showPatterns {
		return 0
	}
	seen := make(map[FindingMarker]bool)
	for _, marker := range m.patternFile(path).markers {
		if marker.overlaps(function) {
			seen[marker.key()] = true
		}
	}
	return len(seen)
}

func (marker FindingMarker) overlaps(function domain.Function) bool {
	end := max(marker.Line, marker.EndLine)
	last := max(function.Line, function.EndLine, function.Line+sourceLineCount(function.Source)-1)
	return marker.Line <= last && end >= function.Line
}
