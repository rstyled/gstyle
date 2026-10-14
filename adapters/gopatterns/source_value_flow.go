package gopatterns

import "github.com/rstyled/gstyle/domain/pdg"

// Project PDG definition facts once. Downstream grouping never retains a
// compiler graph, re-resolves names, or assumes that matching spellings bind.
func projectSourceParameterFlow(sources *pdg.SourceContext, functions []FuncPdg) {
	graphs := make(map[string]*pdg.Graph, len(functions))
	for i := range functions {
		graphs[functions[i].Name] = &functions[i].Pdg
	}
	for i := range sources.Functions {
		fn := &sources.Functions[i]
		graph := graphs[fn.ID]
		if graph == nil {
			continue
		}
		projectFunctionFlow(fn, graph)
	}
}

func projectFunctionFlow(fn *pdg.SourceFunction, graph *pdg.Graph) {
	for i := range fn.ParametersList {
		param := &fn.ParametersList[i]
		symbol := sourceParameterSymbol(graph, param.Name)
		param.FlowKnown = symbol != 0
		param.Stable = parameterHasOneDefinition(graph, symbol) && !param.Addressed && !param.Captured
	}
}

func sourceParameterSymbol(graph *pdg.Graph, name string) pdg.Ref {
	for _, input := range graph.Function.Inputs {
		if input.Role == pdg.ParameterRole && graph.Tables.Text(input.Name.Value) == name {
			return input.Symbol
		}
	}
	return 0
}

func parameterHasOneDefinition(graph *pdg.Graph, symbol pdg.Ref) bool {
	if symbol == 0 {
		return false
	}
	count := 0
	for _, definition := range graph.Definitions {
		if definition.Symbol == symbol {
			count++
		}
	}
	return count == 1
}
