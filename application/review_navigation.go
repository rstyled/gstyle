package application

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/rstyled/gstyle/domain"
)

type reviewTarget struct {
	fileIndex     int
	functionIndex int
	startLine     int
	endLine       int
	annotation    *Annotation
}

func (m Model) focusAdjacentReview(delta int) Model {
	targets := m.reviewTargets()
	if len(targets) == 0 {
		return m
	}

	target := targets[m.adjacentReviewIndex(targets, delta)]
	m.fileIndex, m.functionIndex = target.fileIndex, target.functionIndex
	m = m.resetSourceWorkspace().showDetails()
	m.revision++
	if target.annotation != nil {
		m.sourceWorkspace = m.sourceWorkspace.focusAnnotation(*target.annotation, target.functionIndex >= 0)
	} else {
		function, _ := m.selectedFunction()
		m.sourceWorkspace = m.sourceWorkspace.focusFinding(target.startLine, function.Line)
	}
	if target.functionIndex < 0 {
		return m
	}
	return m.keepSourceCursorVisible()
}

func (m Model) reviewTargets() []reviewTarget {
	targets := m.findingTargets()
	for _, target := range m.annotationTargets() {
		targets = append(targets, reviewTarget{
			fileIndex: target.fileIndex, functionIndex: target.functionIndex,
			startLine: target.StartLine, endLine: target.EndLine, annotation: &target.Annotation,
		})
	}
	slices.SortStableFunc(targets, compareReviewTargets)
	return targets
}

func (m Model) findingTargets() []reviewTarget {
	var targets []reviewTarget
	seen := make(map[[3]int]int)
	for _, marker := range m.findingMarkers() {
		target, ok := m.findingTarget(marker)
		if !ok {
			continue
		}
		key := [3]int{target.fileIndex, target.functionIndex, target.startLine}
		if index, exists := seen[key]; exists {
			targets[index].endLine = max(targets[index].endLine, target.endLine)
			continue
		}
		seen[key] = len(targets)
		targets = append(targets, target)
	}
	return targets
}

func (m Model) findingTarget(marker PatternMarker) (reviewTarget, bool) {
	for fileIndex, file := range m.report.Files {
		if !patternPathMatches(marker.Path, file.Path) {
			continue
		}
		index := patternFunctionIndex(file.Functions, marker.Line)
		return reviewTarget{
			fileIndex: fileIndex, functionIndex: index,
			startLine: marker.Line, endLine: max(marker.Line, marker.EndLine),
		}, index >= 0
	}
	return reviewTarget{}, false
}

func (m Model) activeReviewIndex(targets []reviewTarget) int {
	for i, target := range targets {
		if m.reviewTargetActive(target) {
			return i
		}
	}
	return -1
}

func (m Model) reviewTargetActive(target reviewTarget) bool {
	if target.annotation != nil {
		return target.annotation.ID == m.activeAnnotationID
	}
	return target.fileIndex == m.fileIndex && target.functionIndex == m.functionIndex && target.startLine == m.activeFindingLine
}

func (m Model) adjacentReviewIndex(targets []reviewTarget, delta int) int {
	if index := m.activeReviewIndex(targets); index >= 0 {
		return (index + delta + len(targets)) % len(targets)
	}
	start, stop := 0, len(targets)
	if delta < 0 {
		start, stop = len(targets)-1, -1
	}
	for i := start; i != stop; i += delta {
		if m.reviewTargetAtCursor(targets[i], delta) {
			return i
		}
	}
	return start
}

func (m Model) reviewTargetAtCursor(target reviewTarget, delta int) bool {
	if target.fileIndex != m.fileIndex {
		return cmp.Compare(target.fileIndex, m.fileIndex)*delta > 0
	}
	if target.functionIndex < 0 {
		return true
	}
	if target.functionIndex != m.functionIndex {
		return cmp.Compare(target.functionIndex, m.functionIndex)*delta > 0
	}
	if delta > 0 {
		return target.endLine >= m.sourceLine()
	}
	return target.startLine <= m.sourceLine()
}

func (m Model) reviewNavigationLabel() string {
	targets := m.reviewTargets()
	if len(targets) == len(m.annotations) {
		return m.annotationNavigationLabel()
	}
	label := fmt.Sprintf(" findings (%d", len(targets))
	if position := m.activeReviewIndex(targets) + 1; position > 0 {
		label = fmt.Sprintf(" findings (%d/%d", position, len(targets))
	}
	if _, unmatched := m.annotationNavigationPosition(); unmatched > 0 {
		label += fmt.Sprintf("; %d unmatched", unmatched)
	}
	return label + ")"
}

func patternFunctionIndex(functions []domain.Function, line int) int {
	index := -1
	for i, function := range functions {
		if !functionHasSourceLine(function, line) {
			continue
		}
		if index < 0 || function.Line > functions[index].Line {
			index = i
		}
	}
	return index
}

func functionHasSourceLine(function domain.Function, line int) bool {
	return function.Line <= line && line < function.Line+sourceLineCount(function.Source)
}

func compareReviewTargets(left, right reviewTarget) int {
	return cmp.Or(
		cmp.Compare(left.fileIndex, right.fileIndex),
		cmp.Compare(left.functionIndex, right.functionIndex),
		cmp.Compare(left.startLine, right.startLine),
		cmp.Compare(left.endLine, right.endLine),
	)
}
