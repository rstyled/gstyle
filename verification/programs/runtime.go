package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

func representativeInputs(selected []suite) []int64 {
	values := []int64{math.MinInt64, math.MaxInt64}
	for _, s := range selected {
		for _, c := range s.Tests {
			values = append(values, boundaryInputs(c.Bound)...)
		}
	}
	slices.Sort(values)
	return slices.Compact(values)
}

func boundaryInputs(bound int64) []int64 {
	values := []int64{bound}
	if bound > math.MinInt64 {
		values = append(values, bound-1)
	}
	if bound < math.MaxInt64 {
		values = append(values, bound+1)
	}
	return values
}

func runtimeSource(cases []observation, inputs []int64) string {
	names := make([]string, 0, len(cases))
	for _, c := range cases {
		names = append(names, "generated."+c.Name)
	}
	return "package main\nimport (\"encoding/json\";\"os\";generated \"example.com/generated\")\nfunc main(){\n" +
		"calls:=[]func(int64)int64{" + strings.Join(names, ",") + "}\n" +
		fmt.Sprintf("inputs:=[]int64%v\n", inputsLiteral(inputs)) +
		"results:=make([][]int64,len(calls));for i,call:=range calls {for _,x:=range inputs{results[i]=append(results[i],call(x))}}\n" +
		"if err:=json.NewEncoder(os.Stdout).Encode(results);err!=nil{panic(err)}\n}\n"
}

func inputsLiteral(inputs []int64) string {
	values := make([]string, 0, len(inputs))
	for _, x := range inputs {
		values = append(values, fmt.Sprintf("%d", x))
	}
	return "{" + strings.Join(values, ",") + "}"
}

func checkRuntime(ctx context.Context, root string, cases []observation, inputs []int64) error {
	directory := filepath.Join(root, "oracle")
	if err := os.Mkdir(directory, 0700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(directory, "main.go"), []byte(runtimeSource(cases, inputs)), 0600); err != nil {
		return err
	}
	command := exec.CommandContext(ctx, "go", "run", "./oracle")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("execute generated Go programs: %w\n%s", err, output)
	}
	var results [][]int64
	if err := json.Unmarshal(output, &results); err != nil {
		return err
	}
	return compareRuntime(cases, inputs, results)
}

func compareRuntime(cases []observation, inputs []int64, results [][]int64) error {
	if len(results) != len(cases) {
		return fmt.Errorf("runtime returned %d programs, expected %d", len(results), len(cases))
	}
	for i, c := range cases {
		if err := compareResults(c, inputs, results[i]); err != nil {
			return err
		}
	}
	return nil
}

func compareResults(c observation, inputs, results []int64) error {
	if len(results) != len(inputs) {
		return fmt.Errorf("%s: missing runtime inputs", c.Name)
	}
	for i, x := range inputs {
		if results[i] != c.Program.eval(x) {
			return fmt.Errorf("%s input %d: Go result %d disagrees with source semantics", c.Name, x, results[i])
		}
	}
	return nil
}
