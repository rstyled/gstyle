package gopatterns

import (
	"errors"
	"slices"
)

var errFixScopeChanged = errors.New("fix package inventory changed")

func selectedSourceQueries(records map[string]sourcePackageRecord, affected map[string]bool) map[string]bool {
	if affected == nil {
		return nil
	}
	selected := make(map[string]bool)
	for name := range affected {
		selected[records[name].query] = true
	}
	return selected
}

func selectSourceFamilies(families []sourceFamily, selected map[string]bool) []sourceFamily {
	if selected == nil {
		return families
	}
	var result []sourceFamily
	for _, family := range families {
		if selected[family.query] {
			result = append(result, family)
		}
	}
	return result
}

func verifySourceFamilies(families []sourceFamily, request sourceVisit) error {
	if request.selected == nil {
		return nil
	}
	if len(families) != len(request.known) {
		return errFixScopeChanged
	}
	for _, family := range families {
		record, ok := request.known[family.name]
		if !ok {
			return errFixScopeChanged
		}
		files := slices.Clone(family.files)
		slices.Sort(files)
		files = slices.Compact(files)
		if !slices.Equal(files, record.files) {
			return errFixScopeChanged
		}
	}
	return nil
}
