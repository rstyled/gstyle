// Package callflow derives interprocedural effects from canonical PDG call
// targets. Frame summaries concern explicit frame-sensitive calls, not general
// termination, implicit faults, scheduling, memory effects or program equivalence.
package callflow

import (
	"fmt"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

// FrameEffect distinguishes an observed dependency from incomplete evidence.
type FrameEffect struct{ Observes, Unknown bool }
type frameDescriptor struct {
	local   FrameEffect
	targets []string
}

// FrameEffects computes a monotone fixed point over package-local call graphs.
// Recursion does not invent an observation. Missing bodies/targets stay unknown.
// Calls outside reachable flow are included conservatively, including defers.
func FrameEffects(graphs map[string]*pdg.Graph) (map[string]FrameEffect, error) {
	descriptors := map[string]frameDescriptor{}
	for name, graph := range graphs {
		descriptor, err := describeFrames(graph)
		if err != nil {
			return nil, fmt.Errorf("frame summary %s: %w", name, err)
		}
		if graph != nil && graph.Tables.Text(graph.Function.ID) != name {
			return nil, fmt.Errorf("frame summary key does not match function identity: %s", name)
		}
		descriptors[name] = descriptor
	}
	return solveFrames(descriptors), nil
}
func describeFrames(graph *pdg.Graph) (frameDescriptor, error) {
	if graph == nil {
		return frameDescriptor{local: FrameEffect{Unknown: true}}, nil
	}
	query, err := bindingflow.New(graph)
	if err != nil {
		return frameDescriptor{}, err
	}
	description := frameDescriptor{local: FrameEffect{Unknown: pdg.BodyFlow(graph) == nil}}
	for i, node := range graph.Nodes {
		if node.Category != pdg.Call {
			continue
		}
		targets, known := frameTargets(graph, query, pdg.Ref(i+1))
		description.local.Unknown = description.local.Unknown || !known
		description.targets = append(description.targets, targets...)
	}
	return description, nil
}
func solveFrames(descriptors map[string]frameDescriptor) map[string]FrameEffect {
	summaries := make(map[string]FrameEffect, len(descriptors))
	for name, description := range descriptors {
		summaries[name] = description.local
	}
	for changed := true; changed; {
		changed = false
		for name, description := range descriptors {
			next := completeFrames(description, summaries)
			if summaries[name] == next {
				continue
			}
			summaries[name] = next
			changed = true
		}
	}
	return summaries
}
func completeFrames(description frameDescriptor, summaries map[string]FrameEffect) FrameEffect {
	effect := description.local
	for _, target := range description.targets {
		next := targetFrameEffect(target, summaries)
		effect = joinFrames(effect, next)
	}
	return effect
}
func joinFrames(a, b FrameEffect) FrameEffect {
	return FrameEffect{Observes: a.Observes || b.Observes, Unknown: a.Unknown || b.Unknown}
}

func targetFrameEffect(target string, summaries map[string]FrameEffect) FrameEffect {
	if observesFrame(target) {
		return FrameEffect{Observes: true}
	}
	if effect, known := summaries[target]; known {
		return effect
	}
	return externalFrameEffect(target)
}
