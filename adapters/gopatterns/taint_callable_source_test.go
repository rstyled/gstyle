package gopatterns

import "testing"

func TestTaintCallableContexts(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"callback returns text", `func id(q string)string{return q};func apply(cb func(string)string,q string)string{return cb(q)};func handler(r *http.Request){db.Exec(apply(id,r.FormValue("q")))}`, 1},
		{"callback ignores text", `func clean(q string)string{return "SELECT 1"};func apply(cb func(string)string,q string)string{return cb(q)};func handler(r *http.Request){db.Exec(apply(clean,r.FormValue("q")))}`, 0},
		{"returned declaration", `func id(q string)string{return q};func get()func(string)string{return id};func handler(r *http.Request){db.Exec(get()(r.FormValue("q")))}`, 1},
		{"returned captured text", `func capture(q string)func()string{return func()string{return q}};func handler(r *http.Request){db.Exec(capture(r.FormValue("q"))())}`, 1},
		{"returned clean text", `func capture(q string)func()string{return func()string{return q}};func handler(r *http.Request){db.Exec(capture("SELECT 1")())}`, 0},
		{"live write after construction", `func handler(r *http.Request){q:="SELECT 1";read:=func()string{return q};q=r.FormValue("q");db.Exec(read())}`, 1},
		{"live clear after construction", `func handler(r *http.Request){q:=r.FormValue("q");read:=func()string{return q};q="SELECT 1";db.Exec(read())}`, 0},
		{"shared returned cell write", `func pair(q string)(func()string,func(string)){return func()string{return q},func(v string){q=v}};func handler(r *http.Request){read,set:=pair("SELECT 1");set(r.FormValue("q"));db.Exec(read())}`, 1},
		{"shared returned cell clear", `func pair(q string)(func()string,func(string)){return func()string{return q},func(v string){q=v}};func handler(r *http.Request){read,set:=pair(r.FormValue("q"));set("SELECT 1");db.Exec(read())}`, 0},
		{"separate factory cells", `func capture(q string)func()string{return func()string{return q}};func handler(r *http.Request){bad:=capture(r.FormValue("q"));good:=capture("SELECT 1");_=bad;db.Exec(good())}`, 0},
		{"nested relay", `func capture(q string)func()func()string{return func()func()string{return func()string{return q}}};func handler(r *http.Request){db.Exec(capture(r.FormValue("q"))()())}`, 1},
		{"capture through callback", `func apply(cb func()string)string{return cb()};func handler(r *http.Request){q:=r.FormValue("q");db.Exec(apply(func()string{return q}))}`, 1},
		{"clean capture through callback", `func apply(cb func()string)string{return cb()};func handler(r *http.Request){q:="SELECT 1";db.Exec(apply(func()string{return q}))}`, 0},
		{"captured clean callable", `func clean(q string)string{return "SELECT 1"};func handler(r *http.Request){cb:=clean;run:=func(q string)string{return cb(q)};db.Exec(run(r.FormValue("q")))}`, 0},
		{"captured callable changed", `func clean(q string)string{return "SELECT 1"};func id(q string)string{return q};func handler(r *http.Request){cb:=clean;run:=func(q string)string{return cb(q)};cb=id;db.Exec(run(r.FormValue("q")))}`, 1},
		{"captured callable cleared", `func clean(q string)string{return "SELECT 1"};func id(q string)string{return q};func handler(r *http.Request){cb:=id;run:=func(q string)string{return cb(q)};cb=clean;db.Exec(run(r.FormValue("q")))}`, 0},
		{"unknown callback", `func handler(r *http.Request,cb func(string)string){db.Exec(cb(r.FormValue("q")))}`, 1},
		{"recursive clean callback", `func clean(q string)string{return "SELECT 1"};func apply(cb func(string)string,q string,n int)string{if n==0{return cb(q)};return apply(cb,q,n-1)};func handler(r *http.Request){db.Exec(apply(clean,r.FormValue("q"),3))}`, 0},
		{"recursive unsafe callback", `func id(q string)string{return q};func apply(cb func(string)string,q string,n int)string{if n==0{return cb(q)};return apply(cb,q,n-1)};func handler(r *http.Request){db.Exec(apply(id,r.FormValue("q"),3))}`, 1},
		{"unnamed input", `func clean(string)string{return "SELECT 1"};func handler(r *http.Request){db.Exec(clean(r.FormValue("q")))}`, 0},
		{"escaped captured cell", `func handler(r *http.Request,write func(*string,string)){q:="SELECT 1";read:=func()string{return q};write(&q,r.FormValue("q"));db.Exec(read())}`, 1},
		{"captured pointer helper write", `func write(p *string,q string){*p=q};func handler(r *http.Request){q:="SELECT 1";read:=func()string{return q};write(&q,r.FormValue("q"));db.Exec(read())}`, 1},
		{"nil callback does not reach sink", `func handler(r *http.Request){var cb func(string)string;db.Exec(cb(r.FormValue("q")))}`, 0},
		{"parenthesized clean callback", `func clean(q string)string{return "SELECT 1"};func apply(cb func(string)string,q string)string{return cb(q)};func handler(r *http.Request){db.Exec(apply((clean),r.FormValue("q")))}`, 0},
		{"named clean callback", `type Callback func(string)string;func clean(q string)string{return "SELECT 1"};func apply(cb Callback,q string)string{return cb(q)};func handler(r *http.Request){db.Exec(apply(Callback(clean),r.FormValue("q")))}`, 0},
		{"two factories through one helper", `func capture(q string)func()string{return func()string{return q}};func twins(a,b string)(func()string,func()string){return capture(a),capture(b)};func handler(r *http.Request){_,good:=twins(r.FormValue("q"),"SELECT 1");db.Exec(good())}`, 0},
		{"nil joined with unknown callable", `func handler(r *http.Request,choose bool,dynamic any){var cb func(string)string;if choose{cb=dynamic.(func(string)string)};db.Exec(cb(r.FormValue("q")))}`, 1},
		{"unknown closure consumer", `func handler(r *http.Request,opaque func(func(string),string)){q:="SELECT 1";read:=func()string{return q};set:=func(v string){q=v};opaque(set,r.FormValue("q"));db.Exec(read())}`, 1},
		{"forwarded unknown closure consumer", `var opaque func(func(string),string);func forward(cb func(string),q string){opaque(cb,q)};func handler(r *http.Request){q:="SELECT 1";read:=func()string{return q};set:=func(v string){q=v};forward(set,r.FormValue("q"));db.Exec(read())}`, 1},
		{"unknown call returns captured text", `func handler(r *http.Request,opaque func(func()string)string){q:=r.FormValue("q");db.Exec(opaque(func()string{return q}))}`, 1},
		{"interface closure consumer", `func handler(r *http.Request,opaque func(any,string)){q:="SELECT 1";read:=func()string{return q};set:=func(v string){q=v};opaque(set,r.FormValue("q"));db.Exec(read())}`, 1},
		{"forwarded interface closure consumer", `var opaque func(any,string);func forward(cb any,q string){opaque(cb,q)};func handler(r *http.Request){q:="SELECT 1";read:=func()string{return q};set:=func(v string){q=v};forward(set,r.FormValue("q"));db.Exec(read())}`, 1},
		{"forwarded unknown getter", `var opaque func(any)string;func forward(cb any)string{return opaque(cb)};func handler(r *http.Request){q:=r.FormValue("q");db.Exec(forward(func()string{return q}))}`, 1},
		{"source and sink in closure", `func handler(r *http.Request){func(){db.Exec(r.FormValue("q"))}()}`, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flows := sourceTaintFlows(t, tc.body)
			if len(flows) != tc.want {
				t.Fatalf("got %+v, want %d", flows, tc.want)
			}
		})
	}
}

func TestUnknownCallbackDiscardsCapturedEncoding(t *testing.T) {
	flows := moduleTaintFlows(t, map[string]string{"go.mod": "module example\n\ngo 1.26.0\n", "p.go": `package example
import("database/sql";"net/http";"fmt")
var db *sql.DB
func handler(r *http.Request,opaque func(func()string)string){q:=fmt.Sprintf("%d",len(r.FormValue("q")));db.Exec(opaque(func()string{return q}))}
`})
	if len(flows) != 1 {
		t.Fatalf("got %+v want one conservative flow", flows)
	}
}
