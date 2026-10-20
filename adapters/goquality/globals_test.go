package goquality

import (
	"context"
	"go/ast"
	"golang.org/x/tools/go/packages"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestStableLiteralTableReads(t *testing.T) {
	root := t.TempDir()
	sum, err := os.ReadFile("../../go.sum")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"go.mod": "module example.com/stable\n\ngo 1.26\n\nrequire charm.land/lipgloss/v2 v2.0.6\n",
		"go.sum": string(sum),
		"tables.go": `package stable
import "charm.land/lipgloss/v2"
import "errors"
type valueRecord struct{Value int;Name string;Numbers [1]int}
func(r *valueRecord) Touch(){r.Value++}
var fixedRecord=valueRecord{Value:1,Name:"fixed",Numbers:[1]int{2}}
var writtenRecord=valueRecord{Value:1}
var addressedRecord=valueRecord{Value:1}
var methodRecord=valueRecord{Value:1}
var ExportedRecord=valueRecord{Value:1}
var referencedRecord=struct{Value *int}{Value:new(int)}
func RecordValue()valueRecord{return fixedRecord}
func RecordWritten()valueRecord{return writtenRecord}
func RecordAddress()*valueRecord{return &addressedRecord}
func RecordMethod(){methodRecord.Touch()}
func RecordExported()valueRecord{return ExportedRecord}
func RecordReference()*int{return referencedRecord.Value}
func init(){writtenRecord.Value++}
var sentinel = errors.New("fixed")
var sentinelWritten = errors.New("fixed")
var sentinelAddressed = errors.New("fixed")
var ExportedSentinel = errors.New("fixed")
var sentinelDynamic = errors.New(computeText())
var sentinelAlias = errors.New
var sentinelIndirect = sentinelAlias("fixed")
func computeText() string { return "fixed" }
func ErrorValue() error { return sentinel }
func ErrorWritten() error { return sentinelWritten }
func ErrorAddressed() *error { return &sentinelAddressed }
func ErrorExported() error { return ExportedSentinel }
func ErrorDynamic() error { return sentinelDynamic }
func ErrorIndirect() error { return sentinelIndirect }
const tone = "#" + "123456"
var style = lipgloss.NewStyle().Foreground(lipgloss.Color(tone))
var styleWritten = lipgloss.NewStyle().Foreground(lipgloss.Color(tone))
var styleAddressed = lipgloss.NewStyle().Foreground(lipgloss.Color(tone))
var ExportedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(tone))
var styleDynamic = lipgloss.NewStyle().Foreground(lipgloss.Color(string([]byte(tone))))
var styleCallback = lipgloss.NewStyle().Foreground(lipgloss.Color(tone)).Transform(func(s string) string { println(s); return s })
var colorReference = lipgloss.Color(tone)
var styleColorReference = lipgloss.NewStyle().Foreground(colorReference)
var newStyle = lipgloss.NewStyle
var styleAlias = newStyle().Foreground(lipgloss.Color(tone))
var styleOther any = lipgloss.NewStyle().Foreground(lipgloss.Color(tone))
func StyleValue() lipgloss.Style { return style }
func StyleRender() string { return style.Render("text") }
func StyleMethodValue() func(...string) string { return style.Render }
func StyleWritten() lipgloss.Style { return styleWritten }
func StyleAddressed() *lipgloss.Style { return &styleAddressed }
func StyleExported() lipgloss.Style { return ExportedStyle }
func StyleDynamic() lipgloss.Style { return styleDynamic }
func StyleCallback() lipgloss.Style { return styleCallback }
func StyleColorReference() lipgloss.Style { return styleColorReference }
func StyleAlias() lipgloss.Style { return styleAlias }
func StyleOther() any { return styleOther }
var fixed = map[string]int{"a": 1}
var names = [...]string{"a", "b"}
var records = []struct{ name string; code int }{{name: "a", code: 1}}
var mutated = map[string]int{"a": 1}
var escaped = []int{1}
var addressed = [...]int{1}
var reference = map[string][]int{"a": {1}}
var dynamic = []int{compute()}
var Exported = []int{1}
var initWritten = []int{1}
var closureWritten = []int{1}
var ranged = []int{1}
var nested = [1][1]int{{1}}
var methodCalled = [...]counter{1}
var dispatch = map[string]func()int{"run": compute}
var steps = []struct{run func()int}{{compute}}
var callbacks = []func()int{func()int{callbackCount++;return callbackCount}}
var callbackCount int
var dispatchWritten = []func()int{compute}
var dispatchEscaped = []func()int{compute}
var dispatchAddressed = []func()int{compute}
var dispatchVariable = []func()int{changingFunction}
var changingFunction = compute
var dispatchBound = []func(){receiver.change}
var receiver counter
func Dispatch()int{return dispatch["run"]()}
func DispatchRecord()int{return steps[0].run()}
func DispatchCallback()int{return callbacks[0]()}
func DispatchWritten()int{dispatchWritten[0]=compute;return dispatchWritten[0]()}
func DispatchEscaped()[]func()int{return dispatchEscaped}
func DispatchAddressed()*func()int{return &dispatchAddressed[0]}
func DispatchVariable()int{return dispatchVariable[0]()}
func DispatchBound(){dispatchBound[0]()}
type counter int
func (c *counter) change() { *c++ }
func compute() int { return 1 }
func Fixed() int { return fixed["a"] + len(fixed) + len(names[0]) }
func Records() int { n:=0; for _, record := range records { n += record.code }; return n }
func Mutable() int { return mutated["a"] }
func Escape() []int { return escaped }
func Address() *int { return &addressed[0] }
func Reference() []int { return reference["a"] }
func Dynamic() int { return dynamic[0] }
func Public() int { return Exported[0] }
func Initialized() int { return initWritten[0] }
func Closed() int { return closureWritten[0] }
func RangeWrite() { for ranged[0] = range []int{1} {} }
func Nested() []int { return nested[0][:] }
func Method() { methodCalled[0].change() }
`,
		"writes.go": `package stable
import "charm.land/lipgloss/v2"
func replaceError() { sentinelWritten = nil }
func write() { mutated["a"] = 2 }
func init() { initWritten[0] = 2; styleWritten = lipgloss.NewStyle() }
func closure() func() { return func() { closureWritten[0]++ } }
`,
	}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	facts, err := (Analyzer{Root: root, BuildFlags: []string{"-mod=mod"}}).Extract(context.Background(), []string{"tables.go"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"RecordValue", "ErrorValue", "Fixed", "Records", "Dispatch", "DispatchRecord", "DispatchCallback", "StyleValue", "StyleRender", "StyleMethodValue"} {
		fact := findFunction(t, facts, name)
		for _, effect := range fact.Effects {
			if effect.Kind == quality.Global {
				t.Errorf("%s retains global read: %+v", name, effect)
			}
		}
	}
	for _, name := range []string{"RecordWritten", "RecordAddress", "RecordMethod", "RecordExported", "RecordReference", "ErrorWritten", "ErrorAddressed", "ErrorExported", "ErrorDynamic", "ErrorIndirect", "Mutable", "Escape", "Address", "Reference", "Dynamic", "Public", "Initialized", "Closed", "Method", "RangeWrite", "Nested", "DispatchWritten", "DispatchEscaped", "DispatchAddressed", "DispatchVariable", "DispatchBound", "StyleWritten", "StyleAddressed", "StyleExported", "StyleDynamic", "StyleCallback", "StyleColorReference", "StyleAlias", "StyleOther"} {
		fact := findFunction(t, facts, name)
		found := false
		for _, effect := range fact.Effects {
			found = found || effect.Kind == quality.Global
		}
		if !found {
			t.Errorf("%s loses observable global dependence", name)
		}
	}
	for _, name := range []string{"Dispatch", "DispatchRecord", "DispatchCallback"} {
		fact := findFunction(t, facts, name)
		if len(fact.Calls) != 1 || !fact.Calls[0].Dynamic {
			t.Errorf("%s loses dispatch call evidence: %+v", name, fact.Calls)
		}
	}
	render := findFunction(t, facts, "StyleRender")
	if len(render.Calls) != 1 || render.Calls[0].Callee != lipglossPackage+".Style.Render" {
		t.Fatalf("stable style loses rendering call evidence: %+v", render.Calls)
	}
}

func TestOpaquePackageGlobals(t *testing.T) {
	for name, pkg := range map[string]*packages.Package{
		"native":   {OtherFiles: []string{"write.s"}},
		"unsafe":   {Imports: map[string]*packages.Package{"unsafe": {}}},
		"linkname": {Syntax: []*ast.File{{Comments: []*ast.CommentGroup{{List: []*ast.Comment{{Text: "//go:linkname table other.table"}}}}}}},
	} {
		t.Run(name, func(t *testing.T) {
			if !opaqueGlobals(pkg) {
				t.Fatal("opaque package permits stable-global proof")
			}
		})
	}
}

func TestAuditedStyleModule(t *testing.T) {
	for _, tc := range []struct {
		name   string
		module *packages.Module
		want   bool
	}{
		{"missing metadata", nil, false},
		{"audited", &packages.Module{Path: lipglossPackage, Version: "v2.0.6"}, true},
		{"workspace", &packages.Module{Path: lipglossPackage}, false},
		{"other version", &packages.Module{Path: lipglossPackage, Version: "v2.0.7"}, false},
		{"other module", &packages.Module{Path: "example.com/style", Version: "v2.0.6"}, false},
		{"replaced", &packages.Module{Path: lipglossPackage, Version: "v2.0.6", Replace: &packages.Module{Dir: "/local"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := &packages.Package{Imports: map[string]*packages.Package{lipglossPackage: {Module: tc.module}}}
			if got := auditedStyleModule(pkg); got != tc.want {
				t.Fatalf("audited module=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestStableConstructedMetadata(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("reflect";"regexp")
var expression=regexp.MustCompile("x")
var mutable=regexp.MustCompile("x")
var names=regexp.MustCompile("(?P<name>x)")
var copied=regexp.MustCompile("x")
var escaped=regexp.MustCompile("x")
var alias=escaped
var addressed=regexp.MustCompile("x")
var exported=regexp.MustCompile("x")
var Public=regexp.MustCompile("x")
var typesTable=[]reflect.Type{reflect.TypeOf(0),reflect.TypeFor[string]()}
var typeArray=[1]reflect.Type{reflect.TypeOf(struct{}{})}
var written=[]reflect.Type{reflect.TypeOf(0)}
var sliced=[]reflect.Type{reflect.TypeOf(0)}
var tableAddress=[]reflect.Type{reflect.TypeOf(0)}
type fakeType struct{reflect.Type}
func customType()reflect.Type{return fakeType{}}
var custom=[]reflect.Type{customType()}
func Match()bool{return expression.MatchString("x")}
func Mutate()bool{mutable.Longest();return mutable.MatchString("x")}
func Names()[]string{return names.SubexpNames()}
func Copy()*regexp.Regexp{return copied.Copy()}
func Escaped()bool{return escaped.MatchString("x")}
func Address()**regexp.Regexp{return &addressed}
func Exported()bool{return Public.MatchString("x")}
func Types()reflect.Type{return typesTable[0]}
func TypeArray()reflect.Type{return typeArray[0]}
func Written()reflect.Type{written[0]=reflect.TypeOf("");return written[0]}
func Slice()[]reflect.Type{return sliced[:]}
func TableAddress()*[]reflect.Type{return &tableAddress}
func Custom()reflect.Type{return custom[0]}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Match", "Types", "TypeArray"} {
		for _, effect := range findFunction(t, facts, name).Effects {
			if effect.Kind == quality.Global {
				t.Errorf("%s retains global read: %+v", name, effect)
			}
		}
	}
	for _, name := range []string{"Mutate", "Names", "Copy", "Escaped", "Address", "Exported", "Written", "Slice", "TableAddress", "Custom"} {
		found := false
		for _, effect := range findFunction(t, facts, name).Effects {
			found = found || effect.Kind == quality.Global
		}
		if !found {
			t.Errorf("%s loses shared metadata effect", name)
		}
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "regexp.Regexp.MatchString", Kind: quality.IO})
	report, err := quality.Evaluate(facts, config)
	if err != nil {
		t.Fatal(err)
	}
	result := resultNamed(t, report, "Match")
	if len(result.Effects) != 1 || result.Effects[0].Kind != quality.IO {
		t.Fatalf("metadata proof hid call override: %+v", result)
	}
}
