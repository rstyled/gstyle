package patterncheck

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestFactoryCanReturnLiveFileWithError(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod": "module factoryerror\n\ngo 1.22\n",
		"factory.go": `package factoryerror
import("io/fs";"errors")
func bad(files fs.FS,cause error)(fs.File,error){f,_:=files.Open("file");return f,cause}
type problem struct{}
func (*problem)Error()string{return "typed nil"}
func boxed(files fs.FS)(fs.File,error){f,_:=files.Open("file");var cause *problem;var err error=cause;return f,err}
func brokenBoxed(files fs.FS)error{f,err:=boxed(files);if err!=nil{return err};return f.Close()}
func broken(files fs.FS)error{f,err:=bad(files,errors.New("failure with live file"));if err!=nil{return err};return f.Close()}
`,
		"factory_test.go": `package factoryerror
import("io/fs";"testing")
type trackedFile struct{fs.File;closed bool}
func(f *trackedFile)Close()error{f.closed=true;return nil}
type provider struct{file *trackedFile}
func(p provider)Open(string)(fs.File,error){return p.file,nil}
func TestRuntimeBoxedOwnership(t *testing.T){f:=&trackedFile{};if brokenBoxed(provider{f})==nil{t.Fatal("boxed nil error missing")};if f.closed{t.Fatal("file unexpectedly closed")};f.Close()}
func TestRuntimeFailureOwnership(t *testing.T){f:=&trackedFile{};if broken(provider{f})==nil{t.Fatal("error missing")};if f.closed{t.Fatal("file unexpectedly closed")};f.Close()}
`,
	}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
	if output, err := exec.Command("go", "test", "-p", "1", "-count=1", "./...").CombinedOutput(); err != nil {
		t.Fatalf("native file/error case: %v\n%s", err, output)
	}
	report, err := GetReport(context.Background(), []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, candidate := range report.Candidates {
		if candidate.Kind != patterns.Obligation {
			continue
		}
		for _, site := range candidate.Sites {
			if strings.HasSuffix(site.Name, ".broken") || strings.HasSuffix(site.Name, ".brokenBoxed") {
				found++
			}
		}
	}
	if found != 2 {
		t.Fatalf("live-file/error leak findings=%d want 2", found)
	}
}
