package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"golang.org/x/tools/go/cfg"
)

// applyCallContinuations runs before all dependency passes use the shared CFG.
// A self edge retains infinite paths for divergence-sensitive post-dominance.
func (b *irBuilder) applyCallContinuations(flow *cfg.CFG) {
	for _, block := range flow.Blocks {
		if len(block.Nodes) != 1 {
			continue
		}
		outcomes := b.callExits[b.nodes[block.Nodes[0]]]
		if outcomes == 0 {
			continue
		}
		block.Succs = nil
		if outcomes&callflow.Diverges != 0 {
			block.Succs = []*cfg.Block{block}
			if outcomes&^callflow.Diverges != 0 {
				exit := &cfg.Block{Index: int32(len(flow.Blocks))}
				flow.Blocks = append(flow.Blocks, exit)
				block.Succs = append(block.Succs, exit)
			}
		}
	}
}
