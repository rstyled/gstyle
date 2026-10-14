package gopatterns

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

type sharedValueGroup struct {
	caller  pdg.SourceFunction
	slots   []int
	fields  []string
	calls   []pdg.SourceReference
	targets map[string]bool
}

// A useful group has an owner that forwards the same stable input bindings
// to multiple operations. Matching signatures alone cannot authorize a type.
func (p *sourceFix) sharedValueGroups() ([]sharedValueGroup, error) {
	groups := map[string]*sharedValueGroup{}
	callers := sourceCallers(p.context)
	for _, ref := range p.context.References {
		_, ok := p.targets[ref.Target]
		if !forwardingCall(ok, ref) {
			continue
		}
		p.recordValueForwarding(groups, callers[ref.Source], ref)
	}
	var result []sharedValueGroup
	strong := false
	for _, group := range groups {
		result = append(result, *group)
		if len(group.targets) >= 2 {
			strong = true
		}
	}
	if !strong {
		return nil, unsupportedFix("matching parameter shapes have no proven shared-value flow")
	}
	sort.Slice(result, func(i, j int) bool { return groupKey(result[i]) < groupKey(result[j]) })
	return result, nil
}

func (p *sourceFix) recordValueForwarding(groups map[string]*sharedValueGroup, callers []pdg.SourceFunction, ref pdg.SourceReference) {
	target := p.targets[ref.Target]
	for _, caller := range callers {
		path, _, _ := p.context.Source(caller.Source)
		if strings.HasSuffix(filepath.Base(path), "_test.go") {
			continue
		}
		if !callerOwnsCall(caller, target, ref) {
			continue
		}
		slots := forwardedParameterSlots(caller, ref, p.positions[target.ID])
		if !matchedGroupTypes(caller, target, slots, p.positions[target.ID]) {
			return
		}
		group := sharedValueGroup{caller: caller, slots: slots, fields: targetGroupFields(target, p.positions[target.ID])}
		key := groupKey(group)
		if groups[key] == nil {
			group.targets = map[string]bool{}
			groups[key] = &group
		}
		groups[key].calls = append(groups[key].calls, ref)
		groups[key].targets[ref.Target] = true
		return
	}
}

func (p *sourceFix) propagateValueGroups(groups []sharedValueGroup) error {
	occupied := map[string]bool{}
	for _, group := range groups {
		binding := p.existingGroupBinding(group)
		if binding == "" {
			if len(group.targets) < 2 {
				continue
			}
			binding = freshGroupBinding(group.caller, p.parameterName, occupied)
			if err := p.insertSharedGroup(group, binding); err != nil {
				return err
			}
		}
		occupied[group.caller.ID+"|"+binding] = true
		for _, call := range group.calls {
			p.sharedCalls[valueCallKey(call)] = binding
		}
	}
	return nil
}

func (p *sourceFix) existingGroupBinding(group sharedValueGroup) string {
	positions, ok := p.positions[group.caller.ID]
	if !ok || len(positions) != len(group.slots) {
		return ""
	}
	for i, slot := range group.slots {
		if !slices.Contains(positions, slot) || capitalize(group.caller.ParametersList[slot].Name) != group.fields[i] {
			return ""
		}
	}
	return p.parameterName
}

func (p *sourceFix) insertSharedGroup(group sharedValueGroup, binding string) error {
	var fields []string
	for i, slot := range group.slots {
		param := group.caller.ParametersList[slot]
		value := param.Name
		if slices.Contains(p.positions[group.caller.ID], slot) {
			value = p.parameterName + "." + capitalize(param.Name)
		}
		fields = append(fields, group.fields[i]+": "+value)
	}
	text := fmt.Sprintf("\n%s := %s{%s}\n", binding, p.typeName, strings.Join(fields, ", "))
	start := group.caller.Body.Start + 1
	span := pdg.ByteRange{Start: start, End: start}
	for i, edit := range p.edits[group.caller.Source] {
		if edit.Range == span {
			p.edits[group.caller.Source][i].Replacement += text
			return nil
		}
	}
	return p.add(group.caller.Source, span, text, false)
}

