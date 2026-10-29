package application

import (
	"fmt"
	"strings"
)

func (marker FindingMarker) key() FindingMarker {
	if marker.Diagnostic != nil {
		return FindingMarker{Diagnostic: marker.Diagnostic}
	}
	if marker.Candidate != nil {
		return FindingMarker{Candidate: marker.Candidate}
	}
	return marker
}

func (marker FindingMarker) detailMessages() []string {
	if marker.Diagnostic != nil {
		return qualityDetailMessages(marker.Diagnostic)
	}
	candidate := marker.Candidate
	if candidate == nil {
		return nil
	}
	var messages []string
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

func (marker FindingMarker) relatedMessages() []string {
	var messages []string
	seen := make(map[string]bool)
	for _, site := range marker.Candidate.Sites {
		if patternPathMatches(site.Path, marker.Path) && site.Line == marker.Line {
			continue
		}
		location := fmt.Sprintf("%s:%d", site.Path, site.Line)
		if seen[location] {
			continue
		}
		seen[location] = true
		messages = append(messages, fmt.Sprintf("Related: %s (%s)", site.Name, location))
	}
	return messages
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
