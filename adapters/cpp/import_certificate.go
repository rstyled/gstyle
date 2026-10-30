package cpp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
)

const importCapability = "cpg_method_projection"

func (p *projection) recordRelations() {
	var relations []map[string]any
	for _, link := range p.doc.MethodEdges[p.method] {
		switch link.Label {
		case "CFG", "CDG", "REACHING_DEF":
			continue
		}
		relations = append(relations, sourceLink(link))
	}
	p.extension("gstyle.cpp:relations", relations)
}

func (p *projection) certifyImport(nodes []*vertex) error {
	if err := p.verifyImport(nodes); err != nil {
		return err
	}
	p.graph.Capabilities = append(p.graph.Capabilities, pdg.NamedCapability{Name: p.pool.Text(importCapability), Evidence: p.pool.Evidence(pdg.Supported, "Complete method-owned CPG node/property inventory and incident relation projection verified against the decoded export; this certifies the import, not C++ source-semantic completeness.")})
	p.importedOverlay("cpg_data_relations", "dataflowOss")
	p.importedOverlay("cpg_control_relations", "controlflow")
	p.graph.Capabilities = append(p.graph.Capabilities, pdg.NamedCapability{Name: p.pool.Text("cpg_matching_labels"), Evidence: p.pool.Evidence(pdg.Supported, "C++ matching labels and signatures use recorded operations, conservative call effects, and explicit type classes; matching does not authorize source rewrites.")})
	return nil
}

func (p *projection) verifyImport(nodes []*vertex) error {
	if err := pdg.Validate(p.graph); err != nil {
		return err
	}
	if err := p.verifyNodeProjection(nodes); err != nil {
		return err
	}
	relations, err := p.projectedRelations()
	if err != nil {
		return err
	}
	if err := p.verifyRelations(relations); err != nil {
		return err
	}
	return p.verifyOperands(nodes)
}

func (p *projection) importedOverlay(name, overlay string) {
	status := pdg.Unsupported
	if p.hasOverlay(overlay) {
		status = pdg.Supported
	}
	p.graph.Capabilities = append(p.graph.Capabilities, pdg.NamedCapability{Name: p.pool.Text(name), Evidence: p.pool.Evidence(status, "Import certificate checks every relation from the named producer overlay; source-language semantic completeness is a separate property.")})
}

func (p *projection) verifyNodeProjection(nodes []*vertex) error {
	if len(nodes) != len(p.graph.Nodes) {
		return fmt.Errorf("CPG node projection changes the method inventory")
	}
	for _, source := range nodes {
		if err := p.verifyProjectedNode(source); err != nil {
			return err
		}
	}
	return nil
}

func (p *projection) verifyProjectedNode(source *vertex) error {
	ref := p.refs[source]
	if ref == 0 || int(ref) > len(p.graph.Nodes) {
		return fmt.Errorf("CPG node %d is not projected", source.ID)
	}
	node := p.graph.Nodes[ref-1]
	if p.pool.Tables.Text(node.ID) != strconv.FormatUint(source.ID, 10) || p.pool.Tables.Text(node.OriginalKind.Value) != source.Label || node.Category != category(source) {
		return fmt.Errorf("CPG node identity or kind changes during projection")
	}
	return p.verifyProperties(source, p.graph.NodeAttributes(node))
}

func (p *projection) verifyProperties(source *vertex, attrs pdg.Attributes) error {
	expected, err := json.Marshal(source.Props)
	if err != nil {
		return err
	}
	for _, extension := range attrs.Extensions {
		if p.pool.Tables.Text(extension.Name) == "gstyle.cpp:properties" && p.pool.Tables.Text(extension.Value) == string(expected) {
			return nil
		}
	}
	return fmt.Errorf("CPG node %d loses source properties", source.ID)
}

type projectedRelation struct {
	ID         uint64 `json:"id"`
	Source     uint64 `json:"source"`
	Target     uint64 `json:"target"`
	Kind       string `json:"kind"`
	Variable   string `json:"variable"`
	Properties string `json:"properties"`
}

