package cpp

import (
	"encoding/json"
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (p *projection) jsonExtension(name string, value any) pdg.Extension {
	data, _ := json.Marshal(value)
	return pdg.Extension{Name: p.pool.Text(name), Value: p.pool.Text(string(data))}
}

func (p *projection) externalBindings(nodes []*vertex) {
	for _, node := range nodes {
		if node.Label != "IDENTIFIER" || len(node.Out["REF"]) != 1 {
			continue
		}
		p.importExternalBinding(node.Out["REF"][0])
	}
}

func (p *projection) importExternalBinding(target *vertex) {
	if p.symbols[target] != 0 {
		return
	}
	if role, ok := externalRole(target); ok {
		p.externalBinding(target, role)
	}
}

func externalRole(node *vertex) (pdg.Role, bool) {
	switch node.Label {
	case "MEMBER":
		return memberRole(node), true
	case "METHOD", "TYPE_DECL":
		return pdg.OtherRole, true
	case "LOCAL", "METHOD_PARAMETER_IN":
		if node.Owner != nil && node.Owner.text("NAME") == "<global>" {
			return pdg.GlobalRole, true
		}
		return pdg.CaptureRole, node.Owner != nil
	default:
		return pdg.OtherRole, false
	}
}

func memberRole(node *vertex) pdg.Role {
	if staticStorage(node) {
		return pdg.GlobalRole
	}
	return pdg.OtherRole
}

func (p *projection) externalBinding(node *vertex, role pdg.Role) {
	identity := strconv.FormatUint(node.ID, 10)
	p.graph.Symbols = append(p.graph.Symbols, pdg.Symbol{ID: p.pool.Text("s" + identity), Name: p.pool.Known(node.text("NAME")), Type: p.pool.Known(node.text("TYPE_FULL_NAME")), Role: role,
		Extensions: []pdg.Extension{p.jsonExtension("gstyle.cpp:external-declaration", map[string]any{"id": identity, "properties": node.Props})}})
	p.symbols[node] = pdg.Ref(len(p.graph.Symbols))
}

type captureFact struct {
	ID          string   `json:"id"`
	Strategy    string   `json:"strategy"`
	Declaration []string `json:"declarations"`
}

func (p *projection) captureFacts(nodes []*vertex) {
	for _, node := range nodes {
		bindings := node.Out["CAPTURE"]
		if node.Label == "LOCAL" {
			bindings = p.doc.Captures[node.text("CLOSURE_BINDING_ID")]
		}
		if len(bindings) == 0 {
			continue
		}
		captures := importedCaptures(bindings)
		p.attrs[p.refs[node]-1].Extensions = append(p.attrs[p.refs[node]-1].Extensions, p.jsonExtension("gstyle.cpp:captures", captures))
	}
}

func importedCaptures(bindings []*vertex) []captureFact {
	var captures []captureFact
	for _, binding := range bindings {
		fact := captureFact{ID: strconv.FormatUint(binding.ID, 10), Strategy: binding.text("EVALUATION_STRATEGY")}
		for _, target := range binding.Out["REF"] {
			fact.Declaration = append(fact.Declaration, strconv.FormatUint(target.ID, 10))
		}
		captures = append(captures, fact)
	}
	return captures
}
