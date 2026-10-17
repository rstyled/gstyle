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

func TestNativeExceptionalResourceOwnership(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod": "module exceptionownership\n\ngo 1.24\n",
		"ownership.go": `package exceptionownership
import("io/fs";"runtime";"os")
func panicClosed(files fs.FS){f,e:=files.Open("file");if e!=nil{return};defer f.Close();panic(7)}
func nilPanicClosed(files fs.FS){f,e:=files.Open("file");if e!=nil{return};defer f.Close();panic(nil)}
func deferredPanicClosed(files fs.FS){f,e:=files.Open("file");if e!=nil{return};defer f.Close();defer panic(7)}
func goexitClosed(files fs.FS){f,e:=files.Open("file");if e!=nil{return};defer f.Close();runtime.Goexit()}
func deferredGoexitClosed(files fs.FS){f,e:=files.Open("file");if e!=nil{return};defer f.Close();defer runtime.Goexit()}
func panicLeak(files fs.FS){f,e:=files.Open("file");if e!=nil{return};_=f;panic(7)}
func goexitLeak(files fs.FS){f,e:=files.Open("file");if e!=nil{return};_=f;runtime.Goexit()}
func exitBeforeClose(files fs.FS){f,e:=files.Open("file");if e!=nil{return};defer f.Close();defer os.Exit(3)}
func closeBeforeExit(files fs.FS){f,e:=files.Open("file");if e!=nil{return};defer os.Exit(3);defer f.Close()}
`,
		"ownership_test.go": `package exceptionownership
import("io/fs";"testing";"os";"os/exec";"fmt";"strings")
type file struct{fs.File;closed bool;signal bool}
func(f *file)Close()error{f.closed=true;if f.signal{fmt.Println("released")};return nil}
type provider struct{file *file}
func(p provider)Open(string)(fs.File,error){return p.file,nil}
func TestNative(t *testing.T){
 for _,tc:=range []struct{call func(fs.FS);closed,panics bool}{
 {panicClosed,true,true},{nilPanicClosed,true,true},{deferredPanicClosed,true,true},
 {goexitClosed,true,false},{deferredGoexitClosed,true,false},{panicLeak,false,true},{goexitLeak,false,false},
 }{
  done:=make(chan bool,1)
  go func(){
   f:=&file{};returned:=false
   defer func(){panics:=recover()!=nil;done<-f.closed==tc.closed&&panics==tc.panics&&!returned}()
   tc.call(provider{f});returned=true
  }()
  if !<-done {t.Fatal("cleanup or exceptional continuation differs")}
 }
}
func TestProcessExitProbe(t *testing.T){
 mode:=os.Getenv("EXCEPTION_EXIT_PROBE")
 if mode==""{return}
 f:=&file{signal:true}
 if mode=="before"{exitBeforeClose(provider{f})}else{closeBeforeExit(provider{f})}
 t.Fatal("process exit returned")
}
func TestProcessExitOrder(t *testing.T){
 executable,err:=os.Executable();if err!=nil{t.Fatal(err)}
 for _,mode:=range []string{"before","after"}{
  cmd:=exec.Command(executable,"-test.run=^TestProcessExitProbe$");cmd.Env=append(os.Environ(),"EXCEPTION_EXIT_PROBE="+mode)
  out,err:=cmd.CombinedOutput();failure,ok:=err.(*exec.ExitError)
  if !ok||failure.ExitCode()!=3{t.Fatalf("process exit: %v %s",err,out)}
  if strings.Contains(string(out),"released")!=(mode=="after"){t.Fatalf("deferred process exit order: %s %s",mode,out)}
 }
}
`,
	}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.Command("go", "test", "-p", "1", "-count=1", "./...")
	command.Dir = root
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("native exceptional ownership: %v\n%s", err, out)
	}
	t.Chdir(root)
	report, err := GetReport(context.Background(), []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, candidate := range report.Candidates {
		if candidate.Kind != patterns.Obligation {
			continue
		}
		for _, site := range candidate.Sites {
			counts[site.Name[strings.LastIndex(site.Name, ".")+1:]]++
		}
	}
	for name, want := range map[string]int{"panicClosed": 0, "nilPanicClosed": 0, "deferredPanicClosed": 0, "goexitClosed": 0, "deferredGoexitClosed": 0, "panicLeak": 1, "goexitLeak": 1, "exitBeforeClose": 1, "closeBeforeExit": 0} {
		if counts[name] != want {
			t.Errorf("%s findings=%d want %d", name, counts[name], want)
		}
		delete(counts, name)
	}
	if len(counts) != 0 {
		t.Fatalf("unexpected exceptional findings: %v", counts)
	}
}
