package joern

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rstyled/gstyle/domain"
	"github.com/rstyled/gstyle/domain/pdg"
)

const Profile = "gstyle.joern.import"

type Function struct {
	Path  string
	Score domain.Function
	Graph *pdg.Graph
}

type Extraction struct {
	Root      string
	Language  string
	Functions []Function
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
		result.Functions = append(result.Functions, Function{Path: builder.path(), Score: scoreMethod(method, recursive[method], builder.file), Graph: graph})
	}
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
	return &projection{doc: doc, method: method, pool: pool, graph: &pdg.Graph{Tables: pool.Tables}, refs: map[*vertex]pdg.Ref{}, symbols: map[*vertex]pdg.Ref{}}
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
	p.bindings(nodes)
	for _, node := range nodes {
		p.operands(node)
	}
	p.dependencies()
	p.capabilities()
	if err := pdg.PackAttributes(p.graph, p.attrs); err != nil {
		return nil, err
	}
	if err := pdg.Validate(p.graph); err != nil {
		return nil, err
	}
	return p.graph, nil
}

func (p *projection) header() {
	b, g := p.pool, p.graph
	g.Profile = b.Policy(Profile, "1")
	g.Producer = pdg.Producer{Name: b.Text("gstyle-joern"), Version: b.Text("1"), Language: b.Text(p.doc.Metadata.text("LANGUAGE")), LanguageVersion: b.Missing(pdg.Unknown, "not recorded by CPG metadata"), Configuration: b.Text(`{}`)}
	g.Sources = []pdg.Source{{ID: b.Text(p.path()), Path: b.Text(p.path()), ContentIdentity: b.Missing(pdg.Unknown, "source bytes are not verified by GraphSON")}}
	g.Function = pdg.Function{ID: b.Text("joern:" + strconv.FormatUint(p.method.ID, 10)), Name: b.Known(p.method.text("NAME")), QualifiedName: b.Known(p.method.text("FULL_NAME")), Span: pdg.Span{Source: 1}, Interface: b.Evidence(pdg.Approximated, "imported method interface"), ReferenceCount: pdg.CountFact{Status: pdg.Unknown, Reason: b.Text("reference-count policy not evaluated")}}
	p.source()
	p.extension("gstyle.joern:method", p.method.Props)
	p.extension("gstyle.joern:overlays", p.doc.Metadata.Props["OVERLAYS"])
}

func (p *projection) extension(name string, value any) {
	data, _ := json.Marshal(value) // Values are validated GraphSON primitives.
	p.graph.Extensions = append(p.graph.Extensions, pdg.Extension{Name: p.pool.Text(name), Value: p.pool.Text(string(data))})
}

var categories = map[string]pdg.Category{
	"METHOD": pdg.Entry, "METHOD_PARAMETER_IN": pdg.FormalInput,
	"METHOD_PARAMETER_OUT": pdg.FormalOutput, "METHOD_RETURN": pdg.FormalOutput,
	"RETURN": pdg.Return, "IDENTIFIER": pdg.Read, "LOCAL": pdg.Declaration,
	"CONTROL_STRUCTURE": pdg.Control, "LITERAL": pdg.Computation, "CALL": pdg.Call,
}

func category(node *vertex) pdg.Category {
	if strings.HasPrefix(node.text("NAME"), "<operator>.") {
		return operatorCategory(node.text("NAME"))
	}
	if kind, exists := categories[node.Label]; exists {
		return kind
	}
	return pdg.Other
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
	item := pdg.Node{ID: b.Text(strconv.FormatUint(node.ID, 10)), Category: category(node), OriginalKind: b.Known(node.Label), Line: uint32(max(0, node.number("LINE_NUMBER"))), Attributes: pdg.Ref(len(p.attrs)), Synthetic: !present, Evidence: b.Evidence(pdg.Approximated, "projected CPG source node; source-location extension retains CPG properties"), Provenance: b.Origin(pdg.Projection, "Joern GraphSON node projection")}
	if present {
		p.graph.Spans = append(p.graph.Spans, span)
		item.Span = pdg.Ref(len(p.graph.Spans))
	}
	p.graph.Nodes = append(p.graph.Nodes, item)
	p.parameter(node)
}

