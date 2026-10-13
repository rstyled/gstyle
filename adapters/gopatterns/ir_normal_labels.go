package gopatterns

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

var normalMiningKinds = map[pdg.Category]patterns.NodeKind{
	pdg.Declaration: patterns.Let, pdg.Control: patterns.Branch, pdg.Return: patterns.Return,
	pdg.FormalInput: patterns.Param,
}

var normalMiningDetails = map[pdg.Category]string{pdg.Read: "read", pdg.Merge: "merge", pdg.Assignment: "assignment", pdg.FormalOutput: "result"}

// The abstraction consumer must not silently lose def/read/value edges when
// scalar reads and joins become explicit. All scalar-profile entities receive
// navigation-compatible mining labels; strict CCGraph still uses five categories.
func (b *irBuilder) normalMiningLabels() {
	for i, node := range b.graph.Nodes {
		if node.MatchLabel != 0 {
			continue
		}
		attrs := b.attributes[node.Attributes-1]
		label := b.normalLabel(node.Category, attrs)
		b.graph.Nodes[i].MatchLabel = b.pool.MatchLabel(label)
	}
}

func (b *irBuilder) normalLabel(category pdg.Category, attrs pdg.Attributes) pdg.MatchLabel {
	kind := normalMiningKinds[category]
	if kind == "" {
		kind = patterns.Op
	}
	detail := normalMiningDetails[category]
	if attrs.Operator.Status == pdg.Known {
		detail = b.pool.Tables.Text(attrs.Operator.Value)
	}
	return pdg.MatchLabel{Kind: b.pool.Text(string(kind)), Detail: b.pool.Text(detail)}
}
