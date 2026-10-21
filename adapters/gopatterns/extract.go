// Package gopatterns extracts the shared native PDG IR from typed Go source.
// Gstyle's abstraction labels live in interned profile metadata. Pattern mining
// derives its temporary view from this graph through domain/patterns.Run.
package gopatterns

import (
	"context"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/interfacenil"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
	"golang.org/x/tools/go/packages"
)

// FuncPdg is the dependence graph of one function.
type FuncPdg struct {
	ResourceLifecycles []patterns.ResourceLifecycle
	// SuccessReturnOutputSites are zero-based ErrorCheckSites indices followed directly by a nil error return.
	SuccessReturnOutputSites []int
	// DeferredReleaseLines have explicit cleanup on modeled exits, with foreign calls unproved.
	DeferredReleaseLines []int
	// ReleasedFileLines are typed file acquisitions closed or returned on every normal success path.
	ReleasedFileLines   []int
	FileObligationSites []patterns.ResourceAcquisition
	Name                string
	Path                string
	Line                int
	// EndLine is the function's closing line.
	EndLine int
	// SuppressedKinds are pattern kinds suppressed via //lint:ignore or
	// // gstyle-allow on this function.
	SuppressedKinds []string
	Pdg             pdg.Graph
	// AstTypes is the multiset of AST node type names for the function
	// body (see patterns.AstNodeMultiset). Used by the CCGraph Stage 0
	// AST pre-filter.
	AstTypes map[string]int
	// SelfTy is the receiver's named type for methods ("pkg.Type"), empty
	// for free functions. The miner needs it: signature groups require two
	// or more distinct self types, and facts without SelfTy are skipped.
	SelfTy string
	// GuardClauses are inverted conditionals in this function that want
	// to be guard clauses (AST-level finding, not PDG-derived).
	GuardClauses []GuardClauseHit
	// EnumDispatches are enum-value switches that want to be dispatch
	// tables (AST-level finding, not PDG-derived).
	EnumDispatches []EnumDispatchHit
	// TypeSwitches are type switches in this function whose arms all call
	// the same method on the case-bound value (AST-level finding).
	TypeSwitches []TypeSwitchHit
	// Params are the function's primitive-typed parameters. Used for
	// data-clump detection (value object proposals).
	Params []patterns.ParamInfo
	// EntityIdentities are attribute-based equalities in this function
	// that should compare by ID instead.
	EntityIdentities []patterns.EntityIdentityHit
	// MutableIdentities are ID field assignments outside constructors.
	MutableIdentities []patterns.MutableIdentityHit
	// AggregateMods are named struct types whose fields this function
	// mutates. Co-modified types suggest an aggregate boundary.
	AggregateMods []patterns.AggregateModHit
	// DbCalls are direct database calls in this function.
	DbCalls []patterns.DbCallHit
	// FactoryLits are complex struct literals in this function.
	FactoryLits []patterns.FactoryHit
	// SpecRules are boolean business-rule expressions (2+ conditions).
	SpecRules []patterns.SpecificationHit
	// NilErrHits are `if err != nil { return nil }` sites (gostaticanalysis).
	NilErrHits []patterns.NilErrHit
	// ForceTypeAssertHits are unchecked `x.(T)` (gostaticanalysis).
	ForceTypeAssertHits []patterns.ForceTypeAssertHit
	// TypedNilHits are typed-nil comparisons (gostaticanalysis).
	TypedNilHits []patterns.TypedNilHit
	// ErrorCheckSites tracks per-call error checking for deviant_behavior
	// mining (Engler et al., SOSP 2001).
	ErrorCheckSites []patterns.ErrorCheckSite
}

