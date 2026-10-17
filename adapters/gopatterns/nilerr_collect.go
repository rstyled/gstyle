package gopatterns

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"go/token"
	"go/types"
	"slices"
)

type nilErrCollector struct {
	fset      *token.FileSet
	info      *types.Info
	positions []int
	hits      map[token.Pos]patterns.NilErrHit
}

// One map owns both findings and their deduplication identity.
func (c *nilErrCollector) findings() []patterns.NilErrHit {
	positions := make([]token.Pos, 0, len(c.hits))
	for position := range c.hits {
		positions = append(positions, position)
	}
	slices.Sort(positions)
	var findings []patterns.NilErrHit
	for _, position := range positions {
		findings = append(findings, c.hits[position])
	}
	return findings
}
