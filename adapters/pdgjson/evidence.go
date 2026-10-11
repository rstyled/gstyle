package pdgjson

import "github.com/shanejonas/cyclo/domain/pdg"

func (x exporter) extractionEvidence() object {
	capabilities := object{}
	for _, c := range x.g.Capabilities {
		capabilities[x.text(c.Name)] = x.capability(c.Evidence)
	}
	return object{"capabilities": capabilities, "diagnostics": mapItems(x.g.Diagnostics, x.diagnostic), "conformance": x.conformance(x.g.Conformance)}
}
func (x exporter) diagnostic(d pdg.Diagnostic) object {
	severity := [...]string{"info", "warning", "error"}
	out := object{"code": x.text(d.Code), "severity": severity[d.Severity], "message": x.text(d.Message)}
	x.optionalSpan(out, d.Span)
	if d.Node != 0 {
		out["node_id"] = x.text(x.g.Nodes[d.Node-1].ID)
	}
	x.optionalText(out, "capability", d.Capability)
	return out
}
func (x exporter) conformance(c pdg.Conformance) object {
	states := [...]string{"not_evaluated", "conformant", "non_conformant"}
	return object{"status": states[c.Status], "reasons": x.texts(c.Reasons)}
}
