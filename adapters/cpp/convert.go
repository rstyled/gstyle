package cpp

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/rstyled/gstyle/domain"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/quality"
)

const Profile = "gstyle.cpp.import"

type Function struct {
	Path    string
	Score   domain.Function
	Graph   *pdg.Graph
	Quality quality.Function
	source  sourcePatterns
	mining  *patterns.MiningFacts
	body    bool
}

type Extraction struct {
	Root          string
	Language      string
	Functions     []Function
	sourceOptions patterns.Options
}

type projection struct {
	doc     *document
	method  *vertex
	pool    *pdg.Builder
	graph   *pdg.Graph
	refs    map[*vertex]pdg.Ref
	attrs   []pdg.Attributes
	code    string
	file    *sourceFile
	symbols map[*vertex]pdg.Ref
	inputs  []*vertex
	types   map[*vertex]string
}

func convert(ctx context.Context, doc *document) (*Extraction, error) {
	result := &Extraction{Root: doc.Metadata.text("ROOT"), Language: doc.Metadata.text("LANGUAGE")}
	groups := methodNodes(doc)
	recursive := recursiveMethods(doc)
	for _, method := range doc.Methods {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		builder := newProjection(doc, method)
		graph, err := builder.convert(groups[method])
		if err != nil {
			return nil, fmt.Errorf("method %s: %w", method.text("FULL_NAME"), err)
		}
		score := scoreMethod(method, recursive[method], builder.file)
		result.Functions = append(result.Functions, Function{Path: builder.path(), Score: score, Graph: graph, Quality: builder.qualityFunction(groups[method], score), source: detectSourcePatterns(method, groups[method]), mining: builder.miningFacts(groups[method]), body: methodBody(method)})
	}
	result.sourceOptions = doc.patternOptions(groups)
	return result, nil
}

func methodNodes(doc *document) map[*vertex][]*vertex {
	groups := map[*vertex][]*vertex{}
	for _, node := range doc.Nodes {
		if node.Owner != nil {
			groups[node.Owner] = append(groups[node.Owner], node)
		}
	}
	return groups
}

func newProjection(doc *document, method *vertex) *projection {
	pool := pdg.NewBuilder()
	return &projection{doc: doc, method: method, pool: pool, graph: &pdg.Graph{Tables: pool.Tables}, refs: map[*vertex]pdg.Ref{}, symbols: map[*vertex]pdg.Ref{}, types: map[*vertex]string{}}
}

func (p *projection) path() string {
	if name := p.method.text("FILENAME"); name != "" {
		return name
	}
	return "<unknown>"
}

func (p *projection) convert(nodes []*vertex) (*pdg.Graph, error) {
	p.header()
	for _, node := range nodes {
		p.node(node)
	}
	p.callFacts(nodes)
	p.bindings(nodes)
	p.referenceCount(nodes)
	p.captureFacts(nodes)
	p.sortInputs()
	for _, node := range nodes {
		p.operands(node)
	}
	p.dependencies()
	if err := p.operationFlow(); err != nil {
		return nil, err
	}
	p.capabilities()
	p.recordRelations()
	p.miningLabels(nodes)
	return p.finish(nodes)
}

func (p *projection) finish(nodes []*vertex) (*pdg.Graph, error) {
	if err := pdg.PackAttributes(p.graph, p.attrs); err != nil {
		return nil, err
	}
	if err := p.certifyImport(nodes); err != nil {
		return nil, err
	}
	return p.graph, nil
}

func (p *projection) sortInputs() {
	parameters := map[*vertex]pdg.Parameter{}
	for i, node := range p.inputs {
		parameters[node] = p.graph.Function.Inputs[i]
	}
	slices.SortFunc(p.inputs, func(a, b *vertex) int { return a.number("INDEX") - b.number("INDEX") })
	for i, node := range p.inputs {
		parameter := parameters[node]
		parameter.Position = uint32(i)
		p.graph.Function.Inputs[i] = parameter
	}
}

