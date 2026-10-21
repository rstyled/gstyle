package callflow

import (
	"context"
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
)

// Program retains validated graph indexes, never compiler objects. Registration
// publishes finalized functions; each solve owns its invocation state. Queries
// are immutable until their function is replaced in a new source revision.
type Program struct{ queries map[string]Queries }

func NewProgram() *Program { return &Program{queries: map[string]Queries{}} }

func (program *Program) Add(graph *pdg.Graph) error {
	if program == nil || graph == nil {
		return fmt.Errorf("program function graph is required")
	}
	queries, err := NewQueries(graph)
	if err != nil {
		return err
	}
	if program.queries == nil {
		program.queries = map[string]Queries{}
	}
	program.queries[graph.Tables.Text(graph.Function.ID)] = queries
	return nil
}

// ExitEffects solves current functions and their loaded transitive callees.
// Missing targets retain the normal UnknownExit contract. The current graphs
// override older registrations without publishing provisional graph storage.
func (program *Program) ExitEffects(graphs map[string]*pdg.Graph) (ExitSummaries, error) {
	return program.ExitEffectsContext(context.Background(), graphs)
}

// ExitEffectsContext solves without publishing a canceled analysis.
func (program *Program) ExitEffectsContext(ctx context.Context, graphs map[string]*pdg.Graph) (ExitSummaries, error) {
	if program == nil {
		return nil, fmt.Errorf("program queries are required")
	}
	prepared := make(map[string]Queries, len(graphs))
	for name, graph := range graphs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		queries, err := NewQueries(graph)
		if err != nil {
			return nil, fmt.Errorf("program exit %s: %w", name, err)
		}
		prepared[name] = queries
	}
	program.includeCallees(prepared)
	return ExitEffectsWithQueriesContext(ctx, prepared)
}

func (program *Program) includeCallees(prepared map[string]Queries) {
	queue := make([]Queries, 0, len(prepared))
	for _, query := range prepared {
		queue = append(queue, query)
	}
	for i := 0; i < len(queue); i++ {
		for _, name := range queryTargets(queue[i]) {
			if _, found := prepared[name]; found {
				continue
			}
			query, found := program.queries[name]
			if !found {
				continue
			}
			prepared[name] = query
			queue = append(queue, query)
		}
	}
}

func queryTargets(query Queries) []string {
	if query.Calls == nil {
		return nil
	}
	var names []string
	for _, target := range query.Calls.targets {
		names = append(names, target.names...)
	}
	for _, fact := range query.Calls.facts.Nodes {
		names = append(names, fact.Value.Callables.Names()...)
	}

	return names
}
