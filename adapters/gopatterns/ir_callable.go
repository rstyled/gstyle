package gopatterns

import (
	"encoding/json"
	"go/ast"
	"go/types"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

// Exact callable types live in interned extension values, separate from the
// coarse matching labels. Unknown and generic types do not authorize a fix.
func (b *irBuilder) callableExtensions(expr ast.Expr) []pdg.Extension {
	signature, ok := b.info.TypeOf(expr).(*types.Signature)
	if !ok || signature.TypeParams().Len() != 0 {
		return nil
	}
	normalized := types.NewSignatureType(nil, nil, nil, unnamedTuple(signature.Params()), unnamedTuple(signature.Results()), signature.Variadic())
	source := types.TypeString(normalized, b.typeQualifier)
	if strings.Contains(source, "__gstyle_unavailable_import__") {
		source = ""
	}
	return []pdg.Extension{
		{Name: b.pool.Text("gstyle:callable_type"), Value: b.pool.Text(callableJSON(irTypeString(normalized)))},
		{Name: b.pool.Text("gstyle:callable_source_type"), Value: b.pool.Text(callableJSON(source))},
	}
}

func (b *irBuilder) implicitReceiver(expr ast.Expr) bool {
	selector, ok := unparen(expr).(*ast.SelectorExpr)
	if !ok {
		return false
	}
	selection := b.info.Selections[selector]
	return selection != nil && selection.Kind() == types.MethodVal
}

func unnamedTuple(tuple *types.Tuple) *types.Tuple {
	vars := make([]*types.Var, tuple.Len())
	for i := range vars {
		vars[i] = types.NewVar(0, nil, "", tuple.At(i).Type())
	}
	return types.NewTuple(vars...)
}

func callableQualifier(pkg *packages.Package, fn *ast.FuncDecl) types.Qualifier {
	for _, file := range pkg.Syntax {
		if file.Pos() <= fn.Pos() && fn.End() <= file.End() {
			return sourceTypeQualifier(pkg.TypesInfo, file, pkg.PkgPath)
		}
	}
	return func(pkg *types.Package) string { return pkg.Name() }
}

func sourceTypeQualifier(info *types.Info, file *ast.File, current string) types.Qualifier {
	names := sourceImportNames(info, file)
	return func(pkg *types.Package) string {
		if pkg.Path() == current {
			return ""
		}
		name, ok := names[pkg.Path()]
		if !ok {
			return "__gstyle_unavailable_import__"
		}
		if name == "." {
			return ""
		}
		return name
	}
}

func callableJSON(text string) string {
	// A string contains no unsupported JSON values; Marshal cannot fail.
	encoded, _ := json.Marshal(text)
	return string(encoded)
}

func sourceImportNames(info *types.Info, file *ast.File) map[string]string {
	names := make(map[string]string)
	for _, spec := range file.Imports {
		object := info.Implicits[spec]
		if spec.Name != nil {
			object = info.Defs[spec.Name]
		}
		if name, ok := object.(*types.PkgName); ok {
			names[name.Imported().Path()] = name.Name()
		}
	}
	return names
}
