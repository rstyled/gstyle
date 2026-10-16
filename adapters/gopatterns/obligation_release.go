package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/patterns"
	"golang.org/x/tools/go/cfg"
	"golang.org/x/tools/go/packages"
)

// releaseProof follows normal paths only. Unknown calls, aliases and helper
// chains are not release evidence. Exceptional termination is outside this rule.
type releaseProof struct {
	info         *types.Info
	resource     types.Object
	acquireError types.Object
	allowReturn  bool
	helpers      map[types.Object]bool
	states       map[*cfg.Block]uint8
}

func helperReleases(pkg *packages.Package) map[types.Object]bool {
	helpers := make(map[types.Object]bool)
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && helperReleasesParameter(pkg, fn) {
				helpers[pkg.TypesInfo.Defs[fn.Name]] = true
			}
		}
	}
	return helpers
}

func helperReleasesParameter(pkg *packages.Package, fn *ast.FuncDecl) bool {
	if fn.Body == nil || fn.Recv != nil {
		return false
	}
	signature := pkg.TypesInfo.Defs[fn.Name].Type().(*types.Signature)
	if signature.Params().Len() != 1 || signature.Variadic() {
		return false
	}
	proof := releaseProof{info: pkg.TypesInfo, resource: signature.Params().At(0), states: make(map[*cfg.Block]uint8)}
	flow := cfg.New(fn.Body, func(*ast.CallExpr) bool { return true })
	return proof.block(flow.Blocks[0])
}

func (p *releaseProof) block(block *cfg.Block) bool {
	if state := p.states[block]; state != 0 {
		return state == 2
	}
	p.states[block] = 1 // A cycle does not prove eventual release.
	switch p.nodes(block.Nodes) {
	case -1:
		return false
	case 1:
		p.states[block] = 2
		return true
	}
	if !p.successors(block) {
		return false
	}
	p.states[block] = 2
	return true
}

func (p *releaseProof) successors(block *cfg.Block) bool {
	if len(block.Succs) == 0 {
		return false
	}
	for i, next := range block.Succs {
		if p.failedAcquisitionArm(block, i) {
			continue
		}
		if !p.block(next) {
			return false
		}
	}
	return true
}

// nodes returns -1 for lost identity, 0 for pending, and 1 for released.
func (p *releaseProof) nodes(nodes []ast.Node) int {
	for _, node := range nodes {
		if p.invalidates(node) {
			return -1
		}
		if p.discharges(node) {
			return 1
		}
	}
	return 0
}

func (p *releaseProof) discharges(node ast.Node) bool {
	if returned, ok := node.(*ast.ReturnStmt); ok && p.returnsResource(returned) {
		return true
	}
	for _, expr := range releaseExpressions(node) {
		if p.expression(expr) {
			return true
		}
	}
	return false
}

// Returning the acquired handle transfers its close obligation to the caller.
// Helper summaries must still prove Close; returning a parameter is not cleanup.
func (p *releaseProof) returnsResource(statement *ast.ReturnStmt) bool {
	if !p.allowReturn {
		return false
	}
	for _, result := range statement.Results {
		if p.isResource(result) {
			return true
		}
	}
	return false
}

func releaseExpressions(node ast.Node) []ast.Expr {
	switch n := node.(type) {
	case *ast.AssignStmt:
		return n.Rhs
	case *ast.ReturnStmt:
		return n.Results
	case *ast.ExprStmt:
		return []ast.Expr{n.X}
	case *ast.DeferStmt:
		return []ast.Expr{n.Call}
	case ast.Expr:
		return []ast.Expr{n}
	}
	return nil
}

func (p *releaseProof) expression(expr ast.Expr) bool {
	call, ok := ast.Unparen(expr).(*ast.CallExpr)
	if !ok {
		return false
	}
	if p.releaseCall(call) {
		return true
	}
	for _, arg := range call.Args {
		if p.expression(arg) {
			return true
		}
	}
	return false
}

func (p *releaseProof) releaseCall(call *ast.CallExpr) bool {
	if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
		return p.directClose(selector, call.Args)
	}
	ident, ok := call.Fun.(*ast.Ident)
	if !ok || len(call.Args) != 1 {
		return false
	}
	return p.helpers[p.info.Uses[ident]] && p.isResource(call.Args[0])
}

func (p *releaseProof) directClose(selector *ast.SelectorExpr, args []ast.Expr) bool {
	return selector.Sel.Name == "Close" && len(args) == 0 && p.isResource(selector.X)
}

func (p *releaseProof) isResource(expr ast.Expr) bool {
	ident, ok := ast.Unparen(expr).(*ast.Ident)
	return ok && p.info.ObjectOf(ident) == p.resource
}

