package callflow

import (
	"encoding/json"
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"strings"
)

func callableValue(set bindingflow.CallableSet) exitValue {
	data, _ := json.Marshal(set.Names())
	return exitValue("c" + string(data))
}
func callableSet(value exitValue) bindingflow.CallableSet {
	if !strings.HasPrefix(string(value), "c") {
		return bindingflow.CallableSet{}
	}
	var names []string
	if err := json.Unmarshal([]byte(value[1:]), &names); err != nil {
		panic(err)
	}
	return bindingflow.NewCallableSet(names...)
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
		return callTargets{names: value.Callables.Names(), known: true}
	}
	return calls.targets[ref]
}
