package gopatterns

import (
	"fmt"
	"go/format"
	"sort"
	"strings"
	"unicode"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

type sourceFix struct {
	context                 *pdg.SourceContext
	edits                   map[pdg.Ref][]pdg.SourceEdit
	targets                 map[string]pdg.SourceFunction
	positions               map[string][]int
	typeName, parameterName string
	method                  bool
}

func newSourceFix(spec *patterns.FixSpec) (*sourceFix, error) {
	if spec.Context == nil {
		return nil, fmt.Errorf("typed source context is missing")
	}
	plan := &sourceFix{context: spec.Context, edits: make(map[pdg.Ref][]pdg.SourceEdit), targets: make(map[string]pdg.SourceFunction), positions: make(map[string][]int), method: spec.Kind == patterns.AnemicModel}
	if err := plan.selectTargets(spec); err != nil {
		return nil, err
	}
	if len(plan.targets) == 0 {
		return nil, unsupportedFix("no supported signature targets")
	}
	if err := plan.checkTargetPackages(); err != nil {
		return nil, err
	}
	return plan, nil
}

func (p *sourceFix) checkTargetPackages() error {
	var name string
	for _, target := range p.targets {
		if name != "" && name != target.Package {
			return unsupportedFix("signature targets span multiple packages")
		}
		name = target.Package
	}
	return nil
}

func (p *sourceFix) selectTargets(spec *patterns.FixSpec) error {
	ref, err := signatureSourceRef(p.context, spec)
	if err != nil {
		return err
	}
	for _, fn := range p.context.Functions {
		if !signatureTarget(fn, spec, ref) {
			continue
		}
		positions := signaturePositions(fn, spec)
		if len(positions) == 0 {
			continue
		}
		p.targets[fn.ID] = fn
		p.positions[fn.ID] = positions
	}
	if !p.method {
		return p.valueNames(spec.Params["group"])
	}
	return nil
}

func (p *sourceFix) valueNames(group string) error {
	var params []ClumpParam
	for _, key := range strings.Split(group, ",") {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid parameter group")
		}
		params = append(params, ClumpParam{Name: parts[0], Type: parts[1]})
	}
	p.typeName = valueTypeName(params)
	p.parameterName = lowerFirst(p.typeName)
	for _, fn := range p.targets {
		if hasBinding(fn.Bindings, p.parameterName) {
			return unsupportedFix("generated parameter collides with a binding")
		}
		if !contiguousPositions(p.positions[fn.ID]) {
			return unsupportedFix("parameter grouping would change argument evaluation order")
		}
	}
	return nil
}

func (p *sourceFix) declarations() error {
	ids := make([]string, 0, len(p.targets))
	for id := range p.targets {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		fn := p.targets[id]
		if err := p.declaration(fn); err != nil {
			return err
		}
	}
	if !p.method {
		p.insertValueType(p.targets[ids[0]])
	}
	return nil
}

func (p *sourceFix) declaration(fn pdg.SourceFunction) error {
	positions := p.positions[fn.ID]
	parameters := rewriteParameters(fn, positions, p.parameterName, p.typeName)
	if p.method {
		parameters = methodParameters(fn)
	}
	if err := p.add(fn.Source, fn.Parameters, parameters, true); err != nil {
		return err
	}
	if p.method {
		receiver := fn.ParametersList[0]
		span := pdg.ByteRange{Start: fn.NameRange.Start, End: fn.NameRange.Start}
		return p.add(fn.Source, span, "("+receiver.Name+" "+receiver.TypeSyntax+") ", false)
	}
	return p.parameterUses(fn, positions)
}

