package goquality

import (
	"context"
	"encoding/json"
	"go/ast"
	"golang.org/x/tools/go/packages"
	"os"
	"path/filepath"
	"reflect"
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
import "slices"
import "image/color"
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
var backgroundColor = lipgloss.Color("#2b2140")
func ColorValue() color.Color { return backgroundColor }
func ColorRender() string { return lipgloss.NewStyle().Background(backgroundColor).Render("text") }
func ColorEffects(p *int, callback func()) color.Color { *p=1; println("io"); callback(); return backgroundColor }
func ColorWritten() color.Color { return colorWritten }
func ColorAddressed() color.Color { return colorAddressed }
func ReadExportedColor() color.Color { return ColorExported }
func ColorDynamic() color.Color { return colorDynamic }
func ColorIndirect() color.Color { return colorIndirect }
type mutableColor struct{ C color.RGBA }
func(c *mutableColor) RGBA()(uint32,uint32,uint32,uint32){return c.C.RGBA()}
var userColor color.Color = &mutableColor{}
func UserColor() color.Color { return userColor }
var styleColorReference = lipgloss.NewStyle().Foreground(colorReference)
var colorWritten = lipgloss.Color(tone)
var colorAddressed = lipgloss.Color(tone)
var ColorExported = lipgloss.Color(tone)
var colorDynamic = lipgloss.Color(computeText())
var anotherColor = lipgloss.Color(tone)
var colorIndirect = anotherColor
var styleColorWritten = lipgloss.NewStyle().Foreground(colorWritten)
var styleColorAddressed = lipgloss.NewStyle().Foreground(colorAddressed)
var styleColorExported = lipgloss.NewStyle().Foreground(ColorExported)
var styleColorDynamic = lipgloss.NewStyle().Foreground(colorDynamic)
var styleColorIndirect = lipgloss.NewStyle().Foreground(colorIndirect)
func StyleColorRender() string { return styleColorReference.Render("text") }
func StyleColorWritten() lipgloss.Style { return styleColorWritten }
func StyleColorAddressed() lipgloss.Style { return styleColorAddressed }
func StyleColorExported() lipgloss.Style { return styleColorExported }
func StyleColorDynamic() lipgloss.Style { return styleColorDynamic }
func StyleColorIndirect() lipgloss.Style { return styleColorIndirect }
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
var offsets = map[string]int{"a":0}
var allowed = []string{"a"}
var changedAllowed = []string{"a"}
var shadowAllowed = []string{"a"}
var callbackAllowed = []string{"a"}
var queriedReferences = [][]int{{1}}
var changingOffsets = map[string]int{"a":0}
var selfIndex = map[int]int{0:0}
var addressedIndex = map[string]int{"a":0}
var escapedIndex = map[string]int{"a":0}
var effectCount int
func effectKey()string{effectCount++;return "a"}
func LocalIndex()int{counts:=[1]int{};counts[offsets["a"]]++;return counts[0]}
func NestedIndex()int{counts:=[1]struct{n int}{};counts[offsets["a"]].n++;return counts[0].n}
func BorrowedIndex(counts []int){counts[offsets["a"]]++}
func EffectfulIndex()int{counts:=[1]int{};counts[offsets[effectKey()]]++;return counts[0]}
func FixedContains(s string)bool{return slices.Contains(allowed,s)}
func ParenthesizedContains(s string)bool{return slices.Contains(((allowed)),s)}
func ParenthesizedIndex()int{counts:=[1]int{};counts[((offsets))["a"]]++;return counts[0]}
func GenericContains(s string)bool{return slices.Contains[[]string,string](allowed,s)}
func CallbackContains()bool{return slices.ContainsFunc(callbackAllowed,func(s string)bool{return s=="a"})}
func ReferenceContains(v []int)bool{return slices.ContainsFunc(queriedReferences,func(s []int)bool{s[0]++;return true})}
func changingKey()string{changingOffsets["a"]=0;return "a"}
func ChangingIndex(){counts:=[1]int{};counts[changingOffsets[changingKey()]]++}
func ChangedContains(s string)bool{return slices.Contains(changedAllowed,s)}
func SelfIndex(){selfIndex[selfIndex[0]]++}
func AddressIndex()*map[string]int{return &((addressedIndex))}
func AddressedIndex(){counts:=[1]int{};counts[addressedIndex["a"]]++}
func EscapeIndex()map[string]int{return escapedIndex}
func EscapedIndex(){counts:=[1]int{};counts[escapedIndex["a"]]++}
func ShadowedContains()bool{slices:=struct{Contains func([]string,string)bool}{func(v []string,s string)bool{v[0]=s;return true}};return slices.Contains(shadowAllowed,"x")}
func init(){changedAllowed[0]="b"}
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
func replaceColor() { colorWritten = lipgloss.Color("#ffffff") }
var colorSlot = &colorAddressed
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
	for _, name := range []string{"ColorValue", "ColorRender", "ColorEffects", "LocalIndex", "NestedIndex", "BorrowedIndex", "FixedContains", "ParenthesizedContains", "ParenthesizedIndex", "GenericContains", "RecordValue", "ErrorValue", "Fixed", "Records", "Dispatch", "DispatchRecord", "DispatchCallback", "StyleValue", "StyleRender", "StyleMethodValue", "StyleColorReference", "StyleColorRender"} {
		fact := findFunction(t, facts, name)
		for _, effect := range fact.Effects {
			if effect.Kind.GlobalState() {
				t.Errorf("%s retains global read: %+v", name, effect)
			}
		}
	}
	for _, name := range []string{"UserColor", "ColorWritten", "ColorAddressed", "ReadExportedColor", "ColorDynamic", "ColorIndirect", "CallbackContains", "ReferenceContains", "ChangingIndex", "ChangedContains", "SelfIndex", "AddressedIndex", "EscapedIndex", "ShadowedContains", "RecordWritten", "RecordAddress", "RecordMethod", "RecordExported", "RecordReference", "ErrorWritten", "ErrorAddressed", "ErrorExported", "ErrorDynamic", "ErrorIndirect", "Mutable", "Escape", "Address", "Reference", "Dynamic", "Public", "Initialized", "Closed", "Method", "RangeWrite", "Nested", "DispatchWritten", "DispatchEscaped", "DispatchAddressed", "DispatchVariable", "DispatchBound", "StyleWritten", "StyleAddressed", "StyleExported", "StyleDynamic", "StyleCallback", "StyleColorWritten", "StyleColorAddressed", "StyleColorExported", "StyleColorDynamic", "StyleColorIndirect", "StyleAlias", "StyleOther"} {
		fact := findFunction(t, facts, name)
		found := false
		for _, effect := range fact.Effects {
			found = found || effect.Kind.GlobalState()
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
	borrowed := findFunction(t, facts, "BorrowedIndex")
	if len(borrowed.Mutations) == 0 {
		t.Fatal("table lookup loses borrowed destination mutation")
	}
	key := findFunction(t, facts, "EffectfulIndex")
	if len(key.Calls) != 1 || key.Calls[0].Callee != "example.com/stable.effectKey" {
		t.Fatalf("table lookup loses evaluated key: %+v", key.Calls)
	}
	encoded, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	var restored []quality.Function
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	effects := resultNamed(t, report, "ColorEffects")
	wantKinds := map[quality.Kind]int{quality.MutationEffect: 1, quality.IO: 1, quality.UnknownEffect: 1}
	gotKinds := map[quality.Kind]int{}
	for _, effect := range effects.Effects {
		gotKinds[effect.Kind]++
	}
	if !reflect.DeepEqual(gotKinds, wantKinds) {
		t.Errorf("stable color loses real effects: %+v", effects)
	}
	replay, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil || !reflect.DeepEqual(report, replay) {
		t.Fatal("saved style facts change report", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.Global, quality.None} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: lipglossPackage + ".Style.Render", Kind: kind})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"StyleColorRender", "ColorRender"} {
			result := resultNamed(t, overridden, name)
			if kind == quality.None {
				if len(result.Effects) != 0 {
					t.Errorf("pure style policy lost: %+v", result)
				}
				continue
			}
			if len(result.Effects) != 1 || result.Effects[0].Kind != kind {
				t.Errorf("%s style policy lost: %+v", kind, result)
			}
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
			if effect.Kind.GlobalState() {
				t.Errorf("%s retains global read: %+v", name, effect)
			}
		}
	}
	for _, name := range []string{"Mutate", "Names", "Copy", "Escaped", "Address", "Exported", "Written", "Slice", "TableAddress", "Custom"} {
		found := false
		for _, effect := range findFunction(t, facts, name).Effects {
			found = found || effect.Kind.GlobalState()
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
