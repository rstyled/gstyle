package deferflow

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
)

func validateContext(graph *pdg.Graph, sites []Site) error {
	if err := uniqueInvocations(sites); err != nil {
		return err
	}
	return validateLinks(graph, sites)
}
func validateLinks(graph *pdg.Graph, sites []Site) error {
	for _, site := range sites {
		if err := registrationLink(graph, site); err != nil {
			return err
		}
		if flowContains(graph.ValueFlow, site.Invocation) || flowContains(graph.NormalFlow, site.Invocation) {
			return fmt.Errorf("deferred invocation appears in registration-time flow")
		}
		if err := captureLinks(site); err != nil {
			return err
		}
	}
	return nil
}
func registrationLink(graph *pdg.Graph, site Site) error {
	count := 0
	for _, edge := range graph.Edges {
		if savedRegistration(graph, site, edge) {
			count++
		}
	}
	if count != 1 {
		return fmt.Errorf("deferred invocation requires one saved-registration edge")
	}
	return nil
}
func captureLinks(site Site) error {
	for _, capture := range site.Captures {
		if capture.Source == site.Registration || capture.Source == site.Invocation {
			return fmt.Errorf("deferred capture depends on its own registration or invocation")
		}
	}
	return nil
}
func flowContains(flow *pdg.NormalFlow, ref pdg.Ref) bool {
	if flow == nil {
		return false
	}
	for _, block := range flow.Blocks {
		for _, operation := range block.Operations {
			if operation == ref {
				return true
			}
		}
	}
	return false
}

func savedRegistration(graph *pdg.Graph, site Site, edge pdg.Edge) bool {
	return edge.Source == site.Registration && edge.Target == site.Invocation && edge.Kind == pdg.Data && edge.Position == 0 && graph.Tables.Text(edge.Subkind) == "defer-registration"
}