// Extraction is the full result of package analysis: per-function PDGs
// plus package-level type findings (anemic models).
type Extraction struct {
	Sources *pdg.SourceContext
	// Callables retains canonical declaration and lexical closure body graphs.
	// Funcs contains declaration-level diagnostics and source edit candidates.
	Callables []*pdg.Graph
	// PackageNames contains only declaration spellings, not compiler objects.
	// It lets helper extraction avoid collisions with declarations in other files.
	PackageNames      map[string][]string
	DeclarationOrders []patterns.Candidate
	OffsetFallbacks   []patterns.Candidate
	// NilGuards are known-nil dereference findings (detection-only).
	NilGuards       []patterns.Candidate
	NegativeIndices []patterns.Candidate
	LiteralIndices  []patterns.Candidate
	Funcs           []FuncPdg
	// AnemicModels are exported methodless structs with 3+ functions
	// operating on their fields (DDD anemic domain model detection).
	AnemicModels []patterns.AnemicModelHit
	// MissingIdentities are structs used as entities without an ID field.
	MissingIdentities []patterns.MissingIdentityHit
	// DomainServices are free functions operating on the fields of two
	// or more struct types: stateless domain services (Evans), not
	// anemic-model violations.
	DomainServices []patterns.DomainServiceHit
}

// extractCtx carries per-package maps for entity pairing.
type extractCtx struct {
	resourceFunctions map[types.Object]resourceFunction
	protocols         []protocolFacts
	resourceQueries   map[*pdg.Graph]*resourceflow.Query
	exits             callflow.ExitSummaries
	fileSources       map[types.Object]bool
	fileFailures      map[types.Object]bool
	helpers           map[types.Object]bool
	structLocs        map[string]structLoc
	mutated           map[string]bool
}

// newExtractCtx builds the struct location and mutation maps for pairing
// missing_identity with entity_identity.
func newExtractCtx(pkg *packages.Package, root string, resources packageResourceFacts) extractCtx {
	structLocs := map[string]structLoc{}
	for _, sd := range collectStructs(pkg, root) {
		if _, ok := structLocs[sd.name]; !ok {
			structLocs[sd.name] = structLoc{path: sd.path, line: sd.line}
		}
	}
	return extractCtx{
		resourceFunctions: resources.functions,
		protocols:         resources.protocols,
		fileSources:       resources.sources,
		exits:             resources.exits,
		fileFailures:      resources.failures,
		helpers:           resources.helpers,
		structLocs:        structLocs,
		mutated:           collectMutatedStructs(pkg),
	}
}

// loopCtx records an enclosing iteration for the loop-element
// canonicalization: coll[counter] resolves to the Iterate node, so
// `for i ... { f(names[i]) }` and `for _, n := range names { f(n) }`
// produce the same shape.
type loopCtx struct {
	counter types.Object // loop counter variable (nil for range loops)
	coll    int          // node index of the iterated collection
	iterate int          // Iterate node index
}

// builder lowers one function body to a PDG. Bindings are flow-insensitive
// (first binding wins, like rstyle): a variable use resolves to its
// declaration/parameter/loop-variable node.
type builder struct {
	calls     map[*ast.CallExpr]int
	info      *types.Info
	fset      *token.FileSet
	ir        *irBuilder
	origin    ast.Node
	parameter int
	binds     map[types.Object]int
	loops     []loopCtx
}

func (b *builder) line(pos token.Pos) int {
	return b.fset.PositionFor(pos, false).Line
}

// node records an operation. Control dependencies are computed from the CFG.
func (b *builder) node(n patterns.PdgNode) int {
	return b.matchingNode(n)
}

func (b *builder) data(from, to, pos int) {
	if from == none {
		return
	}
	b.ir.miningValueEdge(from, to, pos)
}

// bind records the node a variable resolves to. First binding wins:
// reassignments do not move the binding (flow-insensitive, like rstyle).
func (b *builder) bind(obj types.Object, node int) {
	if obj == nil || node == none {
		return
	}
	if _, ok := b.binds[obj]; !ok {
		b.binds[obj] = node
	}
}

func (b *builder) lookup(obj types.Object) (int, bool) {
	n, ok := b.binds[obj]
	return n, ok
}

func (b *builder) params(fn *ast.FuncDecl) {
	add := func(fields *ast.FieldList) {
		if fields == nil {
			return
		}
		for _, field := range fields.List {
			typ := patterns.TypeClass(b.info.TypeOf(field.Type))
			for _, name := range field.Names {
				obj := b.info.Defs[name]
				n := b.node(patterns.PdgNode{Kind: patterns.Param, TyClass: typ, Line: b.line(name.Pos())})
				b.bind(obj, n)
			}
			if len(field.Names) == 0 {
				b.node(patterns.PdgNode{Kind: patterns.Param, TyClass: typ, Line: b.line(field.Pos())})
			}
		}
	}
	add(fn.Recv)
	add(fn.Type.Params)
}

