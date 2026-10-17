package gopatterns

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
	"golang.org/x/tools/go/packages"
)

type resourceFunction struct {
	object       types.Object
	graph        *pdg.Graph
	query        *resourceflow.Query
	helperSymbol pdg.Ref
	errorSecond  bool
}
type packageResourceFacts struct {
	sources, failures, helpers map[types.Object]bool
	exits                      callflow.ExitSummaries
}

func summarizeResources(functions map[string]resourceFunction) (packageResourceFacts, error) {
	exits, err := resourceExits(functions)
	if err != nil {
		return packageResourceFacts{}, err
	}
	helpers := cleanupSummaries(functions, exits)
	summaries := resourceFactories(functions, helpers)
	facts := packageResourceFacts{exits: exits, sources: map[types.Object]bool{}, failures: map[types.Object]bool{}, helpers: map[types.Object]bool{}}
	for identity, function := range functions {
		summary := summaries[identity]
		facts.sources[function.object] = summary.Fresh
		facts.failures[function.object] = summary.Fresh && summary.FailureEmpty && function.errorSecond
		facts.helpers[function.object] = helpers[identity]
	}
	return facts, nil
}
func resourceFunctions(pkg *packages.Package, graphs []functionGraph) (map[string]resourceFunction, error) {
	functions := make(map[string]resourceFunction, len(graphs))
	for _, source := range graphs {
		function, err := resourceFunctionQuery(pkg, source)
		if err != nil {
			return nil, err
		}
		functions[source.graph.Tables.Text(source.graph.Function.ID)] = function
	}
	return functions, nil
}
func resourceFunctionQuery(pkg *packages.Package, source functionGraph) (resourceFunction, error) {
	fn, graph := source.declaration, source.graph
	object := pkg.TypesInfo.Defs[fn.Name].(*types.Func)
	signature := object.Type().(*types.Signature)
	query, err := resourceflow.New(graph)
	if err != nil {
		return resourceFunction{}, err
	}
	function := resourceFunction{object: object, graph: graph, query: query, errorSecond: factoryErrorSecond(signature)}
	if releaseHelperShape(pkg.TypesInfo, fn) {
		function.helperSymbol = graph.Function.Inputs[0].Symbol
	}
	return function, nil
}
func cleanupSummaries(functions map[string]resourceFunction, exits callflow.ExitSummaries) map[string]bool {
	helpers := map[string]bool{}
	for changed := true; changed; {
		changed = false
		for identity, function := range functions {
			if helpers[identity] || function.helperSymbol == 0 {
				continue
			}
			if !function.query.FromEntry(resourceflow.Policy{Resource: function.helperSymbol, Helpers: helpers, Exits: exits}) {
				continue
			}
			helpers[identity] = true
			changed = true
		}
	}
	return helpers
}

// Syntax selects a binding/report boundary. Dependency and wrapper summaries
// are canonical graph queries; this function does not walk reads or returns.
func sourceAcquisition(info *types.Info, node ast.Node) *ast.CallExpr {
	targets, values := acquisitionSyntax(node)
	if len(targets) == 0 || len(values) != 1 || !localResourceTarget(info, targets[0]) {
		return nil
	}
	call, ok := ast.Unparen(values[0]).(*ast.CallExpr)
	if !ok {
		return nil
	}
	return call
}
func acquisitionSyntax(node ast.Node) ([]ast.Expr, []ast.Expr) {
	switch node := node.(type) {
	case *ast.AssignStmt:
		return node.Lhs, node.Rhs
	case *ast.ValueSpec:
		return memoryDeclaredTargets(node.Names), node.Values
	default:
		return nil, nil
	}
}
func localResourceTarget(info *types.Info, expr ast.Expr) bool {
	identifier, ok := ast.Unparen(expr).(*ast.Ident)
	if !ok || identifier.Name == "_" {
		return false
	}
	variable, ok := info.ObjectOf(identifier).(*types.Var)
	if !ok || variable.Pkg() == nil {
		return false
	}
	return variable.Parent() != variable.Pkg().Scope()
}
func releaseHelperShape(info *types.Info, fn *ast.FuncDecl) bool {
	if fn.Body == nil || fn.Recv != nil {
		return false
	}
	signature := info.Defs[fn.Name].Type().(*types.Signature)
	return signature.Params().Len() == 1 && !signature.Variadic()
}

func factoryErrorSecond(signature *types.Signature) bool {
	return signature.Results().Len() > 1 && isErrorTypeCheck(signature.Results().At(1).Type())
}

func resourceExits(functions map[string]resourceFunction) (callflow.ExitSummaries, error) {
	graphs := map[string]*pdg.Graph{}
	for identity, function := range functions {
		graphs[identity] = function.graph
	}
	return callflow.ExitEffects(graphs)
}
func resourceFactories(functions map[string]resourceFunction, helpers map[string]bool) map[string]resourceflow.FactorySummary {
	queries := make(map[string]*resourceflow.Query, len(functions))
	for identity, function := range functions {
		queries[identity] = function.query
	}
	return resourceflow.SummarizeFactories(queries, helpers)
}
