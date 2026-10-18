// Package sastcases contains static test inputs. Never execute these functions.
package sastcases

import (
	"context"
	"database/sql"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	text "text/template"
)

func SQLDirect(r *http.Request, db *sql.DB)    { _, _ = db.Exec(r.FormValue("q")) }
func SQLContext(r *http.Request, db *sql.DB)   { _, _ = db.ExecContext(r.Context(), r.FormValue("q")) }
func SQLParameter(r *http.Request, db *sql.DB) { _, _ = db.Exec("SELECT ?", r.FormValue("id")) }
func SQLContextParameter(r *http.Request, db *sql.DB) {
	_, _ = db.ExecContext(r.Context(), "SELECT ?", r.FormValue("id"))
}
func SQLTx(r *http.Request, tx *sql.Tx) { _, _ = tx.QueryContext(r.Context(), r.FormValue("q")) }
func SQLConn(r *http.Request, conn *sql.Conn) {
	_ = conn.QueryRowContext(r.Context(), r.FormValue("q"))
}
func SQLPrepare(r *http.Request, db *sql.DB)     { _, _ = db.PrepareContext(r.Context(), r.FormValue("q")) }
func SQLConstant(db *sql.DB)                     { _, _ = db.Exec("SELECT 1") }
func SSRFGet(r *http.Request)                    { _, _ = http.Get(r.FormValue("url")) }
func SSRFClient(r *http.Request, c *http.Client) { _, _ = c.Get(r.FormValue("url")) }
func SSRFPost(r *http.Request)                   { _, _ = http.Post(r.FormValue("url"), "text/plain", nil) }
func SSRFConstant(r *http.Request)               { _, _ = http.Post("https://example.com", r.FormValue("type"), nil) }
func SSRFClientConstant(r *http.Request, c *http.Client) {
	_, _ = c.Post("https://example.com", r.FormValue("type"), nil)
}
func CommandExecutable(r *http.Request) { _ = exec.Command(r.FormValue("program")) }
func CommandContextExecutable(r *http.Request) {
	_ = exec.CommandContext(r.Context(), r.FormValue("program"))
}
func CommandShell(r *http.Request) { _ = exec.Command("/bin/sh", "-c", r.FormValue("code")) }
func CommandContextShell(r *http.Request) {
	_ = exec.CommandContext(r.Context(), "bash", "-c", r.FormValue("code"))
}
func CommandData(r *http.Request) { _ = exec.Command("/usr/bin/printf", "%s", r.FormValue("data")) }
func CommandContextData(r *http.Request) {
	_ = exec.CommandContext(r.Context(), "/usr/bin/printf", "%s", r.FormValue("data"))
}
func HTMLData(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("page").Parse("<p>{{.}}</p>"))
	_ = t.Execute(w, r.FormValue("data"))
}
func HTMLSource(r *http.Request)   { _, _ = template.New("page").Parse(r.FormValue("template")) }
func TextSource(r *http.Request)   { _, _ = text.New("page").Parse(r.FormValue("template")) }
func TextConstant(r *http.Request) { _ = r.FormValue("unused"); _, _ = text.New("page").Parse("hello") }
func PathOpen(r *http.Request)     { _, _ = os.Open(r.FormValue("file")) }
func PathWritePayload(r *http.Request) {
	_ = os.WriteFile("/tmp/fixed-output", []byte(r.FormValue("data")), 0600)
}
func PathOpenFile(r *http.Request)            { _, _ = os.OpenFile(r.FormValue("file"), os.O_RDONLY, 0) }
func DeadSQL(r *http.Request, db *sql.DB)     { return; _, _ = db.Exec(r.FormValue("q")) }
func IndirectSQL(r *http.Request, db *sql.DB) { sqlHelper(db, r.FormValue("q")) }
func sqlHelper(db *sql.DB, q string)          { _, _ = db.Exec(q) }
func RequestObject(r *http.Request, c *http.Client) {
	req, _ := http.NewRequestWithContext(context.Background(), "GET", r.FormValue("url"), nil)
	_, _ = c.Do(req)
}
func ValidatedPath(r *http.Request) {
	name := r.FormValue("file")
	if name != "allowed.txt" {
		return
	}
	_, _ = os.Open(name)
}