// none is returned by expr when an expression produces no PDG node
// (a variable use resolved through binds, blank, or unresolvable).
const none = -1

type sourceLoadOptions struct {
	disk      *pdg.SourceContext
	overlay   map[string][]byte
	snapshots *pdg.SourceContext
	tests     bool
}

// SourceOverlay holds a simulated source revision and its original disk evidence.
type SourceOverlay struct {
	Files map[string][]byte
	Disk  *pdg.SourceContext
}

// Extract loads the packages enclosing paths and returns PDGs per
// function plus anemic-model hits. Paths are directories (".", "./...")
// or .go files, resolved under root.
func Extract(ctx context.Context, root string, paths []string) (*Extraction, error) {
	return extractWithSources(ctx, root, paths, sourceLoadOptions{})
}

// ExtractForFix includes test callers in the shared typed source inventory.
func ExtractForFix(ctx context.Context, root string, paths []string) (*Extraction, error) {
	return extractWithSources(ctx, root, paths, sourceLoadOptions{tests: true})
}

// ExtractFixOverlay builds fresh PDG facts for a preview revision without writes.
func ExtractFixOverlay(ctx context.Context, root string, paths []string, revision SourceOverlay) (*Extraction, error) {
	return extractWithSources(ctx, root, paths, sourceLoadOptions{tests: true, overlay: revision.Files, disk: revision.Disk})
}

func extractWithSources(ctx context.Context, root string, paths []string, options sourceLoadOptions) (*Extraction, error) {
	abs, err := absRoot(root)
	if err != nil {
		return nil, err
	}
	query, err := patternsQuery(abs, paths)
	if err != nil {
		return nil, err
	}
	sources := pdg.NewSourceContext(abs, query)
	sources.Disk, sources.Overlay = options.disk, options.overlay
	options.snapshots = sources
	if options.tests {
		return extractBoundedFix(ctx, abs, paths, options)
	}
	return extractNormalPackages(ctx, abs, paths, options)
}

func extractNormalPackages(ctx context.Context, abs string, paths []string, options sourceLoadOptions) (*Extraction, error) {
	sources, query := options.snapshots, options.snapshots.Queries
	pkgs, sourceIdentities, err := loadPatternSourcesForFix(ctx, abs, query, options)
	if err != nil {
		return nil, err
	}
	collectSourceReferences(sources, productionPatternPackages(pkgs))
	var funcsOut []FuncPdg
	var callables []*pdg.Graph
	var orders []patterns.Candidate
	var ranges []patterns.Candidate
	var anemic []patterns.AnemicModelHit
	var missing []patterns.MissingIdentityHit
	var services []patterns.DomainServiceHit
	pool := pdg.NewBuilder()
	pool.Tables.Sources = sources
	internSourceMetadata(pool)
	program := callflow.NewProgram()
	for _, pkg := range orderedSourcePackages(productionPatternPackages(pkgs)) {
		// Validation visits shared tables; bound them to this package.
		packagePool := pdg.NewBuilder()
		packagePool.Tables.Sources = sources
		analysis := packageAnalysis{ctx: ctx, pkg: pkg, root: abs, pool: packagePool, sources: sourceIdentities, program: program}
		funcs, err := analysis.run()
		if err != nil {
			return nil, err
		}
		orders = append(orders, findDeclarationOrders(pkg, abs, paths, sources)...)
		ranges = append(ranges, findOffsetFallbacks(pkg, abs, paths, sources)...)
		projectSourceParameterFlow(sources, funcs)
		funcsOut = append(funcsOut, funcs...)
		for _, source := range analysis.graphs {
			callables = append(callables, source.graph)
		}
		anemic = append(anemic, findAnemicModels(pkg, abs)...)
		missing = append(missing, findMissingIdentities(pkg, abs)...)
		services = append(services, findDomainServices(pkg, abs)...)
	}
	populateInterfaceNilHits(funcsOut)
	result := &Extraction{Sources: sources, Callables: callables, PackageNames: packageBindingNames(pkgs), Funcs: funcsOut, DeclarationOrders: orders, OffsetFallbacks: ranges, AnemicModels: anemic, MissingIdentities: missing, DomainServices: services}
	if err := populateRuntimeGuards(result, abs, paths); err != nil {
		return nil, err
	}
	return result, nil
}

