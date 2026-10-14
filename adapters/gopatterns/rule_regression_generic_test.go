package gopatterns

import (
	"bytes"
	"github.com/rstyled/gstyle/domain/patterns"
	"testing"
)

func TestAuditGenericLiteralNotAType(t *testing.T) {
	src := []byte(`package p
var log string
func labelInt(x int)int{log="int";return x}
func labelFloat(x float64)float64{log="float64";return x}
`)
	out, err := applyProjectedFix(t, &patterns.FixSpec{Kind: patterns.GenericFn, File: "p.go", Line: 3}, src)
	if err != nil {
		return
	}
	if bytes.Contains(out, []byte(`log="T"`)) || bytes.Contains(out, []byte(`log = "T"`)) {
		t.Fatalf("string literal replaced as a type:\n%s", out)
	}
}
