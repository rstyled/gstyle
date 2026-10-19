package quality

// Close the bound call graph before accepting a recursive edge. A cycle is
// harmless only when every reachable context has no observable effects.
func (r *callResolver) contextReadOnly(initial Call) bool {
	pending := []Call{initial}
	seen := map[string]bool{}
	for len(pending) > 0 {
		call := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		nested, known := r.readOnlyContext(call, seen)
		if !known {
			return false
		}
		pending = append(pending, nested...)
	}
	return true
}

func (r *callResolver) readOnlyContext(call Call, seen map[string]bool) ([]Call, bool) {
	proof := helperPurity{helpers: r.helpers, prefixes: r.prefixes}
	name, known := proof.target(call)
	if !known || name == "" {
		return nil, known
	}
	key := call.storageKey(name)
	if seen[key] {
		return nil, true
	}
	if len(seen) >= localContextLimit {
		return nil, false
	}
	seen[key] = true
	helper := call.bindHelper(r.helpers[name])
	return helper.Calls, readOnlyBody(helper)
}
