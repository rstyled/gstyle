package cpp

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
)

var textProperties = strings.Fields("NAME FULL_NAME FILENAME CODE TYPE_FULL_NAME METHOD_FULL_NAME SIGNATURE CONTROL_STRUCTURE_TYPE DISPATCH_TYPE CONTENT LANGUAGE VERSION ROOT EdgeProperty VARIABLE CLOSURE_BINDING_ID EVALUATION_STRATEGY CANONICAL_NAME AST_PARENT_TYPE AST_PARENT_FULL_NAME MODIFIER_TYPE")

var numberProperties = strings.Fields("LINE_NUMBER COLUMN_NUMBER LINE_NUMBER_END COLUMN_NUMBER_END ORDER ARGUMENT_INDEX INDEX OFFSET OFFSET_END")

func (n *vertex) text(name string) string { text, _ := n.Props[name].(string); return text }

func (n *vertex) flag(name string) bool { flag, _ := n.Props[name].(bool); return flag }

func (n *vertex) number(name string) int {
	text, ok := n.Props[name].(json.Number)
	if !ok {
		return -1
	}
	integer, _ := text.Int64()
	return int(integer)
}

func (n *vertex) assignOwner() {
	if n.Parent != nil {
		n.Owner = n.Parent.Owner
	}
	if n.Label == "METHOD" {
		n.Owner = n
	}
}

func (d *document) indexVertices(raw []wireNode) (map[uint64]*vertex, error) {
	nodes := make(map[uint64]*vertex, len(raw))
	for _, item := range raw {
		node, err := indexVertex(item, nodes)
		if err != nil {
			return nil, err
		}
		d.Nodes = append(d.Nodes, node)
	}
	return nodes, nil
}

func (d *document) indexSources() {
	d.indexCallables()
	d.indexRecords()
	d.indexUnboundUses()
	methodEdges := map[*vertex][]edge{}
	referrers := map[*vertex][]*vertex{}
	for _, link := range d.Edges {
		if link.Label == "REF" {
			referrers[link.Target] = append(referrers[link.Target], link.Source)
		}
		methodEdges[link.Source.Owner] = append(methodEdges[link.Source.Owner], link)
		if link.Source.Owner != link.Target.Owner {
			methodEdges[link.Target.Owner] = append(methodEdges[link.Target.Owner], link)
		}
	}
	files := map[string]*sourceFile{}
	for _, node := range d.Nodes {
		if node.Label == "FILE" {
			files[node.text("NAME")] = newSourceFile(node.text("CONTENT"))
		}
	}
	d.MethodEdges, d.Referrers = methodEdges, referrers
	d.Sources, d.Files = pdg.NewSourceContext(d.Metadata.text("ROOT"), nil), files
}

func (d *document) indexUnboundUses() {
	d.UnboundUses = map[*vertex]map[string]bool{}
	for _, node := range d.Nodes {
		if node.Label != "IDENTIFIER" || len(node.Out["REF"]) == 1 {
			continue
		}
		if d.UnboundUses[node.Owner] == nil {
			d.UnboundUses[node.Owner] = map[string]bool{}
		}
		d.UnboundUses[node.Owner][node.text("NAME")] = true
	}
}

func (d *document) indexCallables() {
	d.Callables = map[string][]*vertex{}
	d.Captures = map[string][]*vertex{}
	for _, node := range d.Nodes {
		switch node.Label {
		case "METHOD":
			d.Callables[node.text("FULL_NAME")] = append(d.Callables[node.text("FULL_NAME")], node)
		case "CLOSURE_BINDING":
			id := node.text("CLOSURE_BINDING_ID")
			d.Captures[id] = append(d.Captures[id], node)
		}
	}
}

func (d *document) indexEdges(raw []wireEdge, nodes map[uint64]*vertex) error {
	seen := map[uint64]bool{}
	for _, item := range raw {
		link, err := indexEdge(item, nodes)
		if err != nil {
			return err
		}
		if seen[link.ID] {
			return fmt.Errorf("duplicate edge ID: %d", link.ID)
		}
		seen[link.ID] = true
		d.Edges = append(d.Edges, link)
		link.Source.Out[link.Label] = append(link.Source.Out[link.Label], link.Target)
	}
	return nil
}

func (d *document) indexAST() error {
	for _, link := range d.Edges {
		if link.Label != "AST" {
			continue
		}
		if link.Target.Parent != nil {
			return fmt.Errorf("AST node %d has multiple parents", link.Target.ID)
		}
		link.Target.Parent = link.Source
		link.Source.Children = append(link.Source.Children, link.Target)
	}
	return d.owners()
}

func (d *document) owners() error {
	var queue []*vertex
	for _, node := range d.Nodes {
		if node.Parent == nil {
			queue = append(queue, node)
		}
	}
	for i := 0; i < len(queue); i++ {
		node := queue[i]
		if err := validateOperands(node); err != nil {
			return err
		}
		node.assignOwner()
		slices.SortFunc(node.Children, sourceOrder)
		queue = append(queue, node.Children...)
	}
	if len(queue) != len(d.Nodes) {
		return fmt.Errorf("CPG AST contains a cycle")
	}
	return nil
}

func (d *document) metadata() error {
	for _, node := range d.Nodes {
		if err := d.metadataNode(node); err != nil {
			return err
		}
		if sourceMethod(node) {
			d.Methods = append(d.Methods, node)
		}
	}
	return validateMetadata(d.Metadata)
}

func (d *document) metadataNode(node *vertex) error {
	if node.Label != "META_DATA" {
		return nil
	}
	if d.Metadata != nil {
		return fmt.Errorf("CPG has multiple META_DATA nodes")
	}
	d.Metadata = node
	return nil
}

