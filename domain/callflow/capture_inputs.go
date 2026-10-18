package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"strings"
)

func captureInputs(graph *pdg.Graph) []pdg.Ref {
	var result []pdg.Ref
	for i, symbol := range graph.Symbols {
		if symbol.Role == pdg.CaptureRole {
			result = append(result, pdg.Ref(i+1))
		}
	}
	return result
}

func (calls *CallSites) completeInputKey(invocations *exitInvocations, arguments []bindingflow.Value, context bindingflow.Evaluation) string {
	values := make([]byte, len(invocations.inputs)+len(invocations.captures))
	copy(values, calls.knownInputKey(invocations.inputs, arguments))
	if len(context.Captures) != 0 {
		links := invocations.captureBindings(calls.graph)
		for i, symbol := range invocations.captures {
			source, found := links[symbol]
			if !found {
				continue
			}
			values[len(invocations.inputs)+i] = byte(abstractValue(context.Captures[source]))
		}
	}
	key := string(values)
	if strings.Trim(key, "\x00") == "" {
		return ""
	}
	return key
}

func (invocations *exitInvocations) captureBindings(owner *pdg.Graph) map[pdg.Ref]pdg.Ref {
	if links, found := invocations.environments[owner]; found {
		return links
	}
	links, err := bindingflow.CaptureBindings(invocations.graph, owner)
	if err != nil {
		links = nil
	}
	if invocations.building {
		invocations.environments[owner] = links
	}
	return links
}
