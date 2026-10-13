package gopatterns

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

// PlanLocalFixes renders every candidate against one immutable extraction
// revision. Independent edits merge; overlapping candidates wait for the next
// phase cycle. A rewritten buffer never authorizes stale syntax facts.
func PlanLocalFixes(specs []*patterns.FixSpec, source pdg.SourceSnapshot) ([]byte, int, []string, error) {
	var edits []pdg.SourceEdit
	var skipped []string
	for _, spec := range specs {
		edit, reason, err := localFixDecision(spec, source, edits)
		if err != nil {
			return nil, 0, nil, err
		}
		if reason != "" {
			skipped = append(skipped, reason)
			continue
		}
		if edit != nil {
			edits = append(edits, *edit)
		}
	}
	result, err := pdg.ApplySourceEdits(source, edits)
	return []byte(result), len(edits), skipped, err
}

func localFixDecision(spec *patterns.FixSpec, source pdg.SourceSnapshot, edits []pdg.SourceEdit) (*pdg.SourceEdit, string, error) {
	if err := checkLocalInventory(spec, source); err != nil {
		return nil, "", err
	}
	edit, reason, err := localFixEdit(spec, source)
	if err != nil {
		return nil, "", err
	}
	if edit == nil {
		return nil, reason, nil
	}
	if localEditsOverlap(edits, *edit) {
		return nil, "overlapping fix deferred to the next source revision", nil
	}
	return edit, "", nil
}

func checkLocalInventory(spec *patterns.FixSpec, source pdg.SourceSnapshot) error {
	if !hasFixContext(spec) {
		return fmt.Errorf("local fix requires extraction evidence")
	}
	path := fixSourcePath(spec.Context.Root, spec.File)
	if path != source.Path {
		return fmt.Errorf("local fix refers to a different source file")
	}
	if err := spec.Context.Check(path, []byte(source.Content)); err != nil {
		return err
	}
	if source.SyntaxProfile != sourceSyntaxProfile || source.Syntax == "" {
		return fmt.Errorf("source syntax inventory is missing: %s", path)
	}
	return nil
}

func localFixEdit(spec *patterns.FixSpec, source pdg.SourceSnapshot) (*pdg.SourceEdit, string, error) {
	if sourceHasLineDirectives(source) {
		return nil, "source line directives prevent a safe rewrite", nil
	}

	before := []byte(source.Content)
	after, err := ApplyFix(spec, before)
	if errors.Is(err, errSourceSyntax) {
		return nil, "", err
	}
	if err != nil {
		return nil, fmt.Sprintf("%s: %v", spec.Kind, err), nil
	}
	if bytes.Equal(before, after) {
		return nil, "", nil
	}
	start := commonFixPrefix(before, after)
	end, replacementEnd := commonFixSuffix(before, after, start)
	return &pdg.SourceEdit{Range: pdg.ByteRange{Start: uint32(start), End: uint32(end)}, Replacement: string(after[start:replacementEnd])}, "", nil
}

func commonFixPrefix(before, after []byte) int {
	start := 0
	for start < len(before) && start < len(after) && before[start] == after[start] {
		start++
	}
	return start
}

func commonFixSuffix(before, after []byte, start int) (int, int) {
	left, right := len(before), len(after)
	for left > start && right > start && before[left-1] == after[right-1] {
		left--
		right--
	}
	return left, right
}

func localEditsOverlap(edits []pdg.SourceEdit, next pdg.SourceEdit) bool {
	for _, edit := range edits {
		if edit.Range.Start == next.Range.Start {
			return true
		}
		if edit.Range.Start < next.Range.End && next.Range.Start < edit.Range.End {
			return true
		}
	}
	return false
}

func hasFixContext(spec *patterns.FixSpec) bool { return spec != nil && spec.Context != nil }

func sourceHasLineDirectives(source pdg.SourceSnapshot) bool {
	for _, comment := range source.Comments {
		text := source.Content[comment.Start:comment.End]
		if strings.HasPrefix(text, "//line ") || strings.HasPrefix(text, "/*line ") {
			return true
		}
	}
	return false
}
