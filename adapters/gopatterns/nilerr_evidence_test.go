package gopatterns

import "testing"

// Controls are reduced from Prometheus 2213c3c; the positive is Moby #53934.
func TestNilErrHandlingEvidence(t *testing.T) {
	cases := []struct {
		name, source string
		want         int
	}{
		{"Moby empty privileges", `func f(err error)([]string,error){if err!=nil{return []string{},nil};return nil,nil}`, 1},
		{"Moby propagated", `func f(err error)([]string,error){if err!=nil{return []string{},err};return nil,nil}`, 0},
		{"hostname fallback", `func f(err error,hostname string)(string,error){if err!=nil{return hostname,nil};return "",nil}`, 0},
		{"nonempty constant fallback", `func f(err error)(string,error){if err!=nil{return "localhost",nil};return "",nil}`, 0},
		{"field error checked then discarded", `func f(p *result)error{if p.Err!=nil{return nil};return nil}`, 1},
		{"field error validated", `func f(p *result)error{if p.Err!=nil{if validate(p.Err)!=nil{return p.Err};return nil};return nil}`, 0},
		{"EOF classified", `func f(p *result)(int,error){if p.Err!=nil{if classify(p.Err){return 1,nil};return 0,p.Err};return 0,nil}`, 0},
		{"expected error", `func f(p *result,expected bool)error{if p.Err!=nil{if expected{if err:=validate(p.Err);err!=nil{return err};return nil};return p.Err};return nil}`, 0},
		{"field logging not handling", `func f(p *result)error{if p.Err!=nil{println(p.Err);return nil};return nil}`, 1},
		{"other field not handling", `func f(p *result)error{if p.Err!=nil{validate(p.Other);return nil};return nil}`, 1},
		{"other receiver not handling", `func f(p,q *result)error{if p.Err!=nil{validate(q.Err);return nil};return nil}`, 1},
		{"reassigned error unknown", `func f(err error)error{if err!=nil{err=nil;return nil};return nil}`, 0},
		{"field receiver replaced", `func f(p *result)error{if p.Err!=nil{p=&result{};return nil};return nil}`, 0},
		{"alias unknown", `func f(err error)error{if err!=nil{alias:=err;validate(alias);return nil};return nil}`, 0},
		{"declaration alias unknown", `func f(err error)error{if err!=nil{var alias=err;validate(alias);return nil};return nil}`, 0},
		{"alternative recovery unknown", `func f(err error)error{if err!=nil{println(err);if other:=repair();other!=nil{println(other)};return nil};return nil}`, 0},
		{"dereference identity unknown", `func f(p *result)error{if (*p).Err!=nil{validate((*p).Err);return nil};return nil}`, 0},
		{"index identity unknown", `func f(values []error)error{if values[0]!=nil{return nil};return nil}`, 0},
		{"sibling handling not proof", `func f(err error,choose bool)error{if err!=nil{if choose{validate(err);return nil};return nil};return nil}`, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "package p;" + tc.source + `;type result struct{Err,Other error};func validate(error)error{return nil};func classify(error)bool{return true};func repair()error{return nil}`
			fn, fs, info := parseNilErrFunc(t, src)
			if got := findNilErrHits(fn, fs, info); len(got) != tc.want {
				t.Fatalf("findings=%d,want %d: %+v", len(got), tc.want, got)
			}
		})
	}
}

func TestNilErrLoggingIsNotRecovery(t *testing.T) {
	for _, tc := range []struct {
		call string
		want int
	}{
		{`log.Println(err)`, 1}, {`slog.Error("failed","error",err)`, 1}, {`fmt.Fprintln(os.Stderr,err)`, 1}, {`log.Fatal(err)`, 0},
	} {
		t.Run(tc.call, func(t *testing.T) {
			file, fs, info, _ := typeCheck(t, `package p;import("log";"log/slog";"fmt";"os");var _,_,_,_=log.Println,slog.Error,fmt.Fprintln,os.Stderr;func f(err error)error{if err!=nil{`+tc.call+`;return nil};return nil}`)
			if got := findNilErrHits(findFunc(file, "f"), fs, info); len(got) != tc.want {
				t.Fatalf("findings=%d,want %d", len(got), tc.want)
			}
		})
	}
}
