package gopatterns

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
)

func parameterizationImportRisk(pair *parameterizationPair) string {
	if pair.first == pair.second {
		return ""
	}
	if !knownSourceImports(pair.first) || !knownSourceImports(pair.second) {
		return "typed import bindings are unavailable"
	}
	if !matchingUsedImports(pair.first, pair.second, nodeByteRange(pair.first.fset, pair.a.Body), nodeByteRange(pair.first.fset, pair.b.Body)) || !matchingUsedImports(pair.second, pair.first, nodeByteRange(pair.first.fset, pair.b.Body), nodeByteRange(pair.first.fset, pair.a.Body)) {
		return "source bodies use different imported bindings"
	}
	return ""
}

func knownSourceImports(source *parameterizationSource) bool {
	return len(source.file.Imports) == len(source.snapshot.Imports)
}

func matchingUsedImports(source, other *parameterizationSource, declaration, otherBody pdg.ByteRange) bool {
	bindings := map[string]string{}
	for _, item := range other.snapshot.Imports {
		if importUsedWithin(item, otherBody) {
			bindings[item.Path] = item.Path
		}
	}
	for _, item := range source.snapshot.Imports {
		if item.Name == "." {
			return false
		}
		if requiredImportMissing(item, declaration, bindings) {
			return false
		}
	}
	return true
}

func importUsedWithin(item pdg.SourceImport, span pdg.ByteRange) bool {
	for _, use := range item.Uses {
		if sourceRangeContains(span, use) {
			return true
		}
	}
	return false
}

func importUsesOnlyWithin(item pdg.SourceImport, span pdg.ByteRange) bool {
	if len(item.Uses) == 0 {
		return false
	}
	for _, use := range item.Uses {
		if !sourceRangeContains(span, use) {
			return false
		}
	}
	return true
}

// The helper file already imports these packages. Removing the duplicate
// declarations preserves package initialization and leaves unrelated uses alone.
func parameterizationImportEdits(source *parameterizationSource, body pdg.ByteRange) ([]pdg.SourceEdit, error) {
	var edits []pdg.SourceEdit
	seen := map[pdg.ByteRange]bool{}
	for _, item := range source.snapshot.Imports {
		if !importUsesOnlyWithin(item, body) {
			continue
		}
		edit := unusedParameterizationImport(source.snapshot, item, body)
		if !seen[edit.Range] {
			edits = append(edits, edit)
			seen[edit.Range] = true
		}
	}
	return edits, nil
}

func unusedParameterizationImport(snapshot pdg.SourceSnapshot, item pdg.SourceImport, body pdg.ByteRange) pdg.SourceEdit {
	if importHasOwnedComments(snapshot, item) {
		replacement := "_"
		if item.Binding.Start == item.Binding.End {
			replacement += " "
		}
		return pdg.SourceEdit{Range: item.Binding, Replacement: replacement}
	}
	if !item.Grouped || wholeImportGroupUnused(snapshot.Imports, item.Declaration, body) {
		return pdg.SourceEdit{Range: item.Declaration}
	}
	return pdg.SourceEdit{Range: importSpecRemovalRange(snapshot, item.Spec)}
}

func importHasOwnedComments(snapshot pdg.SourceSnapshot, item pdg.SourceImport) bool {
	if snapshot.HasComment(item.Declaration) {
		return true
	}
	for _, owner := range snapshot.Owners {
		if owner.Owner == "import" {
			return true
		}
	}
	return false
}

func wholeImportGroupUnused(imports []pdg.SourceImport, declaration, body pdg.ByteRange) bool {
	for _, item := range imports {
		if item.Declaration == declaration && !importUsesOnlyWithin(item, body) {
			return false
		}
	}
	return true
}

func validateParameterizationImports(first, second *parameterizationSource, a, bSpan pdg.ByteRange) error {
	if !knownSourceImports(first) || !knownSourceImports(second) {
		return fmt.Errorf("typed import evidence is missing")
	}
	if !matchingUsedImports(first, second, a, bSpan) || !matchingUsedImports(second, first, bSpan, a) {
		return fmt.Errorf("parameterization import bindings changed after proof")
	}
	return nil
}

// Only compiler-resolved package uses are normalized. Shadowed locals keep
// their own bindings, and wrappers retain their original type spelling.
func normalizeParameterizationImports(pair *parameterizationPair) {
	names := map[string]string{}
	for _, item := range pair.first.snapshot.Imports {
		if item.Name != "_" && item.Name != "." {
			names[item.Path] = item.Name
		}
	}
	normalizeBodyImports(pair.first, pair.a.Body, names)
	normalizeBodyImports(pair.second, pair.b.Body, names)
}

func normalizeBodyImports(source *parameterizationSource, body *ast.BlockStmt, names map[string]string) {
	uses := map[uint32]string{}
	for _, item := range source.snapshot.Imports {
		name := names[item.Path]
		if name == "" {
			continue
		}
		for _, use := range item.Uses {
			uses[use.Start] = name
		}
	}
	ast.Inspect(body, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		offset := uint32(source.fset.PositionFor(ident.Pos(), false).Offset)
		if name := uses[offset]; name != "" {
			ident.Name = name
		}
		return true
	})
}

func requiredImportMissing(item pdg.SourceImport, body pdg.ByteRange, bindings map[string]string) bool {
	return importUsedWithin(item, body) && bindings[item.Path] != item.Path
}

func importSpecRemovalRange(snapshot pdg.SourceSnapshot, span pdg.ByteRange) pdg.ByteRange {
	end := int(span.End)
	for end < len(snapshot.Content) && (snapshot.Content[end] == ' ' || snapshot.Content[end] == '\t') {
		end++
	}
	if end < len(snapshot.Content) && snapshot.Content[end] == ';' {
		span.End = uint32(end + 1)
	}
	return span
}
