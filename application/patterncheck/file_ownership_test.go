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

func TestFileOwnershipCallers(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{
		"go.mod": "module ownership\n\ngo 1.22\n",
		"ownership_test.go": `package ownership
import("errors";"io/fs";"os";"testing")
type trackedFile struct{fs.File;closed bool}
func(f *trackedFile)Close()error{f.closed=true;return nil}
type trackedFS struct{file *trackedFile}
func(f trackedFS)Open(string)(fs.File,error){return f.file,nil}
func TestRuntimeOwnership(t *testing.T){
 for _,test:=range []struct{call func(fs.FS,string)error;want bool}{{fsSafe,true},{fsLeak,false},{borrowedLeak,false},{borrowedResultLeak,false}}{
  f:=&trackedFile{}
  if err:=test.call(trackedFS{f},"feature");err!=nil{t.Fatal(err)}
  if f.closed!=test.want{t.Fatalf("closed=%v,want %v",f.closed,test.want)}
  f.Close()
 }
 name:=t.TempDir()+"/file"
 if err:=os.WriteFile(name,[]byte("content"),0600);err!=nil{t.Fatal(err)}
 f,err:=pass(name);if err!=nil{t.Fatal(err)}
 defer f.Close()
 buf:=make([]byte,7)
 if _,err:=f.Read(buf);err!=nil{t.Fatalf("returned file not open: %v",err)}
 if string(buf)!="content"{t.Fatalf("content=%q",buf)}
 if err:=f.Close();err!=nil{t.Fatal(err)}
 if _,err:=f.Read(buf);!errors.Is(err,os.ErrClosed){t.Fatalf("closed read error=%v",err)}
}
`,
		"ownership.go": `package ownership
import("os";"io/fs";"io")
func open(name string)(*os.File,error){return os.Open(name)}
var choose bool
var other *os.File
func directCreate(name string)(*os.File,error){return os.Create(name)}
func localOpen(name string)(*os.File,error){f,err:=os.Open(name);if err!=nil{return nil,err};return f,nil}
func returnOrClose(name string)(*os.File,error){f,err:=os.Open(name);if err!=nil{return nil,err};if choose{return f,nil};return nil,f.Close()}
func discarded(name string)(*os.File,error){f,err:=os.Open(name);if err!=nil{return nil,err};_=f;return nil,nil}
func branchDrop(name string)(*os.File,error){f,err:=os.Open(name);if err!=nil{return nil,err};if choose{return f,nil};return nil,nil}
func differentFile(name string)(*os.File,error){f,err:=os.Open(name);if err!=nil{return nil,err};_=f;return other,nil}
func reassigned(name string)(*os.File,error){f,err:=os.Open(name);if err!=nil{return nil,err};f=other;return f,nil}
func sameLine(name string)(*os.File,error){f,err:=os.Open(name);_=f;_=err;return os.Open(name)}
func forward(name string)(*os.File,error){return open(name)}
func pass(name string)(*os.File,error){f,err:=forward(name);if err!=nil{return nil,err};return f,nil}
func safe(name string)error{f,err:=pass(name);if err!=nil{return err};defer f.Close();return nil}
func inverseSafe(name string)error{f,err:=pass(name);if err==nil{defer f.Close()};return err}
func boxedUnknown(name string)(any,error){f,err:=open(name);if err!=nil{return nil,err};return struct{File *os.File}{f},nil}
func varAliasUnknown(name string)error{f,err:=open(name);if err!=nil{return err};var alias=f;_=alias;return nil}
func leak(name string)error{f,err:=pass(name);if err!=nil{return err};_=f;return nil}
func partial(name string,b bool)error{f,err:=forward(name);if err!=nil{return err};if b{return f.Close()};return nil}
func helperSafe(name string)error{f,err:=forward(name);if err!=nil{return err};return closeFile(f)}
func closeFile(f *os.File)error{return f.Close()}
func fsSafe(files fs.FS,name string)error{f,err:=files.Open(name);if err!=nil{return err};defer f.Close();return nil}
func borrowedLeak(files fs.FS,name string)error{f,err:=files.Open(name);if err!=nil{return err};consumeReader(io.TeeReader(f,io.Discard));return nil}
func consumeReader(io.Reader){}
func borrowedResultLeak(files fs.FS,name string)error{f,err:=files.Open(name);if err!=nil{return err};_,err=consumeReaderResult(io.TeeReader(f,io.Discard));return err}
func consumeReaderResult(io.Reader)(int,error){return 0,nil}
func fsLeak(files fs.FS,name string)error{f,err:=files.Open(name);if err!=nil{return err};_=f;return nil}
type unknownFS interface{Open(string)(fs.File,error)}
func unknown(files unknownFS,name string)error{f,err:=files.Open(name);if err!=nil{return err};_=f;return nil}
func fake(name string)(*os.File,error){return nil,nil}
func fakeCaller(name string)error{f,err:=fake(name);_=f;return err}
func consumed(name string)error{f,err:=open(name);if err!=nil{return err};consume(f);return nil}
func consume(f *os.File){}
func aliasUnknown(name string)error{f,err:=open(name);if err!=nil{return err};alias:=f;_=alias;return nil}
func closedSource(name string)(*os.File,error){f,err:=open(name);if err!=nil{return nil,err};f.Close();return f,nil}
func closedCaller(name string)error{f,err:=closedSource(name);_=f;return err}
func cycle(name string)(*os.File,error){return cycle(name)}
func cycleCaller(name string)error{f,err:=cycle(name);_=f;return err}
`,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
	if output, err := exec.Command("go", "test", "-p", "1", "-count=1", "./...").CombinedOutput(); err != nil {
		t.Fatalf("runtime ownership check: %v\n%s", err, output)
	}
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
	for name, want := range map[string]int{
		"aliasUnknown": 1, "varAliasUnknown": 1,
		"leak": 1, "partial": 1, "fsLeak": 1, "borrowedLeak": 1, "borrowedResultLeak": 1,
		"discarded": 1, "branchDrop": 1, "differentFile": 1, "reassigned": 1, "sameLine": 2,
	} {
		if counts[name] != want {
			t.Errorf("%s findings=%d,want %d", name, counts[name], want)
		}
		delete(counts, name)
	}
	if len(counts) != 0 {
		t.Fatalf("unexpected findings: %v", counts)
	}
}
