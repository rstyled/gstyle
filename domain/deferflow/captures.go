package deferflow

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
)

type captureKey struct {
	kind     string
	position pdg.Position
}

func (slot Slot) slotKey() captureKey { return captureKey{slot.Kind, slot.Position} }

func (slot Slot) validSlot() bool {
	switch slot.Kind {
	case "target":
		return slot.Position == 0 && slot.Mode == "value"
	case "argument":
		return slot.Position != 0 && slot.Mode == "value"
	case "receiver":
		return slot.validReceiverSlot()
	default:
		return false
	}
}

func (slot Slot) validReceiverSlot() bool { return slot.Position == 1 && receiverMode(slot.Mode) }

func resolveCaptures(graph *pdg.Graph, ref pdg.Ref, slots []Slot) ([]Capture, error) {
	edges, err := captureEdges(graph, ref)
	if err != nil {
		return nil, err
	}
	var captures []Capture
	for _, slot := range slots {
		capture, err := takeCapture(edges, slot)
		if err != nil {
			return nil, err
		}
		captures = append(captures, capture)
	}
	if len(edges) != 0 {
		return nil, fmt.Errorf("deferred capture edge is absent from inventory")
	}
	return captures, nil
}

func captureEdges(graph *pdg.Graph, ref pdg.Ref) (map[captureKey]pdg.Ref, error) {
	edges := map[captureKey]pdg.Ref{}
	for _, edge := range graph.Edges {
		kind := deferredCaptureKind(graph, ref, edge)
		if kind == "" {
			continue
		}
		key := captureKey{kind, edge.Position}
		if edges[key] != 0 {
			return nil, fmt.Errorf("duplicate deferred capture edge")
		}
		edges[key] = edge.Source
	}
	return edges, nil
}

func takeCapture(edges map[captureKey]pdg.Ref, slot Slot) (Capture, error) {
	if !slot.validSlot() {
		return Capture{}, fmt.Errorf("invalid deferred capture slot")
	}
	key := slot.slotKey()
	source := edges[key]
	if source == 0 {
		return Capture{}, fmt.Errorf("missing deferred capture edge")
	}
	delete(edges, key)
	return Capture{Slot: slot, Source: source}, nil
}

func deferredCaptureKind(graph *pdg.Graph, ref pdg.Ref, edge pdg.Edge) string {
	if edge.Target != ref || edge.Kind != pdg.Data {
		return ""
	}
	switch graph.Tables.Text(edge.Subkind) {
	case "defer-target":
		return "target"
	case "defer-receiver":
		return "receiver"
	case "defer-argument":
		return "argument"
	default:
		return ""
	}
}

func receiverMode(mode string) bool {
	switch mode {
	case "value", "address", "dereference", "unknown":
		return true
	default:
		return false
	}
}