func (p *projection) header() {
	b, g := p.pool, p.graph
	g.Profile = b.Policy(Profile, "1")
	g.Producer = pdg.Producer{Name: b.Text("gstyle-cpp"), Version: b.Text("1"), Language: b.Text(p.doc.Metadata.text("LANGUAGE")), LanguageVersion: b.Missing(pdg.Unknown, "not recorded by CPG metadata"), Configuration: b.Text(`{}`)}
	g.Sources = []pdg.Source{{ID: b.Text(p.path()), Path: b.Text(p.path()), ContentIdentity: b.Missing(pdg.Unknown, "source bytes are not verified by GraphSON")}}
	g.Function = pdg.Function{ID: b.Text("cpp:" + strconv.FormatUint(p.method.ID, 10)), Name: b.Known(p.method.text("NAME")), QualifiedName: b.Known(p.method.text("FULL_NAME")), Span: pdg.Span{Source: 1}, Interface: b.Evidence(pdg.Approximated, "imported method interface"), ReferenceCount: pdg.CountFact{Status: pdg.Unknown, Reason: b.Text("reference-count policy not evaluated")}}
	p.source()
	p.extension("gstyle.cpp:method", p.method.Props)
	p.extension("gstyle.cpp:metadata", p.doc.Metadata.Props)
	p.extension("gstyle.cpp:overlays", p.doc.Metadata.Props["OVERLAYS"])
}

func (p *projection) extension(name string, value any) {
	data, _ := json.Marshal(value)
	p.graph.Extensions = append(p.graph.Extensions, pdg.Extension{Name: p.pool.Text(name), Value: p.pool.Text(string(data))})
}

var categories = map[string]pdg.Category{
	"METHOD": pdg.Entry, "METHOD_PARAMETER_IN": pdg.FormalInput,
	"METHOD_PARAMETER_OUT": pdg.FormalOutput, "METHOD_RETURN": pdg.FormalOutput,
	"RETURN": pdg.Return, "IDENTIFIER": pdg.Read, "LOCAL": pdg.Declaration,
	"CONTROL_STRUCTURE": pdg.Control, "LITERAL": pdg.Computation, "CALL": pdg.Call,
}

func category(node *vertex) pdg.Category {
	if macroCall(node) {
		return pdg.Computation
	}
	if voidReturn(node) {
		return pdg.Exit
	}
	if runtimeCall(node) {
		return pdg.Call
	}
	if operatorCall(node) {
		return operatorCategory(node.text("NAME"))
	}
	if kind, exists := categories[node.Label]; exists {
		return kind
	}
	return pdg.Other
}

func voidReturn(node *vertex) bool {
	return node.Label == "METHOD_RETURN" && node.text("TYPE_FULL_NAME") == "void"
}

func operatorCategory(name string) pdg.Category {
	if strings.HasPrefix(name, "<operator>.assignment") || updateOperators[name] {
		return pdg.Assignment
	}
	return pdg.Computation
}

var updateOperators = map[string]bool{"<operator>.postIncrement": true, "<operator>.preIncrement": true, "<operator>.postDecrement": true, "<operator>.preDecrement": true}

func (p *projection) node(node *vertex) {
	b := p.pool
	attrs := p.attributes(node)
	span, present := p.span(node)
	ref := pdg.Ref(len(p.graph.Nodes) + 1)
	p.refs[node] = ref
	p.attrs = append(p.attrs, attrs)
	item := pdg.Node{ID: b.Text(strconv.FormatUint(node.ID, 10)), Category: category(node), OriginalKind: b.Known(node.Label), Line: uint32(max(0, node.number("LINE_NUMBER"))), Attributes: pdg.Ref(len(p.attrs)), Synthetic: !present, Evidence: b.Evidence(pdg.Approximated, "projected CPG source node; source-location extension retains CPG properties"), Provenance: b.Origin(pdg.Projection, "C++ GraphSON node projection")}
	if present {
		p.graph.Spans = append(p.graph.Spans, span)
		item.Span = pdg.Ref(len(p.graph.Spans))
	}
	p.graph.Nodes = append(p.graph.Nodes, item)
	p.parameter(node)
}

