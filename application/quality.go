package application

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/rstyled/gstyle/domain"
	"github.com/rstyled/gstyle/domain/quality"
)

func (m Model) detailsLines(width, height int, title string) []string {
	if m.qualityView {
		return m.qualityLines(width, height)
	}
	return m.sourceLines(width, height, title)
}

func (m Model) qualityLines(width, height int) []string {
	lines := []string{m.paneTitle("Quality · e source", detailsPane)}
	function, ok := m.selectedFunction()
	if ok {
		lines = append(lines, text.Bold(true).Render(function.Name))
	}
	body := m.qualityBody()
	available := len(body)
	if height > 0 {
		available = max(height-len(lines), 0)
	}
	start := min(m.qualityOffset, max(len(body)-available, 0))
	contentWidth := paneContentWidth(width)
	for _, line := range body[start:min(start+available, len(body))] {
		lines = append(lines, truncate(line, contentWidth))
	}
	return fitPaneLines(lines, width, height)
}

func (m Model) qualityBody() []string {
	if m.refreshing {
		return []string{muted.Render("Scanning quality…")}
	}
	if m.report.Quality == nil {
		return []string{muted.Render("Quality not analyzed")}
	}
	if m.report.Quality.Status == "error" {
		return append([]string{danger.Render("Quality analysis failed")}, strings.Split(m.report.Quality.Error, "\n")...)
	}
	function, ok := m.selectedFunction()
	if !ok {
		return []string{muted.Render("No function selected")}
	}
	if function.Quality == nil {
		return []string{muted.Render("Quality unavailable for this function"), muted.Render("Inactive build, generated, or nested literal")}
	}
	return functionQualityLines(function.Quality)
}

func (m Model) qualityBodyRowCount() int {
	if m.refreshing || m.report.Quality == nil {
		return 1
	}
	if m.report.Quality.Status == "error" {
		return 2 + strings.Count(m.report.Quality.Error, "\n")
	}
	function, ok := m.selectedFunction()
	if !ok {
		return 1
	}
	q := function.Quality
	if q == nil {
		return 2
	}
	// Six summary rows and two heading rows for each evidence section.
	visible := knownQuality(q)
	return 10 + len(visible.Diagnostics) + len(visible.Effects) + len(visible.MutationEvents)
}

func (m Model) qualityHeader() string {
	if m.report.Quality == nil {
		return ""
	}
	if m.report.Quality.Status == "unavailable" {
		return " · quality unavailable"
	}
	if m.report.Quality.Status == "error" {
		return " · quality failed"
	}
	if m.report.Quality.Report == nil {
		return ""
	}
	indexed := m.ensureFindingIndex()
	return fmt.Sprintf(" · Q %d", indexed.findingLookup.qualityCount)
}

func (m Model) detailsViewName() string {
	if m.qualityView {
		return "quality"
	}
	return "source"
}

func (m Model) setDetailsView(view string) Model {
	showQuality := view == "quality"
	if m.qualityView == showQuality {
		return m
	}
	m.qualityView = showQuality
	m.revision++
	return m
}

func (m Model) toggleQualityView() Model {
	view := "quality"
	if m.qualityView {
		view = "source"
	}
	m = m.setDetailsView(view)
	return m.setFocus(detailsPane)
}

func (m Model) moveDetails(delta int) Model {
	if !m.qualityView {
		return m.moveSourceCursor(delta)
	}
	rows := m.qualityBodyRowCount()
	maximum := max(rows-m.qualityViewportHeight(rows), 0)
	next := moveIndex(min(m.qualityOffset, maximum), delta, maximum+1)
	if next == m.qualityOffset {
		return m
	}
	m.qualityOffset = next
	m.revision++
	return m
}

func (m Model) qualityViewportHeight(rows int) int {
	if m.height == 0 {
		return rows
	}
	return max(m.sourceViewportHeight()+m.sourceHeaderHeight()-2, 1)
}

func (m Model) updateDetailsControl(command controlCommand) (Model, tea.Cmd) {
	if command.action != setDetailsViewAction {
		return m.updateSourceControl(command)
	}
	m = m.setDetailsView(command.view)
	m = m.setFocus(detailsPane)
	command.answer(m.controlState(), nil)
	return m, nil
}

func functionQualityLines(q *domain.FunctionQuality) []string {
	visible := knownQuality(q)
	return slices.Concat(
		[]string{
			qualityDensityLabel(q, visible.Mutations),
			fmt.Sprintf("%d known targets", visible.MutatedTargets),
			muted.Render("Known effects only"),
			muted.Render("Modeled effects; zero is not proof of purity"),
			"",
			amber.Render(fmt.Sprintf("Findings · %d", len(visible.Diagnostics))),
		},
		diagnosticMessages(visible.Diagnostics),
		[]string{"", blue.Render(fmt.Sprintf("Effects · %d", len(visible.Effects)))},
		effectLines(visible.Effects),
		mutationQualityLines(visible.MutationEvents),
	)
}

func diagnosticMessages(diagnostics []quality.Diagnostic) []string {
	messages := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		messages = append(messages, diagnostic.Message)
	}
	return messages
}

func effectLines(effects []quality.Effect) []string {
	lines := make([]string, 0, len(effects))
	for _, effect := range effects {
		lines = append(lines, fmt.Sprintf("L%d %s: %s", effect.Line, effect.Kind, effect.Detail))
	}
	return lines
}

func mutationQualityLines(mutations []quality.Mutation) []string {
	lines := []string{"", blue.Render(fmt.Sprintf("Mutations · %d", len(mutations)))}
	for _, mutation := range mutations {
		lines = append(lines, mutationQualityLine(mutation))
	}
	return lines
}

func qualityBadge(function domain.Function) string {
	if function.Quality == nil {
		return ""
	}
	return amber.Render(fmt.Sprintf(" · Q %d", len(knownDiagnostics(function.Quality))))
}

func mutationQualityLine(mutation quality.Mutation) string {
	target := mutation.Root
	if mutation.FieldPath != "" {
		target += "." + mutation.FieldPath
	}
	return fmt.Sprintf("L%d mutation: %s (%s)", mutation.Line, target, mutation.Provenance)
}
