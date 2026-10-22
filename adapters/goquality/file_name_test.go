package goquality

import (
	"context"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestFileNameHasNoIO(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "os"
type Alias = os.File
func Name(file *os.File) string { return file.Name() }
func AliasName(file *Alias) string { return file.Name() }
func Expression(file *os.File) string { return (*os.File).Name(file) }
func Forward(file *os.File) string { return Name(file) }
func Stat(file *os.File) (os.FileInfo, error) { return file.Stat() }
func Close(file *os.File) error { return file.Close() }
func Write(file *os.File, data []byte) (int, error) { return file.Write(data) }
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Name", "AliasName", "Expression", "Forward"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 0 || result.DensityMilli != 0 || !result.Complete {
			t.Errorf("%s reads stored metadata, want no effects: %+v", name, result)
		}
	}
	for _, name := range []string{"Stat", "Close", "Write"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 1 || result.Effects[0].Kind != quality.IO {
			t.Errorf("%s must retain IO: %+v", name, result)
		}
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "os.File.Name", Kind: quality.IO})
	overridden, err := quality.Evaluate(facts, config)
	if err != nil {
		t.Fatal(err)
	}
	result := resultNamed(t, overridden, "Name")
	if len(result.Effects) != 1 || result.Effects[0].Kind != quality.IO {
		t.Fatalf("explicit policy must retain IO: %+v", result)
	}
}