func packageBindingNames(pkgs []*packages.Package) map[string][]string {
	names := make(map[string][]string)
	for _, pkg := range pkgs {
		names[pkg.PkgPath] = ownedBindingNames(pkg.Types.Scope().Names())
	}
	return names
}

func ownedBindingNames(names []string) []string {
	for i := range names {
		names[i] = strings.Clone(names[i])
	}
	return names
}

func absRoot(root string) (string, error) {
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return cwd, nil
	}
	return filepath.Abs(root)
}

func loadPatternPackages(ctx context.Context, dir string, query []string) ([]*packages.Package, error) {
	pkgs, _, err := loadPatternSources(ctx, dir, query)
	return pkgs, err
}

func loadPatternSources(ctx context.Context, dir string, query []string) ([]*packages.Package, map[string]string, error) {
	return loadPatternSourcesInto(ctx, dir, query, nil)
}

func loadPatternSourcesInto(ctx context.Context, dir string, query []string, snapshots *pdg.SourceContext) ([]*packages.Package, map[string]string, error) {
	return loadPatternSourcesForFix(ctx, dir, query, sourceLoadOptions{snapshots: snapshots})
}

func loadPatternSourcesForFix(ctx context.Context, dir string, query []string, options sourceLoadOptions) ([]*packages.Package, map[string]string, error) {
	if err := captureFixConfiguration(options.snapshots, options); err != nil {
		return nil, nil, err
	}
	sources := make(map[string]string)
	config := &packages.Config{Context: ctx, Dir: dir, Mode: packages.LoadSyntax, Tests: options.tests, ParseFile: patternSourceParser(sources, options.snapshots)}
	pkgs, err := loadSourcePackages(config, query, options.overlay)
	if err != nil {
		return nil, nil, fmt.Errorf("load Go packages: %w", err)
	}
	if err := patternPackageErrors(pkgs, dir); err != nil {
		return nil, nil, err
	}
	return pkgs, sources, nil
}

func patternPackageErrors(pkgs []*packages.Package, dir string) error {
	var errs []string
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			errs = append(errs, strings.ReplaceAll(e.Error(), dir+string(filepath.Separator), ""))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("Go analysis fails:\n%s", strings.Join(errs, "\n"))
	}
	return nil
}

func packagePdgs(pkg *packages.Package, root string, pool *pdg.Builder) ([]FuncPdg, error) {
	return packagePdgsWithSources(pkg, root, pool, nil)
}

func packagePdgsWithSources(pkg *packages.Package, root string, pool *pdg.Builder, sources map[string]string) ([]FuncPdg, error) {
	analysis := packageAnalysis{pkg: pkg, root: root, pool: pool, sources: sources}
	return analysis.run()
}

func (analysis *packageAnalysis) run() ([]FuncPdg, error) {
	if err := analysis.build(); err != nil {
		return nil, err
	}
	if err := analysis.summarize(); err != nil {
		return nil, err
	}
	if err := analysis.analyze(); err != nil {
		return nil, err
	}
	return analysis.result, nil
}

func patternsQuery(root string, paths []string) ([]string, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	query := []string{}
	for _, p := range paths {
		q, err := patternInput(root, p)
		if err != nil {
			return nil, err
		}
		query = append(query, q)
	}
	return query, nil
}

func patternInput(root, input string) (string, error) {
	if strings.Contains(input, "...") {
		return "", fmt.Errorf("use directory or .go file paths, not package patterns: %q", input)
	}
	abs := input
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, input)
	}
	abs = filepath.Clean(abs)
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("inspect input %q: %w", input, err)
	}
	if info.IsDir() {
		return filepath.Join(abs, "..."), nil
	}
	if filepath.Ext(abs) != ".go" {
		return "", fmt.Errorf("input must be a directory or Go file: %q", input)
	}
	return "file=" + abs, nil
}

