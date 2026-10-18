package gopatterns

import (
	"strings"
	"testing"
)

func TestTaintDecimalEncoding(t *testing.T) {
	cases := []struct {
		name, body string
		want       int
	}{
		{"decimal length", `func handler(r *http.Request){db.Exec(fmt.Sprintf("SELECT $%d",len(r.FormValue("q"))))}`, 0},
		{"character length", `func handler(r *http.Request){db.Exec(fmt.Sprintf("SELECT %c",len(r.FormValue("q"))))}`, 1},
		{"raw text", `func handler(r *http.Request){db.Exec(fmt.Sprintf("SELECT %s",r.FormValue("q")))}`, 1},
		{"mixed decimal and text", `func handler(r *http.Request){q:=r.FormValue("q");db.Exec(fmt.Sprintf("SELECT $%d %s",len(q),q))}`, 1},
		{"indexed format stays unknown", `func handler(r *http.Request){db.Exec(fmt.Sprintf("SELECT %[1]d",len(r.FormValue("q"))))}`, 1},
		{"dynamic width stays unknown", `func handler(r *http.Request){db.Exec(fmt.Sprintf("SELECT %*d",1,len(r.FormValue("q"))))}`, 1},
		{"custom formatter", `type Number int;func(n Number)Format(s fmt.State,v rune){fmt.Fprint(s,"unsafe")};func handler(r *http.Request){db.Exec(fmt.Sprintf("SELECT %d",Number(len(r.FormValue("q")))))}`, 1},
		{"primitive alias", `type Number = int;func handler(r *http.Request){db.Exec(fmt.Sprintf("SELECT %d",Number(len(r.FormValue("q")))))}`, 0},
		{"interface stays unknown", `func handler(r *http.Request){var n any=len(r.FormValue("q"));db.Exec(fmt.Sprintf("SELECT %d",n))}`, 1},
		{"decode restores dependency", `func handler(r *http.Request){n,_:=strconv.Atoi(fmt.Sprintf("%d",len(r.FormValue("q"))));db.Exec(fmt.Sprintf("SELECT %c",n))}`, 1},
		{"arithmetic decodes character", `func handler(r *http.Request){s:=fmt.Sprintf("%d",len(r.FormValue("q")));db.Exec(fmt.Sprintf("SELECT %c",s[0]+39))}`, 1},
		{"decimal SSRF stays tainted", `func handler(r *http.Request){http.Get(fmt.Sprintf("http://%d",len(r.FormValue("q"))))}`, 1},
		{"encoded helper", `func decimal(n int)string{return fmt.Sprintf("%d",n)};func handler(r *http.Request){db.Exec("SELECT "+decimal(len(r.FormValue("q"))))}`, 0},
		{"raw and encoded join", `func handler(r *http.Request,yes bool){q:=fmt.Sprintf("%d",len(r.FormValue("q")));if yes{q=r.FormValue("q")};db.Exec(q)}`, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flows := moduleTaintFlows(t, map[string]string{"go.mod": "module example\n\ngo 1.26.0\n", "p.go": `package example
import("database/sql";"net/http";"fmt";"strconv")
var db *sql.DB
var _ = strconv.Itoa
` + tc.body})
			if len(flows) != tc.want {
				t.Fatalf("got %+v, want %d", flows, tc.want)
			}
		})
	}
}

// Match the historical builder shape: placeholder numbers depend on args,
// while only the old ranking fragment interpolates the search text itself.
func TestTaintSearchRankingFix(t *testing.T) {
	source := `package example
import("database/sql";"net/http";"fmt";"strings")
var db *sql.DB
type Builder struct{args []any;conditions []string;order string}
func(b *Builder)Search(q string){
 if q!=""{n:=len(b.args)+1;b.conditions=append(b.conditions,fmt.Sprintf("document @@ query($%d)",n));b.args=append(b.args,q);RANK}
}
func(b *Builder)Count(){db.QueryRow("SELECT count(*) WHERE "+strings.Join(b.conditions," AND "),b.args...)}
func(b *Builder)Entries(){db.Query(fmt.Sprintf("SELECT * WHERE %s ORDER BY %s",strings.Join(b.conditions," AND "),b.order),b.args...)}
func handler(r *http.Request){b:=&Builder{};b.Search(r.FormValue("q"));b.Search(r.FormValue("q"));b.Count();b.Entries()}
`
	for _, tc := range []struct {
		name, rank string
		want       int
	}{
		{"vulnerable", `b.order=fmt.Sprintf("rank(query('%s'))",q)`, 2},
		{"fixed", `b.order=fmt.Sprintf("rank(query($%d))",n)`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flows := moduleTaintFlows(t, map[string]string{"go.mod": "module example\n\ngo 1.26.0\n", "p.go": strings.Replace(source, "RANK", tc.rank, 1)})
			if len(flows) != tc.want {
				t.Fatalf("got %+v, want %d", flows, tc.want)
			}
		})
	}
}
