package gopatterns

import (
	"path/filepath"
	"testing"
)

func TestAuditDomainServiceShadow(t *testing.T) {
	pkg := loadTestPackage(t, `package p;type Account struct{V int};type Balance struct{V int};func f(a Account,b Balance)int{b2:=b;_=b2;{b:=struct{V int}{1};return a.V+b.V}}`)
	root := filepath.Dir(pkg.GoFiles[0])
	if got := findDomainServices(pkg, root); len(got) != 0 {
		t.Fatalf("shadow counted as second domain type: %+v", got)
	}
}
func TestAuditAggregateParentheses(t *testing.T) {
	src := `package p;type A struct{V int};type B struct{V int};func f(a *A,b *B){(a.V)=1;(b.V)=2}`
	pkg := loadTestPackage(t, src)
	for _, f := range pkg.Syntax {
		for _, d := range f.Decls {
			if fn, ok := freeFuncDecl(d); ok {
				if got := FindAggregateMods(fn, pkg.TypesInfo); len(got) != 2 {
					t.Fatalf("parenthesized writes: got %d types,want 2", len(got))
				}
			}
		}
	}
}
func TestAuditMutableIdentityParentheses(t *testing.T) {
	fn, fs := parseTestFunc(t, `package p;type A struct{ID int};func update(a *A){(a.ID)=1}`)
	if got := FindMutableIdentities(fn, fs); len(got) != 1 {
		t.Fatalf("parenthesized ID write: got %d findings,want 1", len(got))
	}
}
func TestAuditRepositoryImportIdentity(t *testing.T) {
	cases := []struct {
		name, src string
		want      int
	}{
		{"math called sql", `package p;import sql "math";func f(x float64)float64{return sql.Abs(x)}`, 0},
		{"sql called data", `package p;import data "database/sql";func f(){h,_:=data.Open("missing","input");_=h}`, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pkg := loadTestPackage(t, tc.src)
			for _, f := range pkg.Syntax {
				for _, d := range f.Decls {
					if fn, ok := freeFuncDecl(d); ok {
						got := FindDbCalls(fn, pkg.Fset, pkg.TypesInfo, "domain/use.go")
						if len(got) != tc.want {
							t.Fatalf("got %+v,want %d db calls", got, tc.want)
						}
					}
				}
			}
		})
	}
}

func TestAuditMissingIdentityGroupedParams(t *testing.T) {
	src := `package p
 type Record struct{N int;Name string}
 func change(r *Record){r.N++}
 func same(a,b Record)bool{return a.N==b.N && a.Name==b.Name}
 `
	if !missingIdentityNames(t, src)["Record"] {
		t.Fatal("three parameter uses become two when names share a type field")
	}
}
