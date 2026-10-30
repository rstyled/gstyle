package application

import (
	"fmt"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
)

func (marker FindingMarker) key() FindingMarker {
	if marker.Diagnostic != nil {
		return FindingMarker{Diagnostic: marker.Diagnostic, Line: marker.Line}
	}
	if marker.Candidate != nil {
		return FindingMarker{Candidate: marker.Candidate}
	}
	return marker
}

func (marker FindingMarker) detailMessages() []string {
	if marker.Details != nil {
		return marker.Details.messages
	}
	return marker.uncachedDetailMessages()
}

func (marker FindingMarker) uncachedDetailMessages() []string {
	if marker.Diagnostic != nil {
		return marker.qualityDetailMessages()
	}
	candidate := marker.Candidate
	if candidate == nil {
		return nil
	}
	messages := marker.patternScopeMessages()
	for _, field := range [][2]string{
		{"Observed", candidate.Observation},
		{"Inference", candidate.Inference},
		{"Review", candidate.PossibleRefactor},
	} {
		if value := strings.TrimSpace(field[1]); value != "" {
			messages = append(messages, field[0]+": "+value)
		}
	}
	messages = append(messages, marker.relatedMessages()...)
	for _, evidence := range candidate.CounterEvidence {
		messages = append(messages, "Caution: "+evidence)
	}
	return messages
}

func (marker FindingMarker) patternScopeMessages() []string {
	if marker.Kind != string(patterns.CCGraphClone) && marker.Kind != string(patterns.InconsistentClone) {
		return nil
	}
	if marker.EndLine > marker.Line {
		return []string{fmt.Sprintf("Scope: whole function (lines %d-%d)", marker.Line, marker.EndLine)}
	}
	return []string{"Scope: whole function"}
}

func (marker FindingMarker) relatedMessages() []string {
	var messages []string
	seen := make(map[string]bool)
	for _, site := range marker.Candidate.Sites {
		if patternPathMatches(site.Path, marker.Path) && site.Line == marker.Line {
			continue
		}
		location := patternSiteLocation(site)
		if seen[location] {
			continue
		}
		seen[location] = true
		messages = append(messages, fmt.Sprintf("Related: %s (%s)", site.Name, location))
	}
	return messages
}

func patternSiteLocation(site patterns.Site) string {
	location := fmt.Sprintf("%s:%d", site.Path, site.Line)
	if site.EndLine > site.Line {
		return fmt.Sprintf("%s-%d", location, site.EndLine)
	}
	return location
}

func (m Model) scrollFindingDetails(delta int) (Model, bool) {
	if m.activeFindingLine == 0 || m.sourceViewportHeight() <= 0 {
		return m, false
	}
	rows := m.sourceDisplayRowCount(m.sourceCursor, m.sourceCursor)
	maximum := max(rows-m.sourceViewportHeight(), 0)
	offset := moveIndex(m.findingDetailOffset, delta, maximum+1)
	if offset == m.findingDetailOffset {
		return m, false
	}
	m.findingDetailOffset = offset
	return m, true
}

func (m Model) findingRowsOffset(line int) int {
	if line != m.activeFindingLine {
		return 0
	}
	return m.findingDetailOffset
}

func uniquePatternMarkers(markers []PatternMarker) []PatternMarker {
	var unique []PatternMarker
	seen := make(map[FindingMarker]bool)
	for _, marker := range markers {
		key := marker.key()
		if !seen[key] {
			seen[key] = true
			unique = append(unique, marker)
		}
	}
	return unique
}
