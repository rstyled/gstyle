package patterncheck

import (
	"context"
	"github.com/rstyled/gstyle/domain/patterns"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestResourceReleaseUsesSelectedCallbackStates(t *testing.T) {
	root := t.TempDir()
	source := `package example
import("io/fs";"os")
func stop(){panic(7)}
func abort(){os.Exit(1)}
func Safe(files fs.FS,choose bool){f,e:=files.Open("x");if e!=nil{return};defer f.Close();cb:=stop;if choose{cb=abort};if !choose{cb()}}
func Leak(files fs.FS,choose bool){f,e:=files.Open("x");if e!=nil{return};_=f;cb:=stop;if choose{cb=abort};if !choose{cb()}}
func Aborted(files fs.FS,choose bool){f,e:=files.Open("x");if e!=nil{return};defer f.Close();cb:=stop;if choose{cb=abort};if choose{cb()}}
func Mixed(files fs.FS,choose bool){f,e:=files.Open("x");if e!=nil{return};defer f.Close();cb:=stop;if choose{cb=abort};cb()}
func Unknown(files fs.FS,cb func()){f,e:=files.Open("x");if e!=nil{return};defer f.Close();cb()}
`
	native := `package example
import("io";"io/fs";"testing")
type file struct{closed bool}
func(f *file)Stat()(fs.FileInfo,error){return nil,nil}
func(f *file)Read([]byte)(int,error){return 0,io.EOF}
func(f *file)Close()error{f.closed=true;return nil}
type files struct{file *file}
func(f files)Open(string)(fs.File,error){return f.file,nil}
func TestNative(t *testing.T){for _,choose:=range []bool{false,true}{for _,safe:=range []bool{false,true}{f:=&file{};func(){defer func(){recover()}();if safe{Safe(files{f},choose)}else{Leak(files{f},choose)}}();if f.closed!=safe{t.Fatalf("choose=%v safe=%v closed=%v",choose,safe,f.closed)}}}}
`
	for name, body := range map[string]string{"go.mod": "module example\n\ngo 1.26.0\n", "case.go": source, "case_test.go": native} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "test", "-p", "1", "-count=1", ".")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native resource check: %v\n%s", err, out)
	}
	t.Chdir(root)
	report, err := GetReport(context.Background(), []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	hits := map[string]int{}
	for _, candidate := range report.Candidates {
		if candidate.Kind == patterns.Obligation {
			for _, site := range candidate.Sites {
				hits[site.Name]++
			}
		}
	}
	for _, name := range []string{"Leak", "Aborted", "Mixed", "Unknown"} {
		if hits["example."+name] != 1 {
			t.Errorf("%s findings=%d want 1", name, hits["example."+name])
		}
		delete(hits, "example."+name)
	}
	if len(hits) != 0 {
		t.Fatalf("unexpected resource warnings: %v", hits)
	}
}
