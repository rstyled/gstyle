package pdg

import (
	"fmt"
	"sort"
)

// SourceEdit replaces bytes in an original snapshot. The source identity is
// checked at the IO boundary; ranges always refer to this same revision.
type SourceEdit struct {
	Range       ByteRange
	Replacement string
}

// ApplySourceEdits preserves all bytes outside the edits. Invalid or
// overlapping ranges fail before any result is returned.
func ApplySourceEdits(source SourceSnapshot, edits []SourceEdit) (string, error) {
	ordered := append([]SourceEdit(nil), edits...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Range.Start < ordered[j].Range.Start })
	if err := checkSourceEdits(source, ordered); err != nil {
		return "", err
	}
	result := source.Content
	for i := len(ordered) - 1; i >= 0; i-- {
		edit := ordered[i]
		result = result[:edit.Range.Start] + edit.Replacement + result[edit.Range.End:]
	}
	return result, nil
}

func checkSourceEdits(source SourceSnapshot, edits []SourceEdit) error {
	var previous uint32
	for i, edit := range edits {
		span := edit.Range
		if span.Start > span.End || uint64(span.End) > uint64(len(source.Content)) {
			return fmt.Errorf("edit range is outside source: %s", source.Path)
		}
		if sourceEditsOverlap(edits, i, previous) {
			return fmt.Errorf("source edits overlap: %s", source.Path)
		}
		previous = span.End
	}
	return nil
}

func sourceEditsOverlap(edits []SourceEdit, index int, previousEnd uint32) bool {
	span := edits[index].Range
	if span.Start < previousEnd {
		return true
	}
	return index > 0 && span.Start == edits[index-1].Range.Start
}
