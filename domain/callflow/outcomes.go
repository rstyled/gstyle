package callflow

import (
	"fmt"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

// Outcomes is a may-exit set. Unknown abstracts missing execution evidence;
// Diverges includes a potentially nonterminating/blocking path.
type Outcomes uint16

const (
	Returns Outcomes = 1 << iota
	RecoversInherited
	Panics
	PanicDuringGoexit
	ExitsGoroutine
	Aborts
	Diverges
	UnknownExit
)

type ExitSummary struct {
	returned                string
	arity                   int
	Ordinary, DeferredPanic Outcomes
	Receiver                bool
	invocations             *exitInvocations
}

func (summary ExitSummary) outcomes(inherited bool) Outcomes {
	result := summary.Ordinary
	if inherited {
		result = summary.DeferredPanic
	}
	if !validSummaryOutcomes(result, inherited) {
		return result | UnknownExit
	}
	// Receiver wrappers may dereference before entering the declared body.
	if summary.Receiver {
		result |= Panics
	}
	return result
}

type ExitSummaries map[string]ExitSummary

type callTargets struct {
	names []string
	known bool
}

type CallSites struct {
	deferred       deferflow.Index
	graph          *pdg.Graph
	paths          *bindingflow.Query
	facts          bindingflow.Facts
	targets        map[pdg.Ref]callTargets
	entries        map[pdg.Ref]string
	resourceInputs map[pdg.Ref][]pdg.Position
}

// NewCallSites owns its indexes and borrows a validated canonical graph.
func NewCallSites(graph *pdg.Graph) (*CallSites, error) {
	if graph == nil {
		return nil, nil
	}
	query, err := bindingflow.New(graph)
	if err != nil {
		return nil, err
	}
	facts, err := query.ValueFacts()
	if err != nil {
		return nil, err
	}
	calls := &CallSites{graph: graph, paths: query, facts: facts, targets: map[pdg.Ref]callTargets{}, entries: map[pdg.Ref]string{}, resourceInputs: map[pdg.Ref][]pdg.Position{}}
	if err := calls.index(query); err != nil {
		return nil, err
	}
	if err := calls.validateDeferredInputs(); err != nil {
		return nil, err
	}
	return calls, nil
}

func (calls *CallSites) Outcomes(summaries ExitSummaries, ref pdg.Ref, inherited bool) Outcomes {
	return calls.outcomesWithValues(summaries, ref, inherited, calls.defaultValues())
}

func (calls *CallSites) index(query *bindingflow.Query) error {
	graph := calls.graph
	for i, node := range graph.Nodes {
		if node.Category != pdg.Call {
			continue
		}
		entry, err := readCallBoundary(graph, graph.NodeAttributes(node).Extensions)
		if err != nil {
			return err
		}
		calls.entries[pdg.Ref(i+1)] = entry.kind
		calls.resourceInputs[pdg.Ref(i+1)] = entry.inputs
		names, known := frameTargets(graph, query, pdg.Ref(i+1))
		calls.targets[pdg.Ref(i+1)] = callTargets{names: names, known: known}
	}
	return nil
}

func (calls *CallSites) outcomesWithValues(summaries ExitSummaries, ref pdg.Ref, inherited bool, context bindingflow.Evaluation) Outcomes {
	if calls == nil {
		return UnknownExit
	}
	targets, found := calls.targets[ref]
	if !found || !targets.known || len(targets.names) == 0 {
		return UnknownExit
	}
	var outcomes Outcomes
	invocation := exitInvocationContext{ref: ref, inherited: inherited, values: context}
	for _, name := range targets.names {
		outcomes |= calls.targetWithValues(name, summaries, invocation)
	}
	return outcomes
}

type exitBody struct {
	captures   []pdg.Ref
	graph      *pdg.Graph
	flow       *pdg.NormalFlow
	paths      *bindingflow.Query
	calls      *CallSites
	machine    deferflow.Machine[struct{}]
	facts      bindingflow.Facts
	operations *OperationEffects
}

type operationEffect struct {
	Panic    bool `json:"panic,omitempty"`
	NilInput bool `json:"nil_input,omitempty"`
	Block    bool `json:"block,omitempty"`
	Unknown  bool `json:"unknown,omitempty"`
}

func targetOutcomes(name string, summaries ExitSummaries, inherited bool) Outcomes {
	switch name {
	case "panic":
		return Panics
	case "runtime.Goexit":
		return ExitsGoroutine
	case "os.Exit":
		return Aborts
	}
	if summary, known := summaries[name]; known {
		return summary.outcomes(inherited)
	}
	return externalExitOutcomes(name)
}

func externalExitOutcomes(name string) Outcomes {
	switch name {
	case "recover", "len", "cap", "complex", "real", "imag", "min", "max", "print", "println", "errors.New", "strings.TrimSpace", "path.Clean":
		return Returns
	case "append", "clear", "close", "copy", "delete", "make", "new", "os.File.Close":
		return Returns | Panics
	default:
		return UnknownExit
	}
}

// ExitEffects interprets canonical body/registration flow and composes local
// call outcomes to a monotone fixed point. It does not execute foreign code or
// bind captured cells, callback arguments, heap state or recovered payloads.
// Closure identities compose only when their canonical body graphs are supplied.
func ExitEffects(graphs map[string]*pdg.Graph) (ExitSummaries, error) {
	prepared := make(map[string]Queries, len(graphs))
	for name, graph := range graphs {
		queries, err := NewQueries(graph)
		if err != nil {
			return nil, fmt.Errorf("exit summary %s: %w", name, err)
		}
		prepared[name] = queries
	}
	return ExitEffectsWithQueries(prepared)
}

// ExitEffectsWithQueries reuses the immutable indexes prepared by NewQueries.
func ExitEffectsWithQueries(prepared map[string]Queries) (ExitSummaries, error) {
	bodies := make(map[string]*exitBody, len(prepared))
	for name, queries := range prepared {
		body := exitBodyWithQueries(queries)
		if body != nil && body.graph.Tables.Text(body.graph.Function.ID) != name {
			return nil, fmt.Errorf("exit summary identity mismatch: %s", name)
		}
		bodies[name] = body
	}
	return solveExitBodies(bodies), nil
}

func validSummaryOutcomes(result Outcomes, inherited bool) bool {
	allowed := Returns | Panics | PanicDuringGoexit | ExitsGoroutine | Aborts | Diverges | UnknownExit
	if inherited {
		allowed |= RecoversInherited
	}
	return result&^allowed == 0
}