// A user type with a standard-library-looking method is not a security sink.
type DB struct{}

func (*DB) Exec(string)                     {}
func LookalikeSink(r *http.Request, db *DB) { db.Exec(r.FormValue("q")) }

type Request struct{}

func (*Request) FormValue(string) string     { return "constant" }
func LookalikeSource(r *Request, db *sql.DB) { _, _ = db.Exec(r.FormValue("q")) }

func SQLDBQuery(r *http.Request, db *sql.DB) { _, _ = db.Query(r.FormValue("q")) }
func SQLDBRow(r *http.Request, db *sql.DB)   { _ = db.QueryRow(r.FormValue("q")) }
func SQLDBQueryContext(r *http.Request, db *sql.DB) {
	_, _ = db.QueryContext(r.Context(), r.FormValue("q"))
}
func SQLDBRowContext(r *http.Request, db *sql.DB) {
	_ = db.QueryRowContext(r.Context(), r.FormValue("q"))
}
func SQLDBPrepare(r *http.Request, db *sql.DB) { _, _ = db.Prepare(r.FormValue("q")) }
func SQLTxExec(r *http.Request, tx *sql.Tx)    { _, _ = tx.Exec(r.FormValue("q")) }
func SQLTxQuery(r *http.Request, tx *sql.Tx)   { _, _ = tx.Query(r.FormValue("q")) }
func SQLTxRow(r *http.Request, tx *sql.Tx)     { _ = tx.QueryRow(r.FormValue("q")) }
func SQLTxExecContext(r *http.Request, tx *sql.Tx) {
	_, _ = tx.ExecContext(r.Context(), r.FormValue("q"))
}
func SQLTxRowContext(r *http.Request, tx *sql.Tx) {
	_ = tx.QueryRowContext(r.Context(), r.FormValue("q"))
}
func SQLTxPrepare(r *http.Request, tx *sql.Tx) { _, _ = tx.Prepare(r.FormValue("q")) }
func SQLTxPrepareContext(r *http.Request, tx *sql.Tx) {
	_, _ = tx.PrepareContext(r.Context(), r.FormValue("q"))
}
func SQLConnExec(r *http.Request, conn *sql.Conn) {
	_, _ = conn.ExecContext(r.Context(), r.FormValue("q"))
}
func SQLConnQuery(r *http.Request, conn *sql.Conn) {
	_, _ = conn.QueryContext(r.Context(), r.FormValue("q"))
}
func SQLConnPrepare(r *http.Request, conn *sql.Conn) {
	_, _ = conn.PrepareContext(r.Context(), r.FormValue("q"))
}
func SQLTxParameter(r *http.Request, tx *sql.Tx) {
	_, _ = tx.ExecContext(r.Context(), "SELECT ?", r.FormValue("id"))
}
func SQLConnParameter(r *http.Request, conn *sql.Conn) {
	_, _ = conn.ExecContext(r.Context(), "SELECT ?", r.FormValue("id"))
}
func SQLStatementParameter(r *http.Request, stmt *sql.Stmt) { _, _ = stmt.Exec(r.FormValue("id")) }

func SSRFPostForm(r *http.Request) { _, _ = http.PostForm(r.FormValue("url"), nil) }
func SSRFClientPost(r *http.Request, c *http.Client) {
	_, _ = c.Post(r.FormValue("url"), "text/plain", nil)
}
func SSRFClientPostForm(r *http.Request, c *http.Client) { _, _ = c.PostForm(r.FormValue("url"), nil) }
func SSRFBodyData(r *http.Request) {
	_, _ = http.Post("https://example.com", "text/plain", strings.NewReader(r.FormValue("data")))
}
func SSRFFormData(r *http.Request, c *http.Client) {
	_, _ = c.PostForm("https://example.com", url.Values{"data": {r.FormValue("data")}})
}
