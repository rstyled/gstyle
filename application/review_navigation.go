package application

import (
	"fmt"
	"sort"

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
	sort.SliceStable(targets, func(i, j int) bool { return reviewTargetBefore(targets[i], targets[j]) })
	return targets
}

func reviewTargetBefore(left, right reviewTarget) bool {
	if left.fileIndex != right.fileIndex {
		return left.fileIndex < right.fileIndex
	}
	if left.functionIndex != right.functionIndex {
		return left.functionIndex < right.functionIndex
	}
	if left.startLine != right.startLine {
		return left.startLine < right.startLine
	}
	return left.endLine < right.endLine
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
	if delta > 0 {
		return m.nextReviewIndex(targets)
	}
	return m.previousReviewIndex(targets)
}

func (m Model) nextReviewIndex(targets []reviewTarget) int {
	for i, target := range targets {
		if m.reviewTargetAfterCursor(target) {
			return i
		}
	}
	return 0
}

func (m Model) previousReviewIndex(targets []reviewTarget) int {
	for i := len(targets) - 1; i >= 0; i-- {
		if m.reviewTargetBeforeCursor(targets[i]) {
			return i
		}
	}
	return len(targets) - 1
}

func (m Model) reviewTargetAfterCursor(target reviewTarget) bool {
	if target.fileIndex != m.fileIndex {
		return target.fileIndex > m.fileIndex
	}
	if target.functionIndex < 0 {
		return true
	}
	if target.functionIndex != m.functionIndex {
		return target.functionIndex > m.functionIndex
	}
	return target.endLine >= m.sourceLine()
}

func (m Model) reviewTargetBeforeCursor(target reviewTarget) bool {
	if target.fileIndex != m.fileIndex {
		return target.fileIndex < m.fileIndex
	}
	if target.functionIndex < 0 {
		return true
	}
	if target.functionIndex != m.functionIndex {
		return target.functionIndex < m.functionIndex
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