func forwardedParameterSlots(caller pdg.SourceFunction, ref pdg.SourceReference, positions []int) []int {
	var slots []int
	for _, position := range positions {
		if position >= len(ref.Arguments) {
			return nil
		}
		slot := stableParameterAt(caller, ref.Arguments[position])
		if slot < 0 {
			return nil
		}
		slots = append(slots, slot)
	}
	return slots
}

func stableParameterAt(fn pdg.SourceFunction, span pdg.ByteRange) int {
	for i, param := range fn.ParametersList {
		if !param.FlowKnown || !param.Stable {
			continue
		}
		for _, use := range param.Uses {
			if use == span {
				return i
			}
		}
	}
	return -1
}

func targetGroupFields(fn pdg.SourceFunction, positions []int) []string {
	var fields []string
	for _, position := range positions {
		fields = append(fields, capitalize(fn.ParametersList[position].Name))
	}
	return fields
}

func groupKey(group sharedValueGroup) string {
	var parts []string
	for i, slot := range group.slots {
		parts = append(parts, group.fields[i]+":"+strconv.Itoa(slot))
	}
	sort.Strings(parts)
	return group.caller.ID + "|" + strings.Join(parts, ",")
}

func freshGroupBinding(fn pdg.SourceFunction, name string, occupied map[string]bool) string {
	candidate := name
	for suffix := 1; hasBinding(fn.Bindings, candidate) || occupied[fn.ID+"|"+candidate]; suffix++ {
		candidate = name + strconv.Itoa(suffix)
	}
	return candidate
}

func valueCallKey(ref pdg.SourceReference) string {
	return fmt.Sprintf("%d:%d:%d", ref.Source, ref.Range.Start, ref.Range.End)
}

// Weak candidates remain visible with an explicit reason and no automatic fix.
func VerifyValueGroups(report *patterns.PatternsReport, extraction *Extraction) {
	for i := range report.Candidates {
		candidate := &report.Candidates[i]
		if candidate.Kind != patterns.ValueObject || candidate.FixSpec == nil {
			continue
		}
		spec := *candidate.FixSpec
		spec.Context = extraction.Sources
		for _, site := range candidate.Sites {
			spec.Targets = append(spec.Targets, site.ID)
		}
		reason := valueGroupReason(&spec)
		if reason != "" {
			candidate.CounterEvidence = append(candidate.CounterEvidence, reason)
			candidate.Inference = "matching parameter shapes suggest a possible group; shared-value flow is not established"
			candidate.FixSpec = nil
		}
	}
}

func valueGroupReason(spec *patterns.FixSpec) string {
	plan, err := newSourceFix(spec)
	if err != nil {
		return err.Error()
	}
	_, err = plan.sharedValueGroups()
	if err != nil {
		return err.Error()
	}
	return ""
}

func forwardingCall(targetKnown bool, ref pdg.SourceReference) bool {
	return targetKnown && ref.Direct && !ref.Variadic
}

func callerOwnsCall(caller, target pdg.SourceFunction, ref pdg.SourceReference) bool {
	return caller.Source == ref.Source && caller.Package == target.Package && sourceRangeContains(caller.Body, ref.Range)
}

func groupTypesMatch(caller, target pdg.SourceFunction, slots, positions []int) bool {
	for i, slot := range slots {
		if caller.ParametersList[slot].Type != target.ParametersList[positions[i]].Type {
			return false
		}
	}
	return true
}

func matchedGroupTypes(caller, target pdg.SourceFunction, slots, positions []int) bool {
	return slots != nil && groupTypesMatch(caller, target, slots, positions)
}

func sourceCallers(context *pdg.SourceContext) map[pdg.Ref][]pdg.SourceFunction {
	callers := map[pdg.Ref][]pdg.SourceFunction{}
	for _, fn := range context.Functions {
		callers[fn.Source] = append(callers[fn.Source], fn)
	}
	return callers
}
