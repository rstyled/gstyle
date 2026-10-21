package resourceflow

import (
	"strings"

	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (p *proof) save(site deferflow.Site, machine *deferflow.Machine[bool]) bool {
	values := make([]bool, len(site.Captures))
	for i, capture := range site.Captures {
		values[i] = p.query.resource(capture.Source, p.policy)
	}
	return machine.Register(site.Registration, values) == nil
}

func (p *proof) unwind(machine deferflow.Machine[bool], transfer bool) bool {
	if err := machine.BeginNormalExit(); err != nil {
		return false
	}
	return p.drain(machine, transfer)
}

func (p *proof) drain(machine deferflow.Machine[bool], transfer bool) bool {
	invocation, found, err := machine.Next()
	if err != nil {
		return false
	}
	if !found {
		return transfer && machine.Exit().Kind == deferflow.ReturnExit
	}
	if p.deferredAction(invocation) {
		return true
	}
	return p.deferredCall(machine, invocation, transfer)
}

func (p *proof) deferredAction(invocation deferflow.Invocation[bool]) bool {
	return p.query.deferredRelease(invocation, p.policy) || (p.policy.TransferCalls && p.query.deferredHandoff(invocation))
}

func (q *Query) deferredRelease(invocation deferflow.Invocation[bool], policy Policy) bool {
	if q.cleanupMethod(invocation.Node, policy) {
		if q.methodExpression(invocation.Node) {
			return savedSingleArgument(invocation.Captures)
		}
		return savedReceiver(invocation.Captures)
	}
	if !q.helper(invocation.Node, policy) {
		return false
	}
	return savedSingleArgument(invocation.Captures)
}

func (q *Query) methodExpression(ref pdg.Ref) bool {
	for _, extension := range q.graph.NodeAttributes(q.graph.Nodes[ref-1]).Extensions {
		if q.graph.Tables.Text(extension.Name) == "gstyle.go:method-expression" {
			return q.graph.Tables.Text(extension.Value) == "true"
		}
	}
	return false
}

func (q *Query) deferredReturns(ref pdg.Ref) bool {
	attrs := q.graph.NodeAttributes(q.graph.Nodes[ref-1])
	if attrs.CallForm != pdg.DirectCall || attrs.Callee.Status != pdg.Known {
		return false
	}
	name := q.graph.Tables.Text(attrs.Callee.Value)
	return name == "print" || name == "println"
}

func (q *Query) cleanupMethod(ref pdg.Ref, policy Policy) bool {
	if q.protocol.Resource != "" {
		return q.closeMethod(ref)
	}
	if policy.recognizeDeferred {
		return strings.HasSuffix(q.methods[ref], "."+policy.releaseMethod)
	}
	return q.closeMethod(ref)
}

func savedSingleArgument(captures []deferflow.SavedCapture[bool]) bool {
	found := false
	for _, capture := range captures {
		if capture.Kind == "target" {
			continue
		}
		if capture.Kind != "argument" || capture.Position != 1 || !capture.Value {
			return false
		}
		found = true
	}
	return found
}

func savedReceiver(captures []deferflow.SavedCapture[bool]) bool {
	receiver := false
	for _, capture := range captures {
		if capture.Kind == "argument" {
			return false
		}
		if capture.Kind == "receiver" {
			receiver = capture.Value && capture.Mode == "value"
		}
	}
	return receiver
}
