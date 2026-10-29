package application

import tea "charm.land/bubbletea/v2"

type patternState struct {
	patternMarkers  []PatternMarker
	patternsLoading bool
	patternsError   error
}

func (m Model) finishAnalysis(message reportMsg) (Model, tea.Cmd) {
	m = m.withReport(message)
	if message.err != nil {
		return m, nil
	}
	return m.startPatternScan()
}

func (m Model) resetPatternScan() Model {
	m.patternGeneration++
	m.patternState = patternState{}
	return m
}

func (m Model) startPatternScan() (Model, tea.Cmd) {
	m = m.resetPatternScan()
	m.patternsLoading = true
	return m, m.loadPatternsCmd()
}

func (m Model) withPatterns(message patternsMsg) Model {
	if message.generation != m.patternGeneration {
		return m
	}
	m.patternState = patternState{patternMarkers: message.markers, patternsError: message.err}
	m.revision++
	return m
}

func (m Model) analysisLoadingLabel() string {
	if m.refreshing {
		return " · loading code + quality…"
	}
	if m.patternsLoading {
		return " · loading patterns…"
	}
	if m.patternsError != nil {
		return " · patterns failed"
	}
	return ""
}
