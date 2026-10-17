package gopatterns

import (
	"fmt"

	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (v *parameterizationVerifier) frameChecked(candidate patterns.Candidate) patterns.Candidate {
	if candidate.FixSpec == nil {
		return candidate
	}
	for _, site := range candidate.Sites {
		effect, known := v.frames[site.ID]
		if effect.Observes {
			candidate.FixSpec = nil
			return unverifiedParameterization(candidate, "helper extraction changes an explicit recover or stack-inspection call dependency")
		}
		if !known || effect.Unknown {
			candidate.FixSpec = nil
			return unverifiedParameterization(candidate, "PDG call-frame effects are unknown for this helper extraction")
		}
	}
	return candidate
}

func parameterizationFrameEffects(graphs map[string]*pdg.Graph) (map[string]callflow.FrameEffect, error) {
	current := make(map[string]*pdg.Graph, len(graphs))
	for identity, graph := range graphs {
		if !currentFrameSource(graph) {
			current[identity] = nil
			continue
		}
		current[identity] = graph
	}
	return callflow.FrameEffects(current)
}

func currentFrameSource(graph *pdg.Graph) bool {
	if graph == nil || graph.Tables == nil || graph.Tables.Sources == nil {
		return false
	}
	if len(graph.Sources) != 1 {
		return false
	}
	snapshot, known := graph.Tables.Sources.Snapshot(graph.Sources[0].Snapshot)
	return known && currentGraphSource(graph, fmt.Sprintf("%x", snapshot.Identity))
}

func rejectFrameParameterizations(report *patterns.PatternsReport, reason string) {
	for i, candidate := range report.Candidates {
		if candidate.Kind != patterns.Parameterize {
			continue
		}
		candidate.FixSpec = nil
		report.Candidates[i] = unverifiedParameterization(candidate, "PDG call-frame evidence is invalid: "+reason)
	}
}
