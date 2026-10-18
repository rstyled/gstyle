package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"go/types"
	"strconv"
)

// Record exact primitive types. Named types can implement fmt.Formatter, so
// their underlying primitive type does not establish decimal output.
func (b *irBuilder) basicSourceFact(expr ast.Expr, attrs *pdg.Attributes) {
	typ := b.info.TypeOf(expr)
	if typ == nil {
		return
	}
	basic, ok := types.Unalias(typ).(*types.Basic)
	if !ok {
		return
	}
	attrs.Extensions = append(attrs.Extensions, b.pool.SingletonExtension("gstyle.value:basic", strconv.Quote(basic.Name()))...)
}