func (p *sourceFix) parameterUses(fn pdg.SourceFunction, positions []int) error {
	for _, position := range positions {
		param := fn.ParametersList[position]
		if param.Addressed {
			return unsupportedFix("parameter address-taking is outside the value-object profile: %s", param.Name)
		}
		for _, span := range param.Uses {
			if err := p.add(fn.Source, span, p.parameterName+"."+capitalize(param.Name), false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *sourceFix) insertValueType(fn pdg.SourceFunction) {
	var fields []string
	for _, position := range p.positions[fn.ID] {
		param := fn.ParametersList[position]
		fields = append(fields, capitalize(param.Name)+" "+param.TypeSyntax)
	}
	snapshot, _ := p.context.Snapshot(fn.Source)
	end := uint32(len(snapshot.Content))
	p.edits[fn.Source] = append(p.edits[fn.Source], pdg.SourceEdit{Range: pdg.ByteRange{Start: end, End: end}, Replacement: "\n\ntype " + p.typeName + " struct {\n" + strings.Join(fields, "\n") + "\n}\n"})
}

func (p *sourceFix) add(ref pdg.Ref, span pdg.ByteRange, replacement string, protectComments bool) error {
	path, _, ok := p.context.Source(ref)
	if !ok {
		return fmt.Errorf("source snapshot is missing")
	}
	if protectComments && p.context.HasComment(ref, span) {
		return unsupportedFix("rewrite requires unsupported comment relocation in %s", path)
	}
	p.edits[ref] = append(p.edits[ref], pdg.SourceEdit{Range: span, Replacement: replacement})
	return nil
}

func (p *sourceFix) calls() error {
	refs := append([]pdg.SourceReference(nil), p.context.References...)
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].Range.End-refs[i].Range.Start < refs[j].Range.End-refs[j].Range.Start
	})
	for _, ref := range refs {
		fn, ok := p.targets[ref.Target]
		if !ok {
			continue
		}
		if !ref.Direct || ref.Variadic {
			return unsupportedFix("unsupported function-value or variadic reference to %s", fn.Name)
		}
		if err := p.call(fn, ref); err != nil {
			return err
		}
	}
	return nil
}

func (p *sourceFix) call(fn pdg.SourceFunction, ref pdg.SourceReference) error {
	if len(ref.Arguments) != len(fn.ParametersList) {
		return unsupportedFix("unsupported expanded arguments to %s", fn.Name)
	}
	path, content, _ := p.context.Source(ref.Source)
	snapshot := pdg.SourceSnapshot{Path: path, Content: content}
	if p.context.HasComment(ref.Source, ref.Range) {
		return unsupportedFix("call rewrite requires unsupported comment relocation in %s", snapshot.Path)
	}
	arguments, remaining, err := p.callArguments(snapshot, ref)
	if err != nil {
		return err
	}
	replacement := p.valueCall(fn, arguments)
	if p.method {
		replacement = "(" + fn.ParametersList[0].TypeSyntax + ")(" + arguments[0] + ")." + fn.Name + "(" + strings.Join(arguments[1:], ", ") + ")"
	}
	p.edits[ref.Source] = append(remaining, pdg.SourceEdit{Range: ref.Range, Replacement: replacement})
	return nil
}

// Fold contained edits into each argument before replacing its enclosing call.
// This supports nested calls and resolved parameter uses without stale offsets.
func (p *sourceFix) callArguments(snapshot pdg.SourceSnapshot, ref pdg.SourceReference) ([]string, []pdg.SourceEdit, error) {
	var arguments []string
	remaining := p.edits[ref.Source]
	for _, span := range ref.Arguments {
		text, rest, err := renderArgument(snapshot, span, remaining)
		if err != nil {
			return nil, nil, err
		}
		arguments = append(arguments, text)
		remaining = rest
	}
	return arguments, remaining, nil
}

func (p *sourceFix) valueCall(fn pdg.SourceFunction, arguments []string) string {
	positions := p.positions[fn.ID]
	first, last := positions[0], positions[len(positions)-1]
	var fields []string
	for _, position := range positions {
		fields = append(fields, capitalize(fn.ParametersList[position].Name)+": "+arguments[position])
	}
	args := append([]string(nil), arguments[:first]...)
	args = append(args, p.typeName+"{"+strings.Join(fields, ", ")+"}")
	args = append(args, arguments[last+1:]...)
	return fn.Name + "(" + strings.Join(args, ", ") + ")"
}

func (p *sourceFix) render() (map[string][]byte, error) {
	overlay := make(map[string][]byte)
	for ref, edits := range p.edits {
		path, content, _ := p.context.Source(ref)
		snapshot := pdg.SourceSnapshot{Path: path, Content: content}
		text, err := pdg.ApplySourceEdits(snapshot, edits)
		if err != nil {
			return nil, err
		}
		formatted, err := format.Source([]byte(text))
		if err != nil {
			return nil, err
		}
		overlay[snapshot.Path] = formatted
	}
	return overlay, nil
}

