package gopatterns

import (
	"strings"
	"testing"
)

func TestUncheckedOutputEdges(t *testing.T) {
	for _, tc := range []struct {
		name, imports, decl, body string
		want                      int
	}{
		{"parenthesized call", "io;fmt", "", `(fmt.Fprintln(w,"text"))`, 1},
		{"if initializer", "io;fmt", "", `if n,_:=fmt.Fprintln(w,"text");n>0{}`, 1},
		{"forwarded as arguments", "io;fmt", `func consume(int,error){}`, `consume(fmt.Fprintln(w,"text"))`, 0},
		{"named but ignored error", "io;fmt", "", `_,err:=fmt.Fprintln(w,"text");_=err`, 0},
		{"Fprint", "io;fmt", "", `fmt.Fprint(w,"text")`, 1},
		{"Fprintf", "io;fmt", "", `fmt.Fprintf(w,"%s","text")`, 1},
		{"Fprintln", "io;fmt", "", `fmt.Fprintln(w,"text")`, 1},
		{"dot import", "io;.fmt", "", `Fprintln(w,"text")`, 1},
		{"parenthesized function", "io;fmt", "", `(fmt.Fprintln)(w,"text")`, 1},
		{"blank error", "io;fmt", "", `n,_:=fmt.Fprintln(w,"text");_=n`, 1},
		{"parenthesized blank error", "io;fmt", "", `_,(_)=fmt.Fprintln(w,"text")`, 1},

		{"nested parenthesized blank error", "io;fmt", "", `_,(((_)))=fmt.Fprintln(w,"text")`, 1},

		{"both blank", "io;fmt", "", `_,_=fmt.Fprintln(w,"text")`, 1},
		{"declared blank", "io;fmt", "", `var n,_=fmt.Fprintln(w,"text");_=n`, 1},
		{"checked assignment", "io;fmt", "", `_,err:=fmt.Fprintln(w,"text");if err!=nil{return err}`, 0},
		{"propagated tuple", "io;fmt", "", `returnOutput(w)`, 0},
		{"buffer pointer alias", "io;fmt;bytes", `type Output = *bytes.Buffer`, `var b bytes.Buffer;var out Output=&b;fmt.Fprintln(out,"text")`, 0},
		{"builder alias", "io;fmt;strings", `type Memory = strings.Builder`, `var b Memory;fmt.Fprintln(&b,"text")`, 0},
		{"interface hides buffer", "io;fmt;bytes", "", `var b bytes.Buffer;var out io.Writer=&b;fmt.Fprintln(out,"text")`, 1},
		{"custom buffer can fail", "io;fmt;bytes", `type Buffer struct{bytes.Buffer};func(*Buffer)Write([]byte)(int,error){return 0,io.ErrClosedPipe}`, `var b Buffer;fmt.Fprintln(&b,"text")`, 1},
		{"custom builder can fail", "io;fmt;strings", `type Builder struct{strings.Builder};func(*Builder)Write([]byte)(int,error){return 0,io.ErrClosedPipe}`, `var b Builder;fmt.Fprintln(&b,"text")`, 1},

		{"local function shadow", "io;fmt", `type Printer struct{Fprintln func(io.Writer,...any)(int,error)}`, `fmt:=Printer{Fprintln:fmt.Fprintln};fmt.Fprintln(w,"text")`, 0},
		{"indirect call excluded", "io;fmt", "", `print:=fmt.Fprintln;print(w,"text")`, 0},
		{"deferred call excluded", "io;fmt", "", `defer fmt.Fprintln(w,"text")`, 0},
		{"goroutine excluded", "io;fmt", "", `go fmt.Fprintln(w,"text")`, 0},
		{"nested closure excluded", "io;fmt", "", `_=func(){fmt.Fprintln(w,"text")}`, 0},
		{"outer survives closure", "io;fmt", "", `_=func(){fmt.Fprintln(w,"inner")};fmt.Fprintln(w,"outer")`, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			imports := uncheckedTestImports(tc.imports)
			source := `package p;import(` + imports + `)` + "\n" + tc.decl + "\n" + `func f(w io.Writer)error{` + tc.body + `;return nil};func returnOutput(w io.Writer)(int,error){return fmt.Fprintln(w,"ok")}`
			if tc.imports == "io;.fmt" {
				source = `package p;import("io";. "fmt");func f(w io.Writer)error{` + tc.body + `;return nil}`
			}
			file, fs, info, _ := typeCheck(t, source)
			sites := findErrorCheckSites(findFunc(file, "f"), fs, info)
			var count int
			for _, site := range sites {
				if site.DiscardedOutput {
					count++
				}
			}
			if count != tc.want {
				t.Fatalf("got %d unchecked writes,want %d; sites=%+v", count, tc.want, sites)
			}
		})
	}
}