func validateProperties(props map[string]any) error {
	for _, name := range textProperties {
		if item, exists := props[name]; exists {
			if _, ok := item.(string); !ok {
				return fmt.Errorf("%s must be a string", name)
			}
		}
	}
	return validateNumbers(props)
}

func validateNumbers(props map[string]any) error {
	for _, name := range numberProperties {
		item, exists := props[name]
		if !exists {
			continue
		}
		if err := integerProperty(item); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return validateFlags(props)
}

func integerProperty(item any) error {
	number, ok := item.(json.Number)
	if !ok {
		return fmt.Errorf("property must be an integer")
	}
	integer, err := number.Int64()
	if err != nil || integer < -1 || integer > math.MaxInt32 {
		return fmt.Errorf("integer is outside the supported range")
	}
	return nil
}

func validateFlags(props map[string]any) error {
	for _, name := range []string{"IS_EXTERNAL", "IS_VARIADIC"} {
		if item, exists := props[name]; exists {
			if _, ok := item.(bool); !ok {
				return fmt.Errorf("%s must be a boolean", name)
			}
		}
	}
	return nil
}

func indexGraph(ctx context.Context, wire wireGraph) (*document, error) {
	doc := &document{}
	nodes, err := doc.indexVertices(wire.Value.Vertices)
	if err != nil {
		return nil, err
	}
	if err := doc.indexEdges(wire.Value.Edges, nodes); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := doc.indexAST(); err != nil {
		return nil, err
	}
	if err := doc.metadata(); err != nil {
		return nil, err
	}
	doc.indexSources()
	return doc, nil
}

func indexVertex(raw wireNode, nodes map[uint64]*vertex) (*vertex, error) {
	id, err := identifier(raw.ID)
	if err != nil {
		return nil, fmt.Errorf("vertex ID: %w", err)
	}
	if nodes[id] != nil || raw.Label == "" {
		return nil, fmt.Errorf("duplicate vertex ID or empty label: %d", id)
	}
	props, err := properties(raw.Properties)
	if err != nil {
		return nil, fmt.Errorf("vertex %d: %w", id, err)
	}
	node := &vertex{ID: id, Label: raw.Label, Props: props, Out: map[string][]*vertex{}}
	nodes[id] = node
	return node, nil
}

func indexEdge(raw wireEdge, nodes map[uint64]*vertex) (edge, error) {
	id, err := identifier(raw.ID)
	if err != nil {
		return edge{}, err
	}
	source, err := identifier(raw.Source)
	if err != nil {
		return edge{}, err
	}
	target, err := identifier(raw.Target)
	if err != nil {
		return edge{}, err
	}
	if err := edgeEndpoints(raw.Label, nodes[source], nodes[target]); err != nil {
		return edge{}, fmt.Errorf("edge %d: %w", id, err)
	}
	props, rawProps, err := edgeProperties(raw.Properties)
	if err != nil {
		return edge{}, err
	}
	return edge{ID: id, Label: raw.Label, Source: nodes[source], Target: nodes[target], Variable: edgeVariable(props), Properties: rawProps}, nil
}

func edgeProperties(raw map[string]json.RawMessage) (map[string]any, string, error) {
	props, err := properties(raw)
	if err != nil {
		return nil, "", err
	}
	if len(props) == 0 {
		return props, "", nil
	}
	data, err := json.Marshal(props)
	return props, string(data), err
}

func edgeEndpoints(label string, source, target *vertex) error {
	if label == "" {
		return fmt.Errorf("edge has an empty label")
	}
	if source == nil || target == nil {
		return fmt.Errorf("edge has a dangling endpoint")
	}
	return nil
}

func edgeVariable(props map[string]any) string {
	if variable, ok := props["VARIABLE"].(string); ok {
		return variable
	}
	variable, _ := props["EdgeProperty"].(string)
	return variable
}

func validateOperands(node *vertex) error {
	if logical(node) != "" && len(node.Children) != 2 {
		return fmt.Errorf("logical operator %d requires two AST operands", node.ID)
	}
	return validateMethod(node)
}

func validateMethod(node *vertex) error {
	if !sourceMethod(node) {
		return nil
	}
	if node.text("NAME") == "" || node.text("FULL_NAME") == "" {
		return fmt.Errorf("source METHOD %d requires NAME and FULL_NAME", node.ID)
	}
	for _, child := range node.Children {
		if child.Label == "BLOCK" {
			return nil
		}
	}
	return fmt.Errorf("source METHOD %d has no AST body; require a full export", node.ID)
}

func sourceOrder(a, b *vertex) int {
	if result := a.number("ORDER") - b.number("ORDER"); result != 0 {
		return result
	}
	return compareID(a.ID, b.ID)
}

func compareID(a, b uint64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func validateMetadata(node *vertex) error {
	if node == nil || node.text("LANGUAGE") == "" {
		return fmt.Errorf("CPG META_DATA with LANGUAGE is required")
	}
	if node.text("LANGUAGE") != "NEWC" {
		return fmt.Errorf("unsupported CPG LANGUAGE %q; require C/C++ frontend LANGUAGE NEWC", node.text("LANGUAGE"))
	}
	version := node.text("VERSION")
	if version != "1.1" && !strings.HasPrefix(version, "4.") {
		return fmt.Errorf("unsupported CPG VERSION %q; require schema 1.1 or a 4.x producer", version)
	}
	return nil
}

func sourceMethod(n *vertex) bool {
	return n.Label == "METHOD" && !n.flag("IS_EXTERNAL") && n.text("NAME") != "<global>"
}
