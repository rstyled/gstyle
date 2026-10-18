package callflow

import (
	"encoding/json"
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"strings"
)

func callableValue(set bindingflow.CallableSet) exitValue {
	if !set.Known() {
		return unknownValue
	}
	return exitValue("c" + set.Encoding())
}
func callableSet(value exitValue) bindingflow.CallableSet {
	if !strings.HasPrefix(string(value), "c") {
		return bindingflow.CallableSet{}
	}
	var targets []bindingflow.Callable
	if err := json.Unmarshal([]byte(value[1:]), &targets); err != nil {
		panic(err)
	}
	return bindingflow.NewBoundCallableSet(targets...)
}
func (calls *CallSites) invocationTargets(ref pdg.Ref, context bindingflow.Evaluation) callTargets {
	if calls == nil {
		return callTargets{}
	}
	if calls.paths == nil {
		return calls.targets[ref]
	}
	source := calls.targets[ref].source
	if source == 0 {
		return calls.targets[ref]
	}
	value := calls.paths.Evaluate(source, context)
	if value.Nil {
		return callTargets{nilValue: true, known: true}
	}
	if value.Callables.Known() {
		return callTargets{names: value.Callables.Names(), bound: value.Callables.Targets(), known: true}
	}
	return calls.targets[ref]
}

// InvocationTargets resolves an invocation from canonical callable values and
// the caller's current bindings. The result owns its target slice. A false
// known flag requires callers to retain an unknown-target alternative.
func (calls *CallSites) InvocationTargets(ref pdg.Ref, context bindingflow.Evaluation) ([]bindingflow.Callable, bool) {
	targets := calls.invocationTargets(ref, context)
	return targets.callables(), targets.known
}

// TargetOperand is the canonical callee-value operand, or zero for a direct call.
func (calls *CallSites) TargetOperand(ref pdg.Ref) pdg.Ref {
	if calls == nil {
		return 0
	}
	return calls.targets[ref].source
}
