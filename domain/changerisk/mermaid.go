package changerisk

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

const maxDiagramNodes = 18
const maxDiagramEdges = 24
const maxDiagramFunctions = 3

func graphSection(changes []Delta) string {
	var diagrams []string
	for _, delta := range changes {
		if len(diagrams) == maxDiagramFunctions {
			break
		}
		if diagram := mermaid(delta); diagram != "" {
			diagrams = append(diagrams, diagram)
		}
	}
	if len(diagrams) == 0 {
		return ""
	}
	return "\n<details>\n<summary>Graph changes</summary>\n\n" +
		fmt.Sprintf("Showing %d of %d changed functions. `+` added; `−` removed. Context is bounded.\n\n", len(diagrams), len(changes)) +
		strings.Join(diagrams, "\n") + "\n</details>\n"
}

func diagramNodes(delta Delta) ([]Node, map[string]string) {
	var nodes []Node
	marks := map[string]string{}
	for _, group := range []struct {
		nodes []Node
		mark  string
	}{{delta.Added, "+ "}, {delta.Removed, "− "}, {delta.Context, ""}} {
		for _, node := range callNodesFirst(group.nodes) {
			if len(nodes) == maxDiagramNodes {
				return nodes, marks
			}
			nodes = append(nodes, node)
			marks[node.Key] = group.mark
		}
	}
	return nodes, marks
}

func callNodesFirst(nodes []Node) []Node {
	result := slices.Clone(nodes)
	slices.SortStableFunc(result, func(a, b Node) int {
		if (a.Callee == "") == (b.Callee == "") {
			return 0
		}
		if a.Callee != "" {
			return -1
		}
		return 1
	})
	return result
}

func mermaid(delta Delta) string {
	if delta.Ambiguous {
		return ""
	}
	nodes, marks := diagramNodes(delta)
	if len(nodes) == 0 {
		return ""
	}
	ids := map[string]string{}
	var lines []string
	lines = append(lines, "**"+text(delta.Path)+" · "+text(delta.Name)+"**", "", "```mermaid", "flowchart TD")
	for i, node := range nodes {
		id := fmt.Sprintf("n%d", i)
		ids[node.Key] = id
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]", id, text(marks[node.Key]+boundedLabel(node.Label))))
	}
	return strings.Join(append(lines, diagramEdges(delta.Edges, ids)...), "\n") + "\n```\n"
}

func boundedLabel(label string) string {
	runes := []rune(label)
	if len(runes) > 80 {
		return string(runes[:77]) + "…"
	}
	return label
}

func diagramEdges(edges []ChangedEdge, ids map[string]string) []string {
	unique := map[string]ChangedEdge{}
	for _, edge := range edges {
		unique[edge.Key+edge.Change] = edge
	}
	var lines []string
	for _, key := range slices.Sorted(maps.Keys(unique)) {
		if len(lines) == maxDiagramEdges {
			break
		}
		edge := unique[key]
		if ids[edge.From] == "" || ids[edge.To] == "" {
			continue
		}
		mark := map[string]string{"added": "+ ", "removed": "− "}[edge.Change]
		lines = append(lines, fmt.Sprintf("  %s -->|\"%s\"| %s", ids[edge.From], text(mark+edge.Label), ids[edge.To]))
	}
	return lines
}