// PlanSignatureFix uses the typed source inventory from PDG extraction.
// It never finds callers by spelling or reparses old code to resolve bindings.
func PlanSignatureFix(spec *patterns.FixSpec) (map[string][]byte, error) {
	plan, err := newSourceFix(spec)
	if err != nil {
		return nil, err
	}
	if err := checkSignatureCoverage(plan.context, plan.targets); err != nil {
		return nil, err
	}
	if err := plan.declarations(); err != nil {
		return nil, err
	}
	if err := plan.calls(); err != nil {
		return nil, err
	}
	return plan.render()
}

func astExported(name string) bool {
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
}

func signaturePositions(fn pdg.SourceFunction, spec *patterns.FixSpec) []int {
	if spec.Kind == patterns.AnemicModel {
		return receiverPosition(fn, spec.Params["type_name"])
	}
	group := strings.Split(spec.Params["group"], ",")
	selected := make(map[string]bool)
	for _, key := range group {
		selected[key] = true
	}
	var positions []int
	for i, param := range fn.ParametersList {
		if selected[param.Name+":"+param.TypeSyntax] {
			positions = append(positions, i)
		}
	}
	if !completeParameterGroup(positions, group) {
		return nil
	}
	return positions
}

func hasBinding(bindings []string, name string) bool {
	for _, binding := range bindings {
		if binding == name {
			return true
		}
	}
	return false
}

func contiguousPositions(positions []int) bool {
	for i := 1; i < len(positions); i++ {
		if positions[i] != positions[i-1]+1 {
			return false
		}
	}
	return true
}

func rewriteParameters(fn pdg.SourceFunction, positions []int, name, typeName string) string {
	var fields []string
	first, last := positions[0], positions[len(positions)-1]
	for i, param := range fn.ParametersList {
		if i == first {
			fields = append(fields, name+" "+typeName)
		}
		if i < first || i > last {
			fields = append(fields, param.Name+" "+param.TypeSyntax)
		}
	}
	return "(" + strings.Join(fields, ", ") + ")"
}

func methodParameters(fn pdg.SourceFunction) string {
	var fields []string
	for _, param := range fn.ParametersList[1:] {
		fields = append(fields, param.Name+" "+param.TypeSyntax)
	}
	return "(" + strings.Join(fields, ", ") + ")"
}

func renderArgument(snapshot pdg.SourceSnapshot, span pdg.ByteRange, edits []pdg.SourceEdit) (string, []pdg.SourceEdit, error) {
	var contained, remaining []pdg.SourceEdit
	for _, edit := range edits {
		if edit.Range.Start >= span.Start && edit.Range.End <= span.End {
			edit.Range.Start -= span.Start
			edit.Range.End -= span.Start
			contained = append(contained, edit)
		} else {
			remaining = append(remaining, edit)
		}
	}
	fragment := pdg.SourceSnapshot{Path: snapshot.Path, Content: snapshot.Content[span.Start:span.End]}
	text, err := pdg.ApplySourceEdits(fragment, contained)
	return text, remaining, err
}

func selectedSourceFunction(fn pdg.SourceFunction, spec *patterns.FixSpec, ref pdg.Ref) bool {
	if spec.Kind == patterns.AnemicModel || len(spec.Targets) == 0 {
		return fn.Source == ref
	}
	for _, id := range spec.Targets {
		if id == fn.ID {
			return true
		}
	}
	return false
}

func signatureTarget(fn pdg.SourceFunction, spec *patterns.FixSpec, ref pdg.Ref) bool {
	if fn.Method || fn.Generic || fn.Variadic || astExported(fn.Name) {
		return false
	}
	return selectedSourceFunction(fn, spec, ref)
}

func receiverPosition(fn pdg.SourceFunction, typeName string) []int {
	if len(fn.ParametersList) == 0 {
		return nil
	}
	if strings.TrimPrefix(fn.ParametersList[0].TypeSyntax, "*") != typeName {
		return nil
	}
	return []int{0}
}

func completeParameterGroup(positions []int, group []string) bool {
	return len(positions) >= 2 && len(positions) == len(group)
}