func (p *projection) projectedRelations() (map[uint64]projectedRelation, error) {
	result := map[uint64]projectedRelation{}
	for _, link := range p.graph.Edges {
		if err := p.projectedDependency(result, link); err != nil {
			return nil, err
		}
	}
	for _, extension := range p.graph.Extensions {
		if err := p.projectedExtension(result, extension); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (p *projection) projectedExtension(result map[uint64]projectedRelation, extension pdg.Extension) error {
	if !relationExtension(p.pool.Tables.Text(extension.Name)) {
		return nil
	}
	var links []projectedRelation
	if err := json.Unmarshal([]byte(p.pool.Tables.Text(extension.Value)), &links); err != nil {
		return err
	}
	for _, link := range links {
		if err := addProjectedRelation(result, link); err != nil {
			return err
		}
	}
	return nil
}

func relationExtension(name string) bool {
	switch name {
	case "gstyle.cpp:cfg", "gstyle.cpp:cross-method-dependencies", "gstyle.cpp:relations":
		return true
	}
	return false
}

func (p *projection) projectedDependency(result map[uint64]projectedRelation, link pdg.Edge) error {
	kind := p.pool.Tables.Text(link.Subkind)
	if kind != "CDG" && kind != "REACHING_DEF" {
		return nil
	}
	if !dependencyKindMatches(kind, link.Kind) {
		return fmt.Errorf("CPG dependency changes canonical edge kind")
	}
	relation, err := p.decodeDependency(link, kind)
	if err != nil {
		return err
	}
	return addProjectedRelation(result, relation)
}

func sourceEdgeID(value string) (uint64, error) {
	text, found := strings.CutPrefix(value, "cpp:e")
	if !found {
		return 0, fmt.Errorf("CPG dependency loses original edge identity")
	}
	return strconv.ParseUint(text, 10, 64)
}

func (p *projection) decodeDependency(link pdg.Edge, kind string) (projectedRelation, error) {
	id, err := sourceEdgeID(p.pool.Tables.Text(link.ID))
	if err != nil {
		return projectedRelation{}, err
	}
	source, target, err := p.projectedEndpoints(link)
	if err != nil {
		return projectedRelation{}, err
	}
	variable, err := p.projectedVariable(link)
	if err != nil {
		return projectedRelation{}, err
	}
	return projectedRelation{ID: id, Source: source, Target: target, Kind: kind, Variable: variable, Properties: p.projectedProperties(link)}, nil
}

func (p *projection) projectedProperties(link pdg.Edge) string {
	if link.Annotations == 0 {
		return ""
	}
	for _, annotation := range p.graph.Annotations[link.Annotations-1] {
		if p.pool.Tables.Text(annotation.Name) == "gstyle.cpp:edge-properties" {
			return p.pool.Tables.Text(annotation.Value)
		}
	}
	return ""
}

func (p *projection) projectedEndpoints(link pdg.Edge) (uint64, uint64, error) {
	source, err := strconv.ParseUint(p.pool.Tables.Text(p.graph.Nodes[link.Source-1].ID), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	target, err := strconv.ParseUint(p.pool.Tables.Text(p.graph.Nodes[link.Target-1].ID), 10, 64)
	return source, target, err
}

func dependencyKindMatches(source string, kind pdg.EdgeKind) bool {
	return (source == "CDG" && kind == pdg.ControlEdge) || (source == "REACHING_DEF" && kind == pdg.Data)
}

func (p *projection) projectedVariable(link pdg.Edge) (string, error) {
	if link.Annotations == 0 {
		return "", nil
	}
	for _, annotation := range p.graph.Annotations[link.Annotations-1] {
		if p.pool.Tables.Text(annotation.Name) == "gstyle.cpp:variable" {
			var value string
			err := json.Unmarshal([]byte(p.pool.Tables.Text(annotation.Value)), &value)
			return value, err
		}
	}
	return "", nil
}

func addProjectedRelation(result map[uint64]projectedRelation, link projectedRelation) error {
	if _, exists := result[link.ID]; exists {
		return fmt.Errorf("CPG relation %d is projected more than once", link.ID)
	}
	result[link.ID] = link
	return nil
}

func (p *projection) verifyRelations(actual map[uint64]projectedRelation) error {
	expected := p.doc.MethodEdges[p.method]
	if len(actual) != len(expected) {
		return fmt.Errorf("CPG relation projection changes the method inventory")
	}
	for _, link := range expected {
		want := projectedRelation{ID: link.ID, Source: link.Source.ID, Target: link.Target.ID, Kind: link.Label, Variable: link.Variable, Properties: link.Properties}
		if actual[link.ID] != want {
			return fmt.Errorf("CPG relation %d changes direction, kind, variable, or identity", link.ID)
		}
	}
	return nil
}
