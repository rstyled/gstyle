package gorisk

import (
	"fmt"
	"go/types"
	"slices"
	"strings"
)

func apiLimits(base, head map[string]*types.Package) []string {
	var limits []string
	for name, before := range base {
		if after := head[name]; after != nil {
			limits = append(limits, packageConstraintLimits(before, after)...)
		}
	}
	slices.Sort(limits)
	return limits
}

func packageConstraintLimits(base, head *types.Package) []string {
	var limits []string
	for _, name := range base.Scope().Names() {
		before, after := base.Scope().Lookup(name), head.Scope().Lookup(name)
		if !before.Exported() || after == nil {
			continue
		}
		if constraintFingerprint(before.Type()) != constraintFingerprint(after.Type()) {
			limits = append(limits, base.Path()+"."+name+": generic constraints changed; source compatibility requires review")
		}
	}
	return limits
}

func constraintFingerprint(typ types.Type) string {
	var parts []string
	if parameterized, ok := typ.(interface{ TypeParams() *types.TypeParamList }); ok {
		params := parameterized.TypeParams()
		for i := 0; i < params.Len(); i++ {
			parts = append(parts, fmt.Sprintf("%d:%s", i, constraintShape(params.At(i).Constraint(), map[types.Type]bool{})))
		}
	}
	if iface, ok := typ.Underlying().(*types.Interface); ok && !iface.IsMethodSet() {
		parts = append(parts, constraintShape(iface, map[types.Type]bool{}))
	}
	return strings.Join(parts, "|")
}

func constraintShape(typ types.Type, seen map[types.Type]bool) string {
	label := types.TypeString(typ, func(pkg *types.Package) string { return pkg.Path() })
	if seen[typ] {
		return label
	}
	seen[typ] = true
	iface, ok := typ.Underlying().(*types.Interface)
	if !ok {
		return label
	}
	parts := []string{label}
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		parts = append(parts, constraintShape(iface.EmbeddedType(i), seen))
	}
	slices.Sort(parts)
	return strings.Join(parts, "|")
}
