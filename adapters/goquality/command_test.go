package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestCommandOwnsOnlyOuterRecord(t *testing.T) {
	testCommandOwnsOnlyOuterRecord(t, false)
}

func TestCommandContextOwnsOnlyOuterRecord(t *testing.T) {
	testCommandOwnsOnlyOuterRecord(t, true)
}

func testCommandOwnsOnlyOuterRecord(t *testing.T, withContext bool) {
	t.Helper()
	source := `package repro
import "os/exec"
func GitOutput(dir string, args ...string)(string,error){
 command:=exec.Command("git",args...)
 command.Dir=dir
 output,err:=command.Output()
 return string(output),err
}
func Local(){command:=exec.Command("git");command.Dir="."}
func LocalStderr(){command:=exec.Command("git");command.Stderr=nil}
func Alias(){command:=exec.Command("git");alias:=command;alias.Dir="."}
func Shared(command *exec.Cmd){command.Dir="."}
func Mixed(input *exec.Cmd, flag bool){command:=exec.Command("git");if flag{command=input};command.Dir="."}
func Rebound(input *exec.Cmd){command:=exec.Command("git");command=input;command.Dir="."}
func Reference(){command:=exec.Command("git");command.Args[0]="other"}
func BorrowedField(input []string){command:=exec.Command("git");command.Args=input;command.Args[0]="other"}
func Address(command **exec.Cmd){local:=exec.Command("git");local=*command;local.Dir="."}
var counter int
func name()string{counter++;return "git"}
func ArgumentEffect(){command:=exec.Command(name());command.Dir="."}
func Dynamic(factory func(string,...string)*exec.Cmd){command:=factory("git");command.Dir="."}
var shared *exec.Cmd
type Builder struct{}
func (Builder) Command(string,...string)*exec.Cmd{return shared}
func Shadow(){exec:=Builder{};command:=exec.Command("git");command.Dir="."}
`
	constructor := "os/exec.Command"
	if withContext {
		constructor = "os/exec.CommandContext"
		source = strings.NewReplacer(
			`import "os/exec"`, `import ("os/exec"; "context")`,
			"exec.Command(", "exec.CommandContext(context.Background(),",
			"(Builder) Command(", "(Builder) CommandContext(context.Context,",
		).Replace(source)
	}
	root := qualitySourceModule(t, []byte(source))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"GitOutput", "Local", "LocalStderr", "Alias", "ArgumentEffect"} {
		function := findFunction(t, facts, name)
		if len(function.Mutations) != 1 || function.Mutations[0].Provenance != quality.Local {
			t.Errorf("%s must own the command slot: %+v", name, function.Mutations)
		}
	}
	controls := []string{"Shared", "Mixed", "Rebound", "Reference", "BorrowedField", "Address", "Dynamic", "Shadow"}
	for _, name := range controls {
		function := findFunction(t, facts, name)
		mutation := function.Mutations[len(function.Mutations)-1]
		if mutation.Provenance == quality.Local {
			t.Errorf("%s must retain shared or unknown storage: %+v", name, mutation)
		}
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range controls {
		result := resultNamed(t, report, name)
		if len(result.Effects) == 0 {
			t.Errorf("%s loses its observable or unknown effects: %+v", name, result)
		}
	}
	git := resultNamed(t, report, "GitOutput")
	if len(git.Effects) != 1 || git.Effects[0].Kind != quality.IO || git.DensityMilli != 750 {
		t.Errorf("command setup must retain only process IO: %+v", git)
	}
	argument := resultNamed(t, report, "ArgumentEffect")
	if len(argument.Effects) != 1 || argument.Effects[0].Kind != quality.Global {
		t.Errorf("constructor setup must retain the argument effect: %+v", argument)
	}
	encoded, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	var restored []quality.Function
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	replay, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil || !reflect.DeepEqual(report, replay) {
		t.Fatal("saved command ownership changes evaluation", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: constructor, Kind: quality.IO})
	overridden, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	local := resultNamed(t, overridden, "Local")
	if len(local.Effects) != 1 || local.Effects[0].Kind != quality.IO {
		t.Errorf("command ownership hides policy override: %+v", local)
	}
}
