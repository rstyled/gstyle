package goquality

import (
	"context"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestLocalReferenceFieldsKeepOwnershipBoundaries(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
type Fields struct { Items []int; Lookup map[string]int; Pointer *int; Child *Fields; Numbers Numbers }
type Embedded struct { *Fields }
type Numbers []int
func (n *Numbers) Reset() { *n = shared }
func (f *Fields) Replace() { f.Items = shared }
var shared = []int{1}
var global int
var globalRecord struct { Value int }
var globalFields Fields
func literalFields() Fields { return Fields{Items:[]int{1}} }
func pointerFields() *Fields { return &Fields{Items:[]int{1}} }
func borrowedFields(input []int) Fields { return Fields{Items:input} }
func opaqueFields() Fields { println("effect"); return Fields{Items:[]int{1}} }
func deferredFields() Fields { defer func(){}(); return Fields{Items:[]int{1}} }
func namedFields() (result Fields) { return Fields{Items:[]int{1}} }
func sizedFields(n int) Fields { return Fields{Items:make([]int,n)} }
func nextSize() int { global++; return 1 }
type GenericFields[T any] struct { Items []T }
func genericFields(input []int) GenericFields[int] { return GenericFields[int]{Items:input} }
func Factory() { f:=literalFields(); f.Items[0]=2 }
func PointerFactory() { f:=pointerFields(); f.Items[0]=2 }
func BorrowedFactory() { f:=borrowedFields(shared); f.Items[0]=2 }
func OpaqueFactory() { f:=opaqueFields(); f.Items[0]=2 }
func DeferredFactory() { f:=deferredFields(); f.Items[0]=2 }
func NamedFactory() { f:=namedFields(); f.Items[0]=2 }
func ArgumentEffects() { f:=sizedFields(nextSize()); f.Items[0]=2 }
func BorrowedGenericFactory() { f:=genericFields(shared); f.Items[0]=2 }
func Slice() { f:=Fields{Items:[]int{1}}; f.Items[0]=2 }
func PointerRoot() { f:=&Fields{Items:[]int{1}}; f.Items[0]=2 }
func Mapping() { f:=Fields{Lookup:map[string]int{}}; f.Lookup["key"]=1 }
func Pointer() { f:=Fields{Pointer:new(int)}; *f.Pointer=1 }
func Omitted() { f:=Fields{}; f.Items=make([]int,1); f.Items[0]=1 }
func SelfAppend() { f:=Fields{Items:[]int{1}}; f.Items=append(f.Items,2); f.Items[0]=3 }
func FieldAlias() { f:=Fields{Items:[]int{1}}; alias:=f.Items; alias[0]=2; f.Items[0]=3 }
func FieldCall(fn func([]int)) { f:=Fields{Items:[]int{1}}; fn(f.Items); f.Items[0]=2 }
func LocalClosure() { f:=Fields{Items:[]int{1}}; update:=func(){f.Items[0]=2}; update(); f.Items[0]=3 }
func Returned() Fields { f:=Fields{Items:[]int{1}}; f.Items[0]=2; return f }
func ReturnedField() []int { f:=Fields{Items:[]int{1}}; f.Items[0]=2; return f.Items }
func ReboundLiteral() { f:=Fields{Items:[]int{1}}; f=Fields{Items:make([]int,1)}; f.Items[0]=2 }
func Borrowed() { f:=Fields{Items:shared}; f.Items[0]=2 }
func BorrowedPointer() { f:=Fields{Pointer:&global}; *f.Pointer=1 }
func BorrowedRecordPointer() { f:=Fields{Pointer:&globalRecord.Value}; *f.Pointer=1 }
func DirectGlobalAddress() { p:=&global; *p=1 }
func DirectGlobalFieldAddress() { p:=&globalRecord.Value; *p=1 }
func Parameter(f Fields) { f.Items[0]=2 }
func IndexedPointer(input []*Fields) { alias:=input[0]; alias.Items[0]=2 }
func IndexedSlice(input [][]int) { alias:=input[0]; alias[0]=2 }
func IndexedMap(input map[string]map[string]int) { alias:=input["x"]; alias["y"]=2 }
func RangedPointer(input []*Fields) { for _, alias:=range input { alias.Items[0]=2 } }
func RangedMap(input map[string]*Fields) { for _, alias:=range input { alias.Items[0]=2 } }
func RangedArray(input [1]*Fields) { for _, alias:=range input { alias.Items[0]=2 } }
func RangedSlice(input [][]int) { for _, alias:=range input { alias[0]=2 } }
func LocalShallowRange(input *Fields) { values:=[]*Fields{input}; for _, alias:=range values { alias.Items[0]=2 } }
func MixedRange(input []*Fields, condition bool) { for _, alias:=range input { if condition {alias=&Fields{Items:[]int{1}}}; alias.Items[0]=2 } }
func ChannelRange(input chan *Fields) { for alias:=range input { alias.Items[0]=2 } }
func IteratorRange(input func(func(*Fields)bool)) { for alias:=range input { alias.Items[0]=2 } }
func CopiedRange(input []Fields) { for _, alias:=range input { alias.Items=make([]int,1); alias.Items[0]=2 } }
func KeyRange(input map[*Fields]int) { for alias:=range input { alias.Items[0]=2 } }
func ReassignedRange(input []*Fields) { alias:=new(Fields); for _, alias=range input { alias.Items[0]=2 } }
type RecursiveSlice []RecursiveSlice
func SelfRangedContainer() { value:=RecursiveSlice{nil}; for _,value=range append(value,nil) {value[0]=nil} }
func LocalShallowIndex(input *Fields) { values:=[]*Fields{input}; alias:=values[0]; alias.Items[0]=2 }
func MixedIndex(input []*Fields, condition bool) { values:=input; if condition {values=[]*Fields{}}; alias:=values[0]; alias.Items[0]=2 }
func AssertedPointer(input any) { alias:=input.(*Fields); alias.Items[0]=2 }
func ExternalPointerAlias(input *Fields) { alias:=input; alias.Items[0]=2 }
func ExternalIndexedAlias(input []Fields) { alias:=&input[0]; alias.Items[0]=2 }
func MixedPointerAlias(input *Fields, condition bool) { alias:=input; if condition {alias=&Fields{Items:[]int{1}}}; alias.Items[0]=2 }
func UnknownPointerAlias(input func()*Fields) { alias:=input(); alias.Items[0]=2 }
func LocalPointerBorrowed() { alias:=&Fields{Items:shared}; alias.Items[0]=2 }
func CopiedValue(input Fields) { alias:=input; alias.Items=make([]int,1); alias.Items[0]=2 }

func PackageVariable() { globalFields.Items[0]=2 }
func BorrowedSelfAppend() { f:=Fields{Items:shared}; f.Items=append(f.Items,2); f.Items[0]=3 }
func HeaderLater() { f:=Fields{Items:[]int{1}}; f.Items[0]=2; f.Items=shared }
func HeaderEarlier() { f:=Fields{Items:shared}; f.Items=make([]int,1); f.Items[0]=2 }
func RootLater() { f:=Fields{Items:[]int{1}}; f.Items[0]=2; f=Fields{Items:shared} }
func BranchHeader(condition bool) { f:=Fields{Items:[]int{1}}; if condition {f.Items=shared}; f.Items[0]=2 }
func RootCopy() { f:=Fields{Items:[]int{1}}; copy:=f; copy.Items[0]=2; f.Items[0]=3 }
func RootPointerAlias() { f:=&Fields{Items:[]int{1}}; alias:=f; alias.Items=shared; f.Items[0]=2 }
func RootAddress() { f:=Fields{Items:[]int{1}}; alias:=&f; alias.Items=shared; f.Items[0]=2 }
func RootCall(fn func(Fields)) { f:=Fields{Items:[]int{1}}; fn(f); f.Items[0]=2 }
func RootPointerCall(fn func(*Fields)) { f:=Fields{Items:[]int{1}}; fn(&f); f.Items[0]=2 }
func FieldAddress() { f:=Fields{Items:[]int{1}}; alias:=&f.Items; *alias=shared; f.Items[0]=2 }
func FieldAddressCall(fn func(*[]int)) { f:=Fields{Items:[]int{1}}; fn(&f.Items); f.Items[0]=2 }
func RootMethod() { f:=Fields{Items:[]int{1}}; f.Replace(); f.Items[0]=2 }
func ImplicitFieldAddress() { f:=Fields{Numbers:Numbers{1}}; f.Numbers.Reset(); f.Numbers[0]=2 }
func ClosureHeader() { f:=Fields{Items:[]int{1}}; update:=func(){f.Items=shared}; update(); f.Items[0]=2 }
func ClosureFieldAddress() { f:=Fields{Items:[]int{1}}; update:=func(){alias:=&f.Items; *alias=shared}; update(); f.Items[0]=2 }
func ClosureReturnsRoot(fn func(func()*Fields)) { f:=&Fields{Items:[]int{1}}; fn(func()*Fields{return f}); f.Items[0]=2 }
func NamedReturnAlias() (out *Fields) { f:=&Fields{Items:[]int{1}}; defer func(){out.Items=shared; f.Items[0]=2}(); return f }
func NestedReference() { f:=Fields{Child:&Fields{Items:[]int{1}}}; f.Child.Items[0]=2 }
func PromotedReference() { f:=Embedded{Fields:&Fields{Items:[]int{1}}}; f.Items[0]=2 }
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	argument := resultNamed(t, report, "ArgumentEffects")
	foundGlobal := false
	for _, effect := range argument.Effects {
		foundGlobal = foundGlobal || effect.Kind == quality.Global
	}
	if !foundGlobal {
		t.Fatalf("factory argument lost global effect: %+v", argument.Effects)
	}
	for _, name := range []string{"ExternalPointerAlias", "ExternalIndexedAlias", "IndexedPointer", "IndexedSlice", "IndexedMap", "RangedPointer", "RangedMap", "RangedArray", "RangedSlice"} {
		function := findFunction(t, facts, name)
		if got := function.Mutations[0].Provenance; got != quality.External {
			t.Fatalf("%s provenance=%s, want external", name, got)
		}
		result := resultNamed(t, report, name)
		if len(result.Effects) != 1 || result.Effects[0].Kind != quality.MutationEffect {
			t.Fatalf("shared mutation lost: %+v", result)
		}
	}
	for _, name := range []string{"MixedPointerAlias", "UnknownPointerAlias", "LocalPointerBorrowed", "CopiedValue", "LocalShallowIndex", "MixedIndex", "AssertedPointer", "LocalShallowRange", "MixedRange", "ChannelRange", "IteratorRange", "CopiedRange", "KeyRange", "ReassignedRange", "SelfRangedContainer"} {
		function := findFunction(t, facts, name)
		if got := function.Mutations[len(function.Mutations)-1].Provenance; got != quality.Unknown {
			t.Fatalf("%s provenance=%s, want unknown", name, got)
		}
	}
	global := findFunction(t, facts, "PackageVariable")
	if len(global.Effects) != 1 || global.Effects[0].Kind != quality.Global {
		t.Fatalf("global field write lost: %+v", global.Effects)
	}
	for _, test := range []struct {
		name  string
		local bool
		first bool
	}{
		{"Factory", true, false}, {"PointerFactory", true, false}, {"ArgumentEffects", true, false},
		{"BorrowedFactory", false, false}, {"OpaqueFactory", false, false}, {"DeferredFactory", false, false}, {"NamedFactory", false, false}, {"BorrowedGenericFactory", false, false},
		{"Slice", true, false}, {"PointerRoot", true, false}, {"Mapping", true, false},
		{"Pointer", true, false}, {"Omitted", true, false}, {"SelfAppend", true, false},
		{"FieldAlias", true, false}, {"FieldCall", true, false}, {"LocalClosure", true, false},
		{"Returned", true, false}, {"ReturnedField", true, false}, {"ReboundLiteral", true, false},
		{"Borrowed", false, false}, {"BorrowedPointer", false, false}, {"BorrowedRecordPointer", false, false},
		{"DirectGlobalAddress", false, false}, {"DirectGlobalFieldAddress", false, false},
		{"Parameter", false, false}, {"BorrowedSelfAppend", false, false},
		{"HeaderLater", false, true}, {"HeaderEarlier", false, false}, {"RootLater", false, true},
		{"BranchHeader", false, false}, {"RootCopy", false, false}, {"RootPointerAlias", false, false},
		{"RootAddress", false, false}, {"RootCall", false, false}, {"RootPointerCall", false, false},
		{"FieldAddress", false, false}, {"FieldAddressCall", false, false}, {"RootMethod", false, false},
		{"ImplicitFieldAddress", false, false}, {"ClosureHeader", false, false}, {"ClosureFieldAddress", false, false},
		{"ClosureReturnsRoot", false, false}, {"NamedReturnAlias", false, false}, {"NestedReference", false, false}, {"PromotedReference", false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			function := findFunction(t, facts, test.name)
			if len(function.Mutations) == 0 {
				t.Fatal("missing indexed or indirect mutation")
			}
			index := len(function.Mutations) - 1
			if test.first {
				index = 0
			}
			mutation := function.Mutations[index]
			if (mutation.Provenance == quality.Local) != test.local {
				t.Fatalf("mutation ownership = %s, want local=%t: %+v", mutation.Provenance, test.local, function.Mutations)
			}
		})
	}
}
