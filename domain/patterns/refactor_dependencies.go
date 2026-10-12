package patterns

import (
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

// RefactorDependencies reports the facts used to authorize a whole-body rewrite.
// It borrows the graph; only small counts and gap descriptions are retained.
type RefactorDependencies struct {
	FunctionID           string   `json:"function_id"`
	Status               string   `json:"status"`
	Bindings             int      `json:"bindings"`
	Definitions          int      `json:"definitions"`
	MemoryLocations      int      `json:"memory_locations"`
	ExecutionConstraints int      `json:"execution_constraints"`
	Missing              []string `json:"missing_or_approximate,omitempty"`
}

// Distinct gaps are collected once per function, even with thousands of reads.
// The traversal indexes and collector are released after verification.
type dependencyAudit struct {
	graph    *pdg.Graph
	reaching map[reachingAccess]bool
	gaps     map[string]bool
}

func (a *dependencyAudit) note(failed bool, reason string) {
	if failed {
		a.gaps[reason] = true
	}
}

func (a *dependencyAudit) sortedGaps() []string {
	gaps := make([]string, 0, len(a.gaps))
	for gap := range a.gaps {
		gaps = append(gaps, gap)
	}
	slices.Sort(gaps)
	return gaps
}

func (a *dependencyAudit) bindings() {
	for _, symbol := range a.graph.Symbols {
		a.note(symbol.Type.Status != pdg.Known, "binding type is unknown")
		a.note(nonlocalBinding(symbol.Role), "non-local binding effects require a dedicated rewrite proof")
	}
	for _, input := range a.graph.Function.Inputs {
		a.note(input.Symbol == 0, "function input has no binding identity")
	}
}

func (a *dependencyAudit) node(ref pdg.Ref, node pdg.Node) {
	attributes := a.graph.NodeAttributes(node)
	for _, access := range attributes.Reads {
		a.read(ref, access)
	}
	for _, access := range attributes.Writes {
		a.write(attributes.Defines, access)
	}
	a.note(unresolvedRefactorCall(node, attributes), "call target is unresolved or indirect")
	a.note(!preciseEvidence(a.graph, node.Evidence), "operation evidence is missing or approximate")
	for _, definition := range attributes.Defines {
		a.note(a.graph.Definitions[definition-1].Node != ref, "definition does not belong to its defining operation")
	}
}

func (a *dependencyAudit) read(node pdg.Ref, access pdg.Access) {
	if unresolvedAccess(access) {
		a.note(true, "read binding or memory target is unresolved")
		return
	}
	a.note(!qualifiedRead(node, access, a.reaching), "read has no definition-qualified reaching dependency")
}

func (a *dependencyAudit) write(definitions []pdg.Ref, access pdg.Access) {
	if unresolvedAccess(access) {
		a.note(true, "write binding or memory target is unresolved")
		return
	}
	a.note(!writeHasDefinition(a.graph, definitions, access), "write has no matching owned definition")
}

func (a *dependencyAudit) edge(edge pdg.Edge) {
	a.note(!preciseEvidence(a.graph, edge.Evidence), "dependency edge evidence is missing or approximate")
	if edge.Definition != 0 {
		a.note(!definitionEdgeAgrees(a.graph, edge), "dependency binding disagrees with its source definition")
	}
}

type reachingAccess struct {
	node, symbol, location pdg.Ref
	position               pdg.Position
}

// CheckRefactorDependencies checks local dependency invariants and producer
// coverage. Complete is a prerequisite, not proof that a transformation is safe.
func CheckRefactorDependencies(graph *pdg.Graph) RefactorDependencies {
	if err := pdg.Validate(graph); err != nil {
		return RefactorDependencies{Status: "unavailable", Missing: []string{err.Error()}}
	}
	gaps := slices.Concat(refactorCapabilityGaps(graph), refactorFactGaps(graph))
	status := "complete"
	if len(gaps) != 0 {
		status = "partial"
	}
	return RefactorDependencies{FunctionID: graph.Tables.Text(graph.Function.ID), Status: status,
		Bindings: len(graph.Symbols), Definitions: len(graph.Definitions), MemoryLocations: len(graph.Locations),
		ExecutionConstraints: executionConstraintCount(graph), Missing: gaps}
}

func refactorCapabilityGaps(graph *pdg.Graph) []string {
	var gaps []string
	for _, name := range refactorRequirements(graph) {
		status := graphCapability(graph, name)
		if status != pdg.Known && status != pdg.Supported {
			gaps = append(gaps, name+": "+coverageStatusName(status))
		}
	}
	return gaps
}

func nonlocalBinding(role pdg.Role) bool {
	return role == pdg.GlobalRole || role == pdg.CaptureRole || role == pdg.OtherRole
}

func refactorRequirements(graph *pdg.Graph) []string {
	required := slices.Concat(
		[]string{"node_extraction", "parameters", "scalar_reaching_definitions", "returns", "execution_dependencies"},
		nodeRefactorRequirements(graph), sourceFeatureRequirements(graph), memoryRequirements(graph),
	)
	slices.Sort(required)
	return slices.Compact(required)
}

func nodeRefactorRequirements(graph *pdg.Graph) []string {
	var required []string
	for _, node := range graph.Nodes {
		if name := nodeRefactorCapability(node.Category); name != "" {
			required = append(required, name)
		}
	}
	return required
}

func nodeRefactorCapability(category pdg.Category) string {
	switch category {
	case pdg.Call:
		return "calls"
	case pdg.Control:
		return "control_dependencies"
	default:
		return ""
	}
}

func refactorFactGaps(graph *pdg.Graph) []string {
	audit := dependencyAudit{graph: graph, reaching: reachingAccesses(graph), gaps: make(map[string]bool)}
	audit.bindings()
	for i, node := range graph.Nodes {
		audit.node(pdg.Ref(i+1), node)
	}
	for _, edge := range graph.Edges {
		audit.edge(edge)
	}
	for _, location := range graph.Locations {
		audit.note(!preciseEvidence(graph, location.Evidence), "memory location evidence is missing or approximate")
	}
	audit.note(!acyclicExecution(graph), "execution constraints contain a cycle")
	return audit.sortedGaps()
}

func unresolvedRefactorCall(node pdg.Node, attributes pdg.Attributes) bool {
	if node.Category != pdg.Call {
		return false
	}
	return attributes.CallForm != pdg.DirectCall || attributes.Callee.Status != pdg.Known
}

func writeHasDefinition(graph *pdg.Graph, definitions []pdg.Ref, access pdg.Access) bool {
	for _, ref := range definitions {
		definition := graph.Definitions[ref-1]
		if definition.Symbol == access.Symbol && definition.Location == access.Location {
			return true
		}
	}
	return false
}

func unresolvedAccess(access pdg.Access) bool {
	return access.UnresolvedReason != 0 || access.Symbol == 0 && access.Location == 0
}

func qualifiedRead(node pdg.Ref, access pdg.Access, reaching map[reachingAccess]bool) bool {
	if access.Symbol != 0 && !reaching[reachingAccess{node: node, symbol: access.Symbol, position: access.Position}] {
		return false
	}
	return access.Location == 0 || reaching[reachingAccess{node: node, location: access.Location, position: access.Position}]
}

func reachingAccesses(graph *pdg.Graph) map[reachingAccess]bool {
	accesses := make(map[reachingAccess]bool)
	for _, edge := range graph.Edges {
		if edge.Kind != pdg.Data {
			continue
		}
		indexReachingAccess(accesses, edge)
	}
	return accesses
}

func indexReachingAccess(accesses map[reachingAccess]bool, edge pdg.Edge) {
	if edge.Symbol != 0 && edge.Definition != 0 {
		accesses[reachingAccess{node: edge.Target, symbol: edge.Symbol, position: edge.Position}] = true
	}
	if edge.Location != 0 {
		accesses[reachingAccess{node: edge.Target, location: edge.Location, position: edge.Position}] = true
	}
}

func definitionEdgeAgrees(graph *pdg.Graph, edge pdg.Edge) bool {
	definition := graph.Definitions[edge.Definition-1]
	return edge.Kind == pdg.Data && definition.Node == edge.Source && definition.Symbol == edge.Symbol && definition.Location == edge.Location
}

func preciseEvidence(graph *pdg.Graph, ref pdg.Ref) bool {
	status := evidenceStatus(graph, ref)
	return status == pdg.Known || status == pdg.Supported
}

func executionConstraintCount(graph *pdg.Graph) int {
	count := 0
	for _, edge := range graph.Edges {
		if edge.Kind == pdg.Execution {
			count++
		}
	}
	return count
}
