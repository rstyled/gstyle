package gopatterns

import (
	"encoding/json"
	"go/types"

	"golang.org/x/tools/go/packages"
)

func (b *irBuilder) buildEvidence(pkg *packages.Package) {
	if pkg.Types != nil && pkg.Types.GoVersion() != "" {
		b.graph.Producer.LanguageVersion = b.pool.Known(pkg.Types.GoVersion())
	}
	if pkg.TypesSizes == nil {
		return
	}
	width := pkg.TypesSizes.Sizeof(types.Typ[types.Uintptr]) * 8
	configuration, _ := json.Marshal(struct {
		PointerWidth int64  `json:"pointer_width"`
		Package      string `json:"package"`
	}{width, pkg.PkgPath})
	b.graph.Producer.Configuration = b.pool.Text(string(configuration))
}
