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
	for _, name := range []string{"Fixed", "Records", "StyleValue", "StyleRender", "StyleMethodValue"} {
		fact := findFunction(t, facts, name)
		for _, effect := range fact.Effects {
			if effect.Kind == quality.Global {
				t.Errorf("%s retains global read: %+v", name, effect)
			}
		}
	}
	for _, name := range []string{"Mutable", "Escape", "Address", "Reference", "Dynamic", "Public", "Initialized", "Closed", "Method", "RangeWrite", "Nested", "StyleWritten", "StyleAddressed", "StyleExported", "StyleDynamic", "StyleCallback", "StyleColorReference", "StyleAlias", "StyleOther"} {
		fact := findFunction(t, facts, name)
		found := false
		for _, effect := range fact.Effects {
			found = found || effect.Kind == quality.Global
		}
		if !found {
			t.Errorf("%s loses observable global dependence", name)
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