func (p *projection) attributes(node *vertex) pdg.Attributes {
	data, _ := json.Marshal(node.Props)
	attrs := pdg.Attributes{Description: p.pool.Text(node.Label), Extensions: p.pool.SingletonExtension("gstyle.cpp:properties", string(data))}
	if macroCall(node) {
		attrs.Construct = p.pool.Text("macro-expansion")
	}
	if node.Label == "LITERAL" {
		attrs.Literal = p.pool.Known(node.text("CODE"))
	}
	if node.Label != "CALL" {
		return attrs
	}
	if category(node) != pdg.Call && strings.HasPrefix(node.text("NAME"), "<operator>.") {
		attrs.Operator = p.pool.Known(node.text("NAME"))
		return attrs
	}
	return attrs
}

func (p *projection) parameter(node *vertex) {
	if voidReturn(node) {
		return
	}
	if node.Label != "METHOD_PARAMETER_IN" && node.Label != "METHOD_RETURN" {
		return
	}
	position := uint32(len(p.graph.Function.Inputs))
	parameter := pdg.Parameter{Name: p.pool.Known(node.text("NAME")), Type: p.pool.Known(node.text("TYPE_FULL_NAME")), Position: position, Role: pdg.ParameterRole, Variadic: pdg.BooleanFact{Status: pdg.Known, Value: node.flag("IS_VARIADIC")}}
	if node.Label == "METHOD_RETURN" {
		parameter.Role, parameter.Position = pdg.ReturnRole, 0
		p.graph.Function.Outputs = append(p.graph.Function.Outputs, parameter)
		return
	}
	p.graph.Function.Inputs = append(p.graph.Function.Inputs, parameter)
	p.inputs = append(p.inputs, node)
}

func (p *projection) operands(node *vertex) {
	if node.Label != "CALL" && node.Label != "RETURN" {
		return
	}
	for _, operand := range node.Children {
		if !valueOperand(node, operand) {
			continue
		}
		p.addEdge(dependency{Source: operand, Target: node, Kind: pdg.Data, Subkind: operandRelation(node, operand), Position: operand.number("ARGUMENT_INDEX"), ID: "operand:" + strconv.FormatUint(operand.ID, 10) + ":" + strconv.FormatUint(node.ID, 10)})
	}
}

func operandRelation(node, operand *vertex) string {
	if node.text("NAME") == "<operator>.assignment" && operand.number("ARGUMENT_INDEX") == 1 {
		return "write-address"
	}
	return "value-operand"
}

func valueOperand(node, operand *vertex) bool {
	if operand.Label == "FIELD_IDENTIFIER" || operand.Label == "TYPE_REF" {
		return false
	}
	return !(node.text("NAME") == "<operator>.assignment" && operand.number("ARGUMENT_INDEX") == 1 && operand.Label == "IDENTIFIER")
}

func (p *projection) dependencies() {
	var cfg, cross []map[string]any
	for _, link := range p.doc.MethodEdges[p.method] {
		if link.Label == "CFG" {
			cfg = append(cfg, sourceLink(link))
		}
		kind, exists := map[string]pdg.EdgeKind{"CDG": pdg.ControlEdge, "REACHING_DEF": pdg.Data}[link.Label]
		if !exists {
			continue
		}
		if externalDependency(link, p.method) {
			cross = append(cross, sourceLink(link))
			continue
		}
		p.addEdge(dependency{Source: link.Source, Target: link.Target, Kind: kind, Subkind: link.Label, Position: -1, Variable: link.Variable, Properties: link.Properties, ID: "cpp:e" + strconv.FormatUint(link.ID, 10)})
	}
	p.extension("gstyle.cpp:cfg", cfg)
	p.extension("gstyle.cpp:cross-method-dependencies", cross)
}

