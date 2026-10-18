package bindingflow

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/rstyled/gstyle/domain/pdg"
)

// CallableSet is an immutable finite may-target set. Zero means unknown.
type CallableSet struct{ key string }

// CallableCell binds a body capture slot to a live abstract storage address.
type CallableCell struct {
	Symbol  pdg.Ref
	Address string
}
type Callable struct {
	Name, Revision string
	Cells          []CallableCell
}

func NewCallableSet(names ...string) CallableSet {
	targets := make([]Callable, len(names))
	for i, name := range names {
		targets[i] = Callable{Name: name}
	}
	return NewBoundCallableSet(targets...)
}
func NewBoundCallableSet(targets ...Callable) CallableSet {
	if len(targets) == 0 {
		return CallableSet{}
	}
	records := make([]string, 0, len(targets))
	for _, target := range targets {
		normalized, valid := canonicalCallable(target)
		if !valid {
			return CallableSet{}
		}
		data, _ := json.Marshal(normalized)
		records = append(records, string(data))
	}
	slices.Sort(records)
	return CallableSet{key: "[" + strings.Join(slices.Compact(records), ",") + "]"}
}
func canonicalCallable(target Callable) (Callable, bool) {
	if target.Name == "" || !utf8.ValidString(target.Name) {
		return Callable{}, false
	}
	target.Cells = slices.Clone(target.Cells)
	slices.SortFunc(target.Cells, func(a, b CallableCell) int { return int(a.Symbol) - int(b.Symbol) })
	if !validCallableCells(target.Cells) || !validCallableRevision(target) {
		return Callable{}, false
	}
	return target, true
}
func validCallableRevision(target Callable) bool {
	return len(target.Cells) == 0 || (target.Revision != "" && utf8.ValidString(target.Revision))
}
func validCallableCells(cells []CallableCell) bool {
	previous := pdg.Ref(0)
	for _, cell := range cells {
		if cell.Symbol == 0 || cell.Symbol == previous || cell.Address == "" || !utf8.ValidString(cell.Address) {
			return false
		}
		previous = cell.Symbol
	}
	return true
}
func (set CallableSet) Known() bool { return set.key != "" }

// Encoding returns owned canonical JSON, or an empty string for unknown.
func (set CallableSet) Encoding() string { return set.key }
func (set CallableSet) Targets() []Callable {
	if !set.Known() {
		return nil
	}
	var targets []Callable
	if err := json.Unmarshal([]byte(set.key), &targets); err != nil {
		panic(err)
	}
	return targets
}
func (set CallableSet) Names() []string {
	var names []string
	for _, target := range set.Targets() {
		names = append(names, target.Name)
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// CallableIdentity reads the producer's exact declaration/body identity. It
// does not claim to carry a closure environment or a bound method receiver.
func CallableIdentity(graph *pdg.Graph, ref pdg.Ref) (string, error) {
	if graph == nil || ref == 0 || int(ref) > len(graph.Nodes) {
		return "", fmt.Errorf("invalid callable node")
	}
	attrs := graph.NodeAttributes(graph.Nodes[ref-1])
	key := "gstyle.go:function-value"
	if graph.Tables.Text(attrs.Construct) == "closure" {
		key = "gstyle.closure:body-id"
	}
	return callableIdentity(graph, attrs.Extensions, key)
}
func callableIdentity(graph *pdg.Graph, extensions []pdg.Extension, key string) (string, error) {
	var identity string
	found := false
	for _, extension := range extensions {
		if graph.Tables.Text(extension.Name) != key {
			continue
		}
		if found {
			return "", fmt.Errorf("duplicate callable identity")
		}
		found = true
		if err := json.Unmarshal([]byte(graph.Tables.Text(extension.Value)), &identity); err != nil {
			return "", err
		}
	}
	if identity == "" {
		return "", fmt.Errorf("callable identity is unknown")
	}
	return identity, nil
}

func (q *Query) callableValueFact(attrs pdg.Attributes) (Fact, bool, error) {
	key := "gstyle.go:function-value"
	if q.graph.Tables.Text(attrs.Construct) == "closure" {
		key = "gstyle.closure:body-id"
	}
	for _, extension := range attrs.Extensions {
		if q.graph.Tables.Text(extension.Name) != key {
			continue
		}
		name, err := callableIdentity(q.graph, attrs.Extensions, key)
		if err != nil {
			return Fact{}, false, err
		}
		return Fact{Known: true, Value: Value{NonNil: true, Callables: NewCallableSet(name)}}, true, nil
	}
	return Fact{}, false, nil
}
