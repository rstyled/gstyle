package patterns

// Models use full compiler-resolved identities. A suffix match can mistake an
// unrelated application method for a standard-library security boundary.
var taintSources = []string{
	"net/url.Values.Get", "net/http.Request.FormValue", "net/http.Request.PostFormValue",
	"net/http.Header.Get", "os.Getenv", "io.ReadAll", "io/ioutil.ReadAll",
}

type taintSink struct {
	callee        string
	dangerousArgs []int // Receiver is position 0; context is not query text.
	desc          string
}

var taintSinks = []taintSink{
	{"database/sql.DB.Exec", []int{1}, "SQL injection"},
	{"database/sql.DB.Query", []int{1}, "SQL injection"},
	{"database/sql.DB.QueryRow", []int{1}, "SQL injection"},
	{"database/sql.Tx.Exec", []int{1}, "SQL injection"},
	{"database/sql.Tx.Query", []int{1}, "SQL injection"},
	{"database/sql.Tx.QueryRow", []int{1}, "SQL injection"},
	{"database/sql.Conn.ExecContext", []int{2}, "SQL injection"},
	{"database/sql.Conn.QueryContext", []int{2}, "SQL injection"},
	{"database/sql.Conn.QueryRowContext", []int{2}, "SQL injection"},
	{"database/sql.DB.ExecContext", []int{2}, "SQL injection"},
	{"database/sql.DB.QueryContext", []int{2}, "SQL injection"},
	{"database/sql.DB.QueryRowContext", []int{2}, "SQL injection"},
	{"database/sql.Tx.ExecContext", []int{2}, "SQL injection"},
	{"database/sql.Tx.QueryContext", []int{2}, "SQL injection"},
	{"database/sql.Tx.QueryRowContext", []int{2}, "SQL injection"},
	{"database/sql.DB.Prepare", []int{1}, "SQL injection"},
	{"database/sql.DB.PrepareContext", []int{2}, "SQL injection"},
	{"database/sql.Tx.Prepare", []int{1}, "SQL injection"},
	{"database/sql.Tx.PrepareContext", []int{2}, "SQL injection"},
	{"database/sql.Conn.PrepareContext", []int{2}, "SQL injection"},
	{"os/exec.Command", []int{0}, "command injection"},
	{"os/exec.CommandContext", []int{1}, "command injection"},
	// Data passed to html/template.Execute is escaped. The template source
	// itself must be trusted for both template packages.
	{"html/template.Template.Parse", []int{1}, "template injection"},
	{"text/template.Template.Parse", []int{1}, "template injection"},
	{"os.Open", []int{0}, "path traversal"},
	{"os.OpenFile", []int{0}, "path traversal"},
	{"os.Create", []int{0}, "path traversal"},
	{"os.WriteFile", []int{0}, "path traversal"},
	{"io/ioutil.WriteFile", []int{0}, "path traversal"},
}

// SecurityFinding identifies the weakness family. It is advisory evidence,
// not a claim that the path is feasible or exploitable.
type SecurityFinding struct {
	RuleID string `json:"rule_id"`
	CWE    string `json:"cwe"`
}

var securityRules = map[string]SecurityFinding{
	"SQL injection":      {"go/sql-injection", "CWE-89"},
	"command injection":  {"go/command-injection", "CWE-78"},
	"template injection": {"go/template-injection", "CWE-1336"},
	"path traversal":     {"go/path-traversal", "CWE-22"},
}

func securityForRisk(risk string) *SecurityFinding {
	rule, ok := securityRules[risk]
	if !ok {
		return nil
	}
	return &rule
}