func (p *releaseProof) invalidates(node ast.Node) bool {
	invalid := false
	ast.Inspect(node, func(n ast.Node) bool {
		if invalid {
			return false
		}
		switch n := n.(type) {
		case *ast.FuncLit, *ast.GoStmt:
			invalid = true
			return false
		case *ast.UnaryExpr:
			invalid = n.Op == token.AND && p.trackedBinding(n.X)
		case *ast.AssignStmt:
			invalid = p.rebinds(n.Lhs)
		}
		return !invalid
	})
	return invalid
}

func (p *releaseProof) rebinds(expressions []ast.Expr) bool {
	for _, expr := range expressions {
		if _, ok := ast.Unparen(expr).(*ast.Ident); !ok {
			return true
		}
		if p.trackedBinding(expr) {
			return true
		}
	}
	return false
}

func (p *releaseProof) trackedBinding(expr ast.Expr) bool {
	ident, ok := ast.Unparen(expr).(*ast.Ident)
	if !ok {
		return false
	}
	obj := p.info.ObjectOf(ident)
	if obj == nil {
		return false
	}
	return obj == p.resource || obj == p.acquireError
}

func (p *releaseProof) failedAcquisitionArm(block *cfg.Block, arm int) bool {
	if len(block.Nodes) == 0 {
		return false
	}
	expr, ok := block.Nodes[len(block.Nodes)-1].(*ast.BinaryExpr)
	if !ok {
		return false
	}
	if p.isAcquisitionError(expr.X) && typedNil(p.info, expr.Y) {
		return failedFileErrorArm(expr.Op, arm)
	}
	return false
}

func failedFileErrorArm(operator token.Token, arm int) bool {
	switch operator {
	case token.NEQ:
		return arm == 0
	case token.EQL:
		return arm == 1
	default:
		return false
	}
}

func (p *releaseProof) isAcquisitionError(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && p.info.ObjectOf(ident) == p.acquireError
}

func typedNil(info *types.Info, expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && info.Uses[ident] == types.Universe.Lookup("nil")
}

// Compute cleanup and caller obligations together, using one CFG per function.
func fileObligationFacts(pkg *packages.Package, fn *ast.FuncDecl, ctx extractCtx) (released []int, sites []patterns.ResourceAcquisition) {
	flow := cfg.New(fn.Body, func(*ast.CallExpr) bool { return true })
	for _, block := range flow.Blocks {
		for i := range block.Nodes {
			line, site := fileNodeFacts(pkg, fn, &cfg.Block{Nodes: block.Nodes[i:], Succs: block.Succs}, ctx)
			if line != 0 {
				released = append(released, line)
			}
			if site != nil {
				sites = append(sites, *site)
			}
		}
	}
	return released, sites
}

func fileNodeFacts(pkg *packages.Package, fn *ast.FuncDecl, tail *cfg.Block, ctx extractCtx) (int, *patterns.ResourceAcquisition) {
	assign, ok := tail.Nodes[0].(*ast.AssignStmt)
	if !ok {
		return returnedFileLine(pkg, tail.Nodes[0]), nil
	}
	call := sourceAssignment(pkg.TypesInfo, assign, ctx.fileSources)
	if call == nil {
		return 0, nil
	}
	proof := releaseProof{info: pkg.TypesInfo, resource: pkg.TypesInfo.ObjectOf(assign.Lhs[0].(*ast.Ident)), acquireError: pkg.TypesInfo.ObjectOf(assign.Lhs[1].(*ast.Ident)), allowReturn: true, helpers: ctx.helpers, states: map[*cfg.Block]uint8{}}
	if proof.block(&cfg.Block{Nodes: tail.Nodes[1:], Succs: tail.Succs}) {
		return pkg.Fset.PositionFor(assign.Pos(), false).Line, nil
	}
	if osFileCall(pkg.TypesInfo, call) || proof.ownershipEscapes(fn) {
		return 0, nil
	}
	return 0, &patterns.ResourceAcquisition{Line: pkg.Fset.PositionFor(call.Pos(), false).Line, Resource: "file", AcquireCall: fileCallObject(pkg.TypesInfo, call).FullName()}
}

// A direct multi-result return forwards the os file and its error unchanged.
func returnedFileLine(pkg *packages.Package, node ast.Node) int {
	statement, ok := node.(*ast.ReturnStmt)
	if !ok || len(statement.Results) != 1 {
		return 0
	}
	call, ok := ast.Unparen(statement.Results[0]).(*ast.CallExpr)
	if !ok || !osFileCall(pkg.TypesInfo, call) {
		return 0
	}
	return pkg.Fset.PositionFor(call.Pos(), false).Line
}
