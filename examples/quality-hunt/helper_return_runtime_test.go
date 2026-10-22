package main

import (
	"testing"
)

func TestHelperReturnDevelopmentRuntime(t *testing.T) {
	definitions := []assuranceFamily{
		{"helper-composed-fresh", "local := returnFreshChain()\nlocal.Count++", 0, false},
		{"helper-composed-borrowed", "local := returnBorrowChain(shared)\nlocal.Count++", 1, false},
		{"helper-fresh-io", "local := returnFreshIO()\nlocal.Count++", 0, false},
		{"helper-borrowed-io", "local := returnBorrowIO(shared)\nlocal.Count++", 1, false},
		{"helper-mixed", "local := returnMixed(shared, shared.Count >= 0)\nlocal.Count++", -1, false},
		{"helper-rebound", "returnGlobal = shared\nlocal := returnRebound(new(State))\nlocal.Count++", 1, false},
		{"helper-published", "local := returnPublished(shared)\nlocal.Count++", 1, false},
	}
	names := []string{"RuntimeFresh", "RuntimeBorrow", "RuntimeFreshIO", "RuntimeBorrowIO", "RuntimeMixed", "RuntimeRebound", "RuntimePublished"}
	var cases []specimen
	families := map[string]assuranceFamily{}
	for index, family := range definitions {
		name := names[index]
		cases = append(cases, specimen{Name: name, Base: name, Family: family.name, Variant: "original", Body: family.body})
		families[name] = family
	}
	source := append(program(cases), []byte(`
var returnGlobal *State
func returnFreshLeaf()*State{return new(State)}
func returnFreshChain()*State{return returnFreshLeaf()}
func returnBorrowLeaf(value *State)*State{return value}
func returnBorrowChain(value *State)*State{return returnBorrowLeaf(value)}
func returnFreshIO()*State{println("retained IO");return new(State)}
func returnBorrowIO(value *State)*State{println("retained IO");return value}
func returnMixed(value *State,borrow bool)*State{if borrow{return value};return new(State)}
func returnRebound(value *State)*State{value=returnGlobal;return value}
func returnPublished(value *State)*State{returnGlobal=value;return value}
`)...)
	assuranceAssess(t, cases, families, source)
}