func uncheckedTestImports(list string) string {
	var out []string
	for _, name := range strings.Split(list, ";") {
		out = append(out, `"`+name+`"`)
	}
	return strings.Join(out, ";")
}

// Cross the call, destination, and consumption forms instead of testing each
// feature only in isolation. The writer's concrete type controls exclusions.
func TestUncheckedOutputCombinations(t *testing.T) {
	writers := []struct {
		name, expr string
		fallible   bool
	}{
		{"parameter", "w", true},
		{"buffer", "new(bytes.Buffer)", false},
		{"builder", "new(strings.Builder)", false},
		{"alias", "new(Memory)", false},
		{"interface", "io.Writer(new(bytes.Buffer))", true},
		{"custom buffer", "new(FailingBuffer)", true},
	}
	uses := []struct {
		name, before, after string
		discarded           bool
	}{
		{"standalone", "", ";return 0,nil", true},
		{"blank assignment", "_,_=", ";return 0,nil", true},
		{"parenthesized blank assignment", "_,(_)=", ";return 0,nil", true},
		{"nested parenthesized blank assignment", "_,(((_)))=", ";return 0,nil", true},

		{"count assignment", "n,_:=", ";return n,nil", true},
		{"blank declaration", "var n,_=", ";return n,nil", true},
		{"checked", "n,err:=", ";if err!=nil{return n,err};return n,nil", false},
		{"propagated", "return ", "", false},
	}
	for _, callee := range []string{"Fprint", "Fprintf", "Fprintln"} {
		for _, writer := range writers {
			for _, use := range uses {
				t.Run(callee+"/"+writer.name+"/"+use.name, func(t *testing.T) {
					call := `fmt.` + callee + `(` + writer.expr + `,"%s","text")`
					source := `package p;import("fmt";"io";"bytes";"strings")
type Memory = strings.Builder
type FailingBuffer struct{bytes.Buffer}
func(*FailingBuffer)Write([]byte)(int,error){return 0,io.ErrClosedPipe}
func f(w io.Writer)(int,error){` + use.before + call + use.after + `}`
					file, fs, info, _ := typeCheck(t, source)
					sites := findErrorCheckSites(findFunc(file, "f"), fs, info)
					if len(sites) != 1 {
						t.Fatalf("expected one typed output call: %+v", sites)
					}
					want := writer.fallible && use.discarded
					if sites[0].DiscardedOutput != want {
						t.Fatalf("discarded=%v,want %v", sites[0].DiscardedOutput, want)
					}
				})
			}
		}
	}
}

func TestUncheckedOutputRenamedImport(t *testing.T) {
	source := `package p; import("io"; print "fmt")
func f(w io.Writer)error{print.Fprintln(w,"text");return nil}`
	file, fs, info, _ := typeCheck(t, source)
	sites := findErrorCheckSites(findFunc(file, "f"), fs, info)
	if len(sites) != 1 || !sites[0].DiscardedOutput || sites[0].Callee != "print.Fprintln" {
		t.Fatalf("renamed stdlib import loses identity: %+v", sites)
	}
}

func TestUncheckedOutputUsesPhysicalSourceLine(t *testing.T) {
	source := "package p\nimport(\"fmt\";\"io\")\nfunc f(w io.Writer)error{\n//line virtual.go:900\nfmt.Fprintln(w,\"text\")\nreturn nil\n}\n"
	file, fs, info, _ := typeCheck(t, source)
	sites := findErrorCheckSites(findFunc(file, "f"), fs, info)
	if len(sites) != 1 || !sites[0].DiscardedOutput || sites[0].Line != 5 {
		t.Fatalf("finding does not point to physical source line 5: %+v", sites)
	}
}