func (p *projection) attributes(node *vertex) pdg.Attributes {
	data, _ := json.Marshal(node.Props)
	attrs := pdg.Attributes{Description: p.pool.Text(node.Label), Extensions: p.pool.SingletonExtension("gstyle.joern:properties", string(data))}
	if node.Label == "LITERAL" {
		attrs.Literal = p.pool.Known(node.text("CODE"))
	}
	if node.Label != "CALL" {
		return attrs
	}
	if strings.HasPrefix(node.text("NAME"), "<operator>.") {
		attrs.Operator = p.pool.Known(node.text("NAME"))
		return attrs
	}
	attrs.Callee = p.pool.Missing(pdg.Unknown, "CPG call target is unresolved")
	if name := node.text("METHOD_FULL_NAME"); name != "" {
		attrs.Callee = p.pool.Known(name)
	}
	attrs.CallForm = pdg.UnknownCall
	if node.text("DISPATCH_TYPE") == "STATIC_DISPATCH" {
		attrs.CallForm = pdg.DirectCall
	}
	return attrs
}

func (p *projection) parameter(node *vertex) {
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

// Syntax operands run from the value expression to its consumer. AST and
// ARGUMENT edges themselves remain containment, never reaching definitions.
func (p *projection) operands(node *vertex) {
	if node.Label != "CALL" && node.Label != "RETURN" {
		return
	}
	for _, operand := range node.Children {
		if !valueOperand(node, operand) {
			continue
		}
		p.addEdge(dependency{Source: operand, Target: node, Kind: pdg.Data, Subkind: "value-operand", Position: max(0, operand.number("ARGUMENT_INDEX")), ID: "operand:" + strconv.FormatUint(operand.ID, 10) + ":" + strconv.FormatUint(node.ID, 10)})
	}
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
		if link.Target.Owner != p.method {
			cross = append(cross, sourceLink(link))
			continue
		}
		p.addEdge(dependency{Source: link.Source, Target: link.Target, Kind: kind, Subkind: link.Label, Position: -1, Variable: link.Variable, ID: "joern:e" + strconv.FormatUint(link.ID, 10)})
	}
	p.extension("gstyle.joern:cfg", cfg)
	p.extension("gstyle.joern:cross-method-dependencies", cross)
}

func sourceLink(link edge) map[string]any {
	return map[string]any{"source": link.Source.ID, "target": link.Target.ID, "kind": link.Label, "variable": link.Variable}
}

type dependency struct {
	Source, Target        *vertex
	Kind                  pdg.EdgeKind
	ID, Subkind, Variable string
	Position              int
}

func (p *projection) addEdge(input dependency) {
	from, to := p.refs[input.Source], p.refs[input.Target]
	if from == 0 || to == 0 {
		return
	}
	link := pdg.Edge{ID: p.pool.Text(input.ID), Source: from, Target: to, Kind: input.Kind, Subkind: p.pool.Text(input.Subkind), Evidence: p.pool.Evidence(pdg.Approximated, "imported Joern dependencies; alias, call and exceptional precision not certified"), Provenance: p.pool.Origin(pdg.Projection, "Joern relation projection")}
	if input.Position >= 0 {
		link.Position = pdg.Position(input.Position + 1)
	}
	if input.Variable != "" {
		link.Annotations = p.annotation("gstyle.joern:variable", input.Variable)
	}
	p.graph.Edges = append(p.graph.Edges, link)
}

func (p *projection) annotation(name, text string) pdg.Ref {
	data, _ := json.Marshal(text)
	p.graph.Annotations = append(p.graph.Annotations, p.pool.SingletonExtension(name, string(data)))
	return pdg.Ref(len(p.graph.Annotations))
}

func (p *projection) capabilities() {
	for _, name := range pdg.RequiredCapabilities {
		status := pdg.Unknown
		if name == "node_extraction" || name == "parameters" || name == "returns" {
			status = pdg.Approximated
		}
		p.graph.Capabilities = append(p.graph.Capabilities, pdg.NamedCapability{Name: p.pool.Text(name), Evidence: p.pool.Evidence(status, "CPG import preserves producer facts; gstyle source conformance is not established")})
	}
}
