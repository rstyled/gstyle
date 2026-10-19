package quality

// Read-only recursion does not add observable effects. Start with all visible
// helpers, then propagate every unproved operation to its callers. This proves
// effects, not termination, and never expands a recursive call tree.
type helperPurity struct {
	helpers  map[string]Helper
	prefixes []Prefix
	callers  map[string][]string
	pure     map[string]bool
	pending  []string
}

func readOnlyHelpers(helpers map[string]Helper, prefixes []Prefix) map[string]bool {
	proof := helperPurity{helpers: helpers, prefixes: prefixes, callers: map[string][]string{}, pure: map[string]bool{}}
	for name, helper := range helpers {
		if !proof.index(name, helper) {
			proof.pending = append(proof.pending, name)
		}
	}
	return proof.solve()
}

func (proof *helperPurity) index(name string, helper Helper) bool {
	proof.pure[name] = true
	valid := readOnlyBody(helper)
	for _, call := range helper.Calls {
		target, known := proof.target(call)
		valid = valid && known
		if target != "" {
			proof.callers[target] = append(proof.callers[target], name)
		}
	}
	return valid
}

func readOnlyBody(helper Helper) bool {
	if len(helper.Effects) != 0 {
		return false
	}
	for _, mutation := range helper.Mutations {
		if mutation.Provenance != Local {
			return false
		}
	}
	return true
}

func (proof *helperPurity) target(call Call) (string, bool) {
	name := NormalizeCallee(call.Callee)
	kind, configured := call.policyKind(proof.prefixes)
	if configured {
		return "", kind == None || (kind == MutationEffect && call.HasLocalContainerTarget())
	}
	if !call.staticLocal() {
		return "", false
	}
	_, found := proof.helpers[name]
	return name, found
}

func (proof *helperPurity) solve() map[string]bool {
	for next := 0; next < len(proof.pending); next++ {
		name := proof.pending[next]
		if !proof.pure[name] {
			continue
		}
		delete(proof.pure, name)
		proof.pending = append(proof.pending, proof.callers[name]...)
	}
	return proof.pure
}
