package changerisk

import (
	"cmp"
	"fmt"
	"html"
	"slices"
	"strings"
)

func Markdown(report Report) string {
	lines := []string{
		"**Change risk: " + string(report.Risk) + "**", "",
		"**PDG changes:** " + verdict(report.PDGRisk) + "  ", pdgSummary(report), "",
		"**Dependencies:** " + dependencyVerdict(report) + "  ", dependencySummary(report), "",
		"**Compatibility:** unknown 👀  ", "API and CLI contracts have not been compared.", "",
		"**Checks:** unknown 👀  ", "CI results have not been collected.", "",
		"**Coverage:** unknown 👀  ", "Base-to-head test coverage has not been measured.", "",
		fmt.Sprintf("Compared `%s` → `%s`.", text(report.Base), text(report.Head)),
	}
	return strings.Join(lines, "\n") + "\n" + graphSection(prioritizedChanges(report))
}

func prioritizedChanges(report Report) []Delta {
	levels := map[string]Level{}
	for _, reason := range report.Reasons {
		levels[reason.Function] = strongest(levels[reason.Function], reason.Level)
	}
	changes := slices.Clone(report.Changes)
	slices.SortStableFunc(changes, func(a, b Delta) int {
		return cmp.Or(cmp.Compare(levelScore(levels[b.Name]), levelScore(levels[a.Name])), cmp.Compare(a.Name, b.Name))
	})
	return changes
}

func verdict(level Level) string {
	return string(level) + " " + map[Level]string{Low: "✅", Medium: "👀", High: "⚠️", Unknown: "👀"}[level]
}

func text(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	return strings.ReplaceAll(html.EscapeString(value), "`", "&#96;")
}

func pdgSummary(report Report) string {
	for _, reason := range report.Reasons {
		if reason.Level == report.PDGRisk && reason.Rule != "dependency_change" {
			return reasonSummary(reason)
		}
	}
	if report.ChangedFunctions == 0 {
		return "No changes detected in the compared function projection."
	}
	return fmt.Sprintf("%d function(s) changed; see graph changes.", report.ChangedFunctions)
}

func reasonSummary(reason Reason) string {
	message := text(reason.Message)
	if reason.Path != "" {
		message += fmt.Sprintf(" in `%s:%d`", text(reason.Path), reason.Line)
	}
	return message + "."
}

func dependencyVerdict(report Report) string {
	if report.DependencyRisk == Low && !report.DependenciesChanged {
		return "unchanged ✅"
	}
	return verdict(report.DependencyRisk)
}

func dependencySummary(report Report) string {
	for _, reason := range report.Reasons {
		if reason.Rule == "dependencies_unavailable" || reason.Rule == "dependency_change" {
			return reasonSummary(reason)
		}
	}
	return "No module-requirement, replacement or package-import changes."
}
