package callflow

import (
	"cmp"
	"encoding/json"
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/constant"
	"go/version"
	"maps"
	"slices"
	"strings"
)

type exitValue string

const (
	unknownValue exitValue = ""
	nilValue     exitValue = "n"
	nonNilValue  exitValue = "p"
	falseValue   exitValue = "f"
	trueValue    exitValue = "t"
)

type exitValues struct {
	bindings, nodes map[pdg.Ref]exitValue
	cells           map[pdg.Ref]exitValue
	tuples          map[pdg.Ref]string
	unorderedReads  map[pdg.Ref]bool
}

type valueEntry struct {
	Ref   pdg.Ref
	Value exitValue
}

func (evaluation *exitEvaluation) valueContext(point exitPoint) bindingflow.Evaluation {
	values := evaluation.values[point.values]
	return bindingflow.Evaluation{Captures: evaluation.callCellValues(values), Bindings: concreteValues(values.bindings), Nodes: concreteValues(values.nodes), Facts: evaluation.body.facts, Results: concreteResults(values.tuples)}
}

func (evaluation *exitEvaluation) internValues(values exitValues) string {
	if len(values.bindings)+len(values.nodes)+len(values.tuples)+len(values.cells)+len(values.unorderedReads) == 0 {
		return ""
	}
	record := struct {
		Bindings, Nodes, Cells []valueEntry
		Tuples                 []tupleEntry
		Unordered              []pdg.Ref
	}{
		valueEntries(values.bindings), valueEntries(values.nodes), valueEntries(values.cells), tupleEntries(values.tuples), slices.Sorted(maps.Keys(values.unorderedReads)),
	}
	data, _ := json.Marshal(record)
	key := string(data)
	evaluation.values[key] = values
	return key
}

func (evaluation *exitEvaluation) recoveryValue(point exitPoint, ref pdg.Ref) exitPoint {
	value := nilValue
	if point.pending {
		value = unknownValue
		if evaluation.body.nonNilRecovery() {
			value = nonNilValue
		}
	}
	values := evaluation.values[point.values]
	nodes := map[pdg.Ref]exitValue{}
	maps.Copy(nodes, values.nodes)
	if value == unknownValue {
		delete(nodes, ref)
	} else {
		nodes[ref] = value
	}
	next := exitValues{bindings: values.bindings, nodes: nodes, tuples: values.tuples, cells: values.cells, unorderedReads: values.unorderedReads}
	point.values = evaluation.internValues(next)
	return point
}

func (evaluation *exitEvaluation) transferValues(point exitPoint, ref pdg.Ref) exitPoint {
	point, transferred := evaluation.transferCall(point, ref)
	if transferred {
		return point
	}
	point = evaluation.captureValues(point, ref)
	return evaluation.rawValueTransfer(point, ref)
}

func (evaluation *exitEvaluation) successors(point exitPoint, block pdg.FlowBlock) completions {
	truth, known := evaluation.body.paths.Evaluate(block.Condition, evaluation.valueContext(point)).Bool()
	evaluation.observeCondition(block.Condition, truth, known)
	result := completions{}
	for arm, successor := range block.Successors {
		if known && len(block.Successors) == 2 && truth != (arm == 0) {
			continue
		}
		next := point
		if len(block.Successors) == 2 {
			next = evaluation.assumeCondition(next, block.Condition, arm == 0)
		}
		next.block, next.offset = successor, 0
		result.add(evaluation.visit(next))
	}
	return result
}

// The language-default panic(nil) guarantee starts at Go 1.21.
func (body *exitBody) nonNilRecovery() bool {
	producer := body.graph.Producer
	if body.graph.Tables.Text(producer.Language) != "go" || producer.LanguageVersion.Status != pdg.Known {
		return false
	}
	target := "go" + strings.TrimPrefix(body.graph.Tables.Text(producer.LanguageVersion.Value), "go")
	return version.IsValid(target) && version.Compare(target, "go1.21") >= 0
}

func concreteValues(values map[pdg.Ref]exitValue) map[pdg.Ref]bindingflow.Value {
	result := make(map[pdg.Ref]bindingflow.Value, len(values))
	for ref, value := range values {
		result[ref] = concreteValue(value)
	}
	return result
}

func concreteValue(value exitValue) bindingflow.Value {
	switch value {
	case nilValue:
		return bindingflow.Value{Nil: true}
	case nonNilValue:
		return bindingflow.Value{NonNil: true}
	case falseValue, trueValue:
		return bindingflow.Value{Scalar: constant.MakeBool(value == trueValue)}
	default:
		set := callableSet(value)
		return bindingflow.Value{Callables: set, NonNil: set.Known()}
	}
}

func abstractValue(value bindingflow.Value) exitValue {
	if value.Callables.Known() {
		return callableValue(value.Callables)
	}
	if value.Nil {
		return nilValue
	}
	if value.NonNil {
		return nonNilValue
	}
	if truth, known := value.Bool(); known {
		if truth {
			return trueValue
		}
		return falseValue
	}
	return unknownValue
}

func valueEntries(values map[pdg.Ref]exitValue) []valueEntry {
	entries := make([]valueEntry, 0, len(values))
	for ref, value := range values {
		entries = append(entries, valueEntry{ref, value})
	}
	slices.SortFunc(entries, func(a, b valueEntry) int { return cmp.Compare(a.Ref, b.Ref) })
	return entries
}

func nextValueBindings(bindings map[pdg.Ref]exitValue, writes []bindingflow.ValueWrite) map[pdg.Ref]exitValue {
	next := map[pdg.Ref]exitValue{}
	maps.Copy(next, bindings)
	for _, write := range writes {
		if write.UnknownTarget() {
			return map[pdg.Ref]exitValue{}
		}
		if write.Symbol == 0 || write.Location != 0 {
			continue
		}
		if value := abstractValue(write.Value); value != unknownValue {
			next[write.Symbol] = value
		} else {
			delete(next, write.Symbol)
		}
	}
	return next
}
