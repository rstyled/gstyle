package gopatterns

import (
	"go/types"

	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/resourceflow"
)

type protocolFacts struct {
	protocol                   resourceflow.Protocol
	sources, failures, helpers map[types.Object]bool
}

func summarizeProtocols(functions map[string]resourceFunction, exits callflow.ExitSummaries) []protocolFacts {
	var facts []protocolFacts
	for _, protocol := range resourceflow.Protocols() {
		if !packageUsesProtocol(functions, protocol) {
			continue
		}
		facts = append(facts, summarizeProtocol(functions, exits, protocol))
	}
	return facts
}

func packageUsesProtocol(functions map[string]resourceFunction, protocol resourceflow.Protocol) bool {
	for _, function := range functions {
		if resourceflow.UsesProtocol(function.graph, protocol) {
			return true
		}
	}
	return false
}

func summarizeProtocol(functions map[string]resourceFunction, exits callflow.ExitSummaries, protocol resourceflow.Protocol) protocolFacts {
	queries := make(map[string]resourceFunction, len(functions))
	for identity, function := range functions {
		function.query = function.query.WithProtocol(protocol)
		queries[identity] = function
	}
	helpers := cleanupSummaries(queries, exits)
	summaries := resourceFactories(queries, helpers)
	facts := protocolFacts{protocol: protocol, sources: map[types.Object]bool{}, failures: map[types.Object]bool{}, helpers: map[types.Object]bool{}}
	for identity, function := range functions {
		facts.sources[function.object] = summaries[identity].Fresh
		facts.failures[function.object] = summaries[identity].Fresh && summaries[identity].FailureEmpty && function.errorSecond
		facts.helpers[function.object] = helpers[identity]
	}
	return facts
}
