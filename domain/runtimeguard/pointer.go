package runtimeguard

import (
	"encoding/json"
	"github.com/rstyled/gstyle/domain/pdg"
	"strconv"
	"strings"
)

func (v *graphView) fieldBindings() {
	v.fields = make([]pdg.Ref, len(v.graph.Nodes))
	v.dereferences = make(map[pdg.Ref][]pdg.Ref)
	v.paths = make(map[string]pdg.Ref)
	for i := range v.graph.Nodes {
		ref := pdg.Ref(i + 1)
		var path string
		if decodeFact(v.fact(ref, "field-path"), &path) {
			v.fields[i] = v.fieldBinding(path)
		}
		var paths []string
		if decodeFact(v.fact(ref, "dereference-paths"), &paths) {
			for _, path := range paths {
				v.dereferences[ref] = append(v.dereferences[ref], v.fieldBinding(path))
			}
		}
	}
}

func (v *graphView) fieldBinding(path string) pdg.Ref {
	if ref := v.paths[path]; ref != 0 {
		return ref
	}
	root, _, _ := strings.Cut(path, "/")
	id, err := strconv.Atoi(root)
	if err != nil || id <= 0 || id > len(v.graph.Symbols) {
		return 0
	}
	ref := pdg.Ref(len(v.kinds) + 1)
	v.paths[path] = ref
	v.kinds = append(v.kinds, `"pointer"`)
	v.unstable = append(v.unstable, v.unstable[id-1] || extension(v.graph, v.graph.Extensions, "concurrent-memory") == "true")
	v.tracked = append(v.tracked, true)
	return ref
}

func (v *graphView) embeddedValue(ref pdg.Ref, state State, receiver Value) Value {
	paths := v.dereferences[ref]
	if len(paths) == 0 {
		return Value{}
	}
	if v.fact(ref, "receiver-pointer") == "true" && !receiver.NonNil {
		return Value{}
	}
	for _, path := range paths {
		if !v.binding(path, state).NonNil {
			return Value{}
		}
	}
	return Value{NonNil: true}
}

// Local copies share a value identity until a write separates them. Static
// write identities are forgotten before reuse, so loop iterations cannot alias.
func (v *graphView) assign(ref, symbol pdg.Ref, value Value, state State) {
	value = v.assignmentValue(symbol, value)
	if v.kinds[symbol-1] == `"pointer"` && value.Identity == 0 {
		value.Identity = uint64(ref)*2 + 1
		v.forgetIdentity(value.Identity, state)
	}
	state[symbol] = value
}

func (v *graphView) forgetIdentity(identity uint64, state State) {
	for symbol, value := range state {
		if value.Identity == identity {
			value.Identity = 0
			state[symbol] = value
		}
	}
}

func (v *graphView) propagatePointer(symbol pdg.Ref, value Value, state State) {
	if value.Identity != 0 {
		for i, tracked := range v.tracked {
			if !tracked {
				continue
			}
			peer := pdg.Ref(i + 1)
			previous := v.binding(peer, state)
			if previous.Identity == value.Identity {
				previous.NonNil, previous.Nil, previous.Guarded = value.NonNil, value.Nil, value.Guarded
				state[peer] = previous
			}
		}
	}
	state[symbol] = value
}

// Unknown calls and writes may change a field through an alias. Forget field
// proofs at those sites instead of extending local binding facts into memory.
func (v *graphView) invalidateFields(ref pdg.Ref, state State) {
	node := v.graph.Nodes[ref-1]
	if node.Category != pdg.Call && len(v.attrs(ref).Writes) == 0 {
		return
	}
	for symbol, value := range state {
		if int(symbol) > len(v.graph.Symbols) {
			delete(state, symbol)
			continue
		}
		if fieldIdentity(value.Identity, len(v.graph.Symbols)) {
			value.Identity = 0
			state[symbol] = value
		}
	}
}

func fieldIdentity(identity uint64, symbols int) bool {
	return identity != 0 && identity%2 == 0 && identity/2 > uint64(symbols)
}

func decodeFact(raw string, target any) bool {
	return raw != "" && json.Unmarshal([]byte(raw), target) == nil
}
