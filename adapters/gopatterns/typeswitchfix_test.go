package gopatterns

import (
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestApplyTypeSwitchFix(t *testing.T) {
	src := `package main

import "fmt"

type Dog struct{ name string }
type Cat struct{ name string }

func (d Dog) Speak() { fmt.Println("woof") }
func (c Cat) Speak() { fmt.Println("meow") }

func speakAll(x interface{}) {
	switch v := x.(type) {
	case Dog:
		v.Speak()
	case Cat:
		v.Speak()
	}
}
`
	spec := &patterns.FixSpec{
		Kind: patterns.TypeSwitch,
		File: "main.go",
		Line: 12,
		Params: map[string]string{
			"bound":  "v",
			"expr":   "x",
			"method": "Speak",
			"types":  "Dog,Cat",
			"args":   "",
		},
	}
	out, err := applyProjectedFix(t, spec, []byte(src))
	if err != nil {
		t.Fatalf("fix failed: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, `v.(interface{ Speak() }).Speak()`) {
		t.Errorf("missing interface assertion:\n%s", got)
	}
	if !strings.Contains(got, "case Dog, Cat:") {
		t.Errorf("case set not preserved:\n%s", got)
	}
}

func TestApplyTypeSwitchFixSkipsMissingMethod(t *testing.T) {
	src := `package main

func speakAll(x interface{}) {
	switch v := x.(type) {
	case Dog:
		v.Speak()
	case Cat:
		v.Speak()
	}
}
`
	spec := &patterns.FixSpec{
		Kind: patterns.TypeSwitch,
		File: "main.go",
		Line: 4,
		Params: map[string]string{
			"bound":  "v",
			"expr":   "x",
			"method": "Speak",
			"types":  "Dog,Cat",
			"args":   "",
		},
	}
	if _, err := applyProjectedFix(t, spec, []byte(src)); err == nil {
		t.Error("expected error when method decls missing")
	}
}

func TestApplyTypeSwitchFixSkipsMismatchedSigs(t *testing.T) {
	src := `package main

type Dog struct{}
type Cat struct{}

func (d Dog) Speak() {}
func (c Cat) Speak(s string) {}

func speakAll(x interface{}) {
	switch v := x.(type) {
	case Dog:
		v.Speak()
	case Cat:
		v.Speak()
	}
}
`
	spec := &patterns.FixSpec{
		Kind: patterns.TypeSwitch,
		File: "main.go",
		Line: 10,
		Params: map[string]string{
			"bound":  "v",
			"expr":   "x",
			"method": "Speak",
			"types":  "Dog,Cat",
			"args":   "",
		},
	}
	if _, err := applyProjectedFix(t, spec, []byte(src)); err == nil {
		t.Error("expected error when signatures differ")
	}
}

func TestApplyTypeSwitchFixWithSignature(t *testing.T) {
	src := `package main

type Dog struct{}
type Cat struct{}

func (d Dog) Speak(volume int) string { return "woof" }
func (c Cat) Speak(volume int) string { return "meow" }

func speakAll(x interface{}) string {
	switch v := x.(type) {
	case Dog:
		v.Speak(3)
	case Cat:
		v.Speak(3)
	}
	return ""
}
`
	// Note: arms are ExprStmt so this wouldn't be detected (return value
	// discarded), but the fixer should still build the right signature
	// when method decls exist. We test sig extraction directly.
	spec := &patterns.FixSpec{
		Kind: patterns.TypeSwitch,
		File: "main.go",
		Line: 10,
		Params: map[string]string{
			"bound":  "v",
			"expr":   "x",
			"method": "Speak",
			"types":  "Dog,Cat",
			"args":   "3",
		},
	}
	out, err := applyProjectedFix(t, spec, []byte(src))
	if err != nil {
		t.Fatalf("fix failed: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "interface{ Speak(volume int) string }") {
		t.Errorf("wrong signature:\n%s", got)
	}
}
