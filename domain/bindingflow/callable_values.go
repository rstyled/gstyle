package bindingflow

import (
	"encoding/json"
	"fmt"
	"slices"
	"unicode/utf8"

	"github.com/rstyled/gstyle/domain/pdg"
)

// CallableSet is an immutable finite may-target set. Zero means unknown.
type CallableSet struct{ key string }

func NewCallableSet(names ...string) CallableSet {
	if !validCallableNames(names) {
		return CallableSet{}
	}
	ordered := slices.Clone(names)
	slices.Sort(ordered)
	data, _ := json.Marshal(slices.Compact(ordered))
	return CallableSet{key: string(data)}
}
func validCallableNames(names []string) bool {
	if len(names) == 0 {
		return false
	}
	for _, name := range names {
		if name == "" || !utf8.ValidString(name) {
			return false
		}
	}
	return true
}
func (set CallableSet) Known() bool { return set.key != "" }
func (set CallableSet) Names() []string {
	if !set.Known() {
		return nil
	}
	var names []string
	if err := json.Unmarshal([]byte(set.key), &names); err != nil {
		panic(err)
	}
	return names
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