func externalDependency(link edge, method *vertex) bool {
	return link.Source.Owner != method || link.Target.Owner != method
}

func sourceLink(link edge) map[string]any {
	return map[string]any{"id": link.ID, "source": link.Source.ID, "target": link.Target.ID, "kind": link.Label, "variable": link.Variable, "properties": link.Properties}
}

type dependency struct {
	Source, Target        *vertex
	Kind                  pdg.EdgeKind
	ID, Subkind, Variable string
	Properties            string
	Position              int
}

func (p *projection) addEdge(input dependency) {
	from, to := p.refs[input.Source], p.refs[input.Target]
	if from == 0 || to == 0 {
		return
	}
	link := pdg.Edge{ID: p.pool.Text(input.ID), Source: from, Target: to, Kind: input.Kind, Subkind: p.pool.Text(input.Subkind), Evidence: p.pool.Evidence(pdg.Approximated, "imported C++ dependencies; alias, call and exceptional precision not certified"), Provenance: p.pool.Origin(pdg.Projection, "C++ relation projection")}
	if input.Position >= 0 {
		link.Position = pdg.Position(input.Position + 1)
	}
	if input.Variable != "" {
		link.Annotations = p.annotation("gstyle.cpp:variable", input.Variable)
	}
	if input.Properties != "" {
		link.Annotations = p.edgeProperties(link.Annotations, input.Properties)
	}
	p.reachingIdentity(&link, input.Variable)
	p.graph.Edges = append(p.graph.Edges, link)
}

func (p *projection) edgeProperties(previous pdg.Ref, properties string) pdg.Ref {
	var annotations []pdg.Extension
	if previous != 0 {
		annotations = slices.Clone(p.graph.Annotations[previous-1])
	}
	annotations = append(annotations, pdg.Extension{Name: p.pool.Text("gstyle.cpp:edge-properties"), Value: p.pool.Text(properties)})
	p.graph.Annotations = append(p.graph.Annotations, annotations)
	return pdg.Ref(len(p.graph.Annotations))
}

func (p *projection) annotation(name, text string) pdg.Ref {
	data, _ := json.Marshal(text)
	p.graph.Annotations = append(p.graph.Annotations, p.pool.SingletonExtension(name, string(data)))
	return pdg.Ref(len(p.graph.Annotations))
}

func (p *projection) capabilities() {
	for _, name := range pdg.RequiredCapabilities {
		status, description := p.capability(name)
		p.graph.Capabilities = append(p.graph.Capabilities, pdg.NamedCapability{Name: p.pool.Text(name), Evidence: p.pool.Evidence(status, description)})
	}
}

func (p *projection) capability(name string) (pdg.Status, string) {
	if name == "reference_counting" && p.graph.Function.ReferenceCount.Status == pdg.Known {
		return pdg.Supported, referencePolicy
	}
	if importedCapability(name) || p.hasOverlay(capabilityOverlays[name]) {
		return pdg.Approximated, "CPG import preserves producer facts; gstyle source conformance is not established"
	}
	return pdg.Unknown, "CPG import does not establish this source-semantic capability"
}

func importedCapability(name string) bool {
	return name == "node_extraction" || name == "parameters" || name == "returns"
}

var capabilityOverlays = map[string]string{"scalar_reaching_definitions": "dataflowOss", "control_dependencies": "controlflow", "calls": "callgraph"}

func (p *projection) hasOverlay(name string) bool {
	if name == "" {
		return false
	}
	items, _ := p.doc.Metadata.Props["OVERLAYS"].([]any)
	for _, item := range items {
		if item == name {
			return true
		}
	}
	return false
}