// SigKeyOf returns the normalized signature key for signature mining:
// "fn(" + parameter type classes + ")". Functions with the same shape of
// signature block together in sigmine; the receiver counts as a parameter
// but normalizes away for user-defined types, so methods on different
// types can still group.
func SigKeyOf(fp FuncPdg) string {
	var params []string
	for _, node := range fp.Pdg.Nodes {
		if node.MatchLabel == 0 {
			continue
		}
		label := fp.Pdg.Tables.MatchLabels[node.MatchLabel-1]
		if fp.Pdg.Tables.Text(label.Kind) == string(patterns.Param) {
			params = append(params, fp.Pdg.Tables.Text(label.TypeClass))
		}
	}
	return "fn(" + strings.Join(params, ",") + ")"
}

func extractFunc(pkg *packages.Package, source functionGraph, ctx extractCtx) (FuncPdg, error) {
	fn, path, graph := source.declaration, source.path, source.graph
	name := fn.Name.Name
	selfTy := ""
	if object, ok := pkg.TypesInfo.Defs[fn.Name].(*types.Func); ok {
		name = patterns.FuncID(object)
		selfTy = patterns.ReceiverType(object)
	}
	pos := pkg.Fset.PositionFor(fn.Pos(), false)
	end := pkg.Fset.PositionFor(fn.End(), false)
	sites, successReturnSites, err := reachableErrorCheckFacts(fn, pkg.Fset, pkg.TypesInfo, graph)
	if err != nil {
		return FuncPdg{}, err
	}
	releasedFiles, fileSites, deferredLines, err := resourceObligationFacts(pkg, fn, ctx, graph)
	if err != nil {
		return FuncPdg{}, err
	}
	lifecycles := resourceLifecycleFacts(pkg, fn, ctx, graph)
	var bindings *bindingflow.Query
	if query := ctx.resourceQueries[graph]; query != nil {
		bindings = query.CallQueries().Bindings
	}
	return FuncPdg{
		ResourceLifecycles: lifecycles,
		Name:               name,
		Path:               path,
		Line:               pos.Line,
		EndLine:            end.Line,
		SuppressedKinds:    findSuppressedKinds(fn),
		Pdg:                *graph,
		AstTypes:           patterns.AstNodeMultiset(fn),
		SelfTy:             selfTy,
		GuardClauses:       findGuardClauses(fn, pkg.Fset),
		EnumDispatches:     findEnumDispatches(fn, pkg.Fset, pkg.TypesInfo),
		TypeSwitches:       findTypeSwitches(fn, pkg.Fset, pkg.TypesInfo),
		Params:             primitiveParams(fn, pkg.TypesInfo),
		// EntityIdentities suggest reviewing attribute-based equality.
		// Existing and missing IDs both require a domain invariant before edits.
		EntityIdentities: append(
			findEntityIdentities(fn, pkg.Fset, pkg.TypesInfo),
			findEntityIdentitiesNoID(fn, noIDScanCtx{
				fset:       pkg.Fset,
				info:       pkg.TypesInfo,
				structLocs: ctx.structLocs,
				mutated:    ctx.mutated,
			})...,
		),
		// MutableIdentities are ID assignments outside constructors.
		MutableIdentities: FindMutableIdentities(fn, pkg.Fset),
		// AggregateMods are struct types whose fields this function mutates.
		AggregateMods: FindAggregateMods(fn, pkg.TypesInfo),
		// DbCalls are direct database calls (skipped in repository files).
		DbCalls: FindDbCalls(fn, pkg.Fset, pkg.TypesInfo, path),
		// FactoryLits are complex struct literals.
		FactoryLits: findFactoryLits(fn, pkg.Fset, pkg.TypesInfo, pkg.Types),
		// SpecRules are boolean business-rule expressions (2+ conditions).
		SpecRules: findSpecificationHits(fn, pkg.Fset, pkg.TypesInfo),
		// NilErrHits are `if err != nil { return nil }` (gostaticanalysis).
		NilErrHits: findNilErrGraphHits(fn, pkg.Fset, pkg.TypesInfo, graph, bindings),
		// ForceTypeAssertHits are unchecked `x.(T)` (gostaticanalysis).
		ForceTypeAssertHits: findForceTypeAssertHits(fn, pkg.Fset, guardedTypeAssertions(pkg, source, bindings)),
		// TypedNilHits are typed-nil comparisons (gostaticanalysis).
		TypedNilHits: nil,
		// ErrorCheckSites tracks per-call error checking for deviant_behavior.
		ErrorCheckSites:     sites,
		ReleasedFileLines:   releasedFiles,
		FileObligationSites: fileSites,

		DeferredReleaseLines:     deferredLines,
		SuccessReturnOutputSites: successReturnSites,
	}, nil
}

