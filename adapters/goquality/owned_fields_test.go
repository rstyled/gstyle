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
	global := findFunction(t, facts, "PackageVariable")
	if len(global.Effects) != 1 || global.Effects[0].Kind != quality.Global {
		t.Fatalf("global field write lost: %+v", global.Effects)
	}
	for _, test := range []struct {
		name  string
		local bool
		first bool
	}{
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
