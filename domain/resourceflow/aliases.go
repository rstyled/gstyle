package resourceflow

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
	"slices"
)

// Alias values belong to one path. All RHS values use the state before a
// parallel assignment; every direct overwrite kills the previous alias.
func (p *proof) assignAliases(ref pdg.Ref) {
	writes := p.query.paths.Writes(ref)
	if len(writes) == 0 {
		return
	}
	p.policy.aliases = p.query.nextAliases(writes, p.policy)
}

func (q *Query) nextAliases(writes []bindingflow.Write, policy Policy) map[pdg.Ref]bool {
	if unknownBindingWrite(writes) {
		return map[pdg.Ref]bool{}
	}
	next := map[pdg.Ref]bool{}
	maps.Copy(next, policy.aliases)
	for _, write := range writes {
		if write.Symbol == 0 || write.Location != 0 {
			continue
		}
		if q.resource(write.Source, policy) {
			next[write.Symbol] = true
		} else {
			delete(next, write.Symbol)
		}
	}
	return next
}

func ownedSymbol(symbol pdg.Ref, policy Policy) bool {
	if symbol == 0 {
		return false
	}
	if policy.aliases == nil {
		return symbol == policy.Resource
	}
	return policy.aliases[symbol]
}

func aliasKey(aliases map[pdg.Ref]bool) string {
	symbols := slices.Collect(maps.Keys(aliases))
	slices.Sort(symbols)
	return fmt.Sprint(symbols)
}

func unknownBindingWrite(writes []bindingflow.Write) bool {
	for _, write := range writes {
		if write.UnknownTarget() {
			return true
		}
	}
	return false
}