// primitiveParams returns the function's basic-typed parameters as
// (name, type) pairs. Named types are excluded: they are already domain
// types and need no value-object proposal.
func primitiveParams(fn *ast.FuncDecl, info *types.Info) []patterns.ParamInfo {
	if fn.Type.Params == nil {
		return nil
	}
	var out []patterns.ParamInfo
	for _, field := range fn.Type.Params.List {
		basic, ok := info.TypeOf(field.Type).(*types.Basic)
		if !ok {
			continue
		}
		for _, name := range field.Names {
			out = append(out, patterns.ParamInfo{Name: name.Name, Type: basic.Name()})
		}
	}
	return out
}

func pdgSourceIdentity(pool *pdg.Builder, digest string) pdg.Fact {
	if digest == "" {
		return pool.Missing(pdg.Unknown, "Compiler source snapshot was not captured.")
	}
	return pool.Known(digest)
}

// Findings derive from the same graphs exported and consumed by matching.
func populateInterfaceNilHits(functions []FuncPdg) {
	graphs := make([]pdg.Graph, len(functions))
	for i := range functions {
		graphs[i] = functions[i].Pdg
	}
	for i, hits := range interfacenil.Find(graphs) {
		for _, ref := range hits {
			functions[i].TypedNilHits = append(functions[i].TypedNilHits, patterns.TypedNilHit{Line: int(graphs[i].Nodes[ref-1].Line), ExprText: "interface value is provably non-nil"})
		}
	}
}

func patternSourceParser(sources map[string]string, snapshots *pdg.SourceContext) func(*token.FileSet, string, []byte) (*ast.File, error) {
	var lock sync.Mutex
	return func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
		content, err := patternSourceBytes(filename, src)
		if err != nil {
			return nil, err
		}
		file, err := parser.ParseFile(fset, filename, content, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		lock.Lock()
		defer lock.Unlock()
		sources[filename] = fmt.Sprintf("%x", sha256.Sum256(content))
		if err := publishSourceSnapshot(snapshots, content, fset, file); err != nil {
			return nil, err
		}
		return file, nil
	}
}

func publishSourceSnapshot(snapshots *pdg.SourceContext, content []byte, fset *token.FileSet, file *ast.File) error {
	if snapshots == nil {
		return nil
	}
	filename := fset.File(file.Pos()).Name()
	if snapshots.Reference(filename) != 0 {
		return snapshots.Check(filename, content)
	}
	ref := snapshots.AddOwned(filename, content, sourceCommentRanges(fset, file, content), sourceCommentOwners(fset, file, content))
	// Tests retain exact bytes and typed caller ranges for signature fixes. They
	// are not local rewrite targets, so they need no full syntax projection.
	if strings.HasSuffix(filename, "_test.go") {
		return nil
	}
	syntax, err := captureSourceSyntax(fset, file, content)
	if err != nil {
		return err
	}
	return snapshots.SetSyntax(ref, sourceSyntaxProfile, syntax)
}

func testPackageVariant(id string) bool {
	return strings.Contains(id, " [") || strings.HasSuffix(id, ".test")
}

func sourceSnapshotReference(sources *pdg.SourceContext, path string) pdg.Ref {
	if sources == nil {
		return 0
	}
	return sources.Reference(path)
}

func productionPatternPackages(pkgs []*packages.Package) []*packages.Package {
	var production []*packages.Package
	for _, pkg := range pkgs {
		if !testPackageVariant(pkg.ID) {
			production = append(production, pkg)
		}
	}
	return production
}
