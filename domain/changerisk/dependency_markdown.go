package changerisk

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

type moduleChange struct {
	name, before, after string
}

func moduleVersions(records []string) map[string]string {
	result := map[string]string{}
	for _, record := range records {
		scope, value, found := strings.Cut(record, ":module:")
		if !found {
			continue
		}
		var name, version, goVersion string
		if count, _ := fmt.Sscanf(value, "%q:%q:%q", &name, &version, &goVersion); count == 3 {
			if version == "" {
				version = "local"
			}
			result[scope+":"+name] = version
		}
	}
	return result
}

func moduleChanges(report Report) []moduleChange {
	before, after := moduleVersions(report.DependenciesRemoved), moduleVersions(report.DependenciesAdded)
	var changes []moduleChange
	for name, version := range before {
		changes = append(changes, moduleChange{name: name, before: version, after: after[name]})
		delete(after, name)
	}
	for name, version := range after {
		changes = append(changes, moduleChange{name: name, after: version})
	}
	slices.SortFunc(changes, func(a, b moduleChange) int {
		return cmp.Or(cmp.Compare(moduleChangeRank(a), moduleChangeRank(b)), cmp.Compare(a.name, b.name))
	})
	return changes
}

func moduleChangeRank(change moduleChange) int {
	if change.before != "" && change.after != "" {
		return 0
	}
	return 1
}

func dependencyLines(report Report) []string {
	var lines []string
	for _, change := range moduleChanges(report) {
		lines = append(lines, "- "+moduleChangeLine(change))
	}
	return lines
}

func moduleSummary(changes []moduleChange) string {
	if len(changes) == 0 {
		return "Package imports or module requirements changed."
	}
	var updated, added, removed int
	for _, change := range changes {
		switch {
		case change.before == "":
			added++
		case change.after == "":
			removed++
		default:
			updated++
		}
	}
	return fmt.Sprintf("Changes: %d updated, %d added, %d removed.", updated, added, removed)
}

func moduleChangeLine(change moduleChange) string {
	name := strings.TrimPrefix(change.name, ".:")
	if change.before == "" {
		return fmt.Sprintf("Added `%s` at `%s`.", text(name), text(change.after))
	}
	if change.after == "" {
		return fmt.Sprintf("Removed `%s` at `%s`.", text(name), text(change.before))
	}
	if change.before == change.after {
		return fmt.Sprintf("Updated module settings for `%s` at `%s`.", text(name), text(change.after))
	}
	return fmt.Sprintf("Updated `%s`: `%s` → `%s`.", text(name), text(change.before), text(change.after))
}
