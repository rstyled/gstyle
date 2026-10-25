package changerisk

type Level string

const (
	Low     Level = "low"
	Medium  Level = "medium"
	High    Level = "high"
	Unknown Level = "unknown"
	Unrated Level = "unrated"
)

type CallRule struct {
	Callee string `toml:"callee" json:"callee"`
	Level  Level  `toml:"level" json:"level"`
}

type Config struct {
	PDGChange        Level      `toml:"pdg_change"`
	DependencyChange Level      `toml:"dependency_change"`
	Calls            []CallRule `toml:"calls"`
}

type Node struct {
	Key            string `json:"key"`
	Label          string `json:"label"`
	Callee         string `json:"callee,omitempty"`
	Line           int    `json:"line"`
	UnresolvedCall bool   `json:"unresolved_call,omitempty"`
}

type Edge struct {
	Key   string `json:"key"`
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"`
}

type Graph struct {
	Shape     string
	Nodes     []Node
	Edges     []Edge
	Ambiguous bool
}

type Function struct {
	ID    string
	Name  string
	Path  string
	Line  int
	Body  string
	Graph Graph
}

type Snapshot struct {
	Commit          string
	Functions       []Function
	Dependencies    []string
	DependencyError string
	AnalysisError   string
}

type Reason struct {
	Rule     string `json:"rule"`
	Level    Level  `json:"level"`
	Path     string `json:"path,omitempty"`
	Function string `json:"function,omitempty"`
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message"`
	Callee   string `json:"callee,omitempty"`
}

type Delta struct {
	Name      string        `json:"function"`
	Path      string        `json:"path"`
	Line      int           `json:"line"`
	Added     []Node        `json:"added"`
	Removed   []Node        `json:"removed"`
	Context   []Node        `json:"context"`
	Edges     []ChangedEdge `json:"edges"`
	Ambiguous bool          `json:"ambiguous"`
}

type ChangedEdge struct {
	Edge
	Change string `json:"change"`
}

type Report struct {
	SchemaVersion       int      `json:"schema_version"`
	Base                string   `json:"base"`
	Head                string   `json:"head"`
	Policy              string   `json:"policy"`
	Risk                Level    `json:"risk"`
	PDGRisk             Level    `json:"pdg_risk"`
	DependencyRisk      Level    `json:"dependency_risk"`
	DependenciesChanged bool     `json:"dependencies_changed"`
	DependenciesAdded   []string `json:"dependencies_added"`
	DependenciesRemoved []string `json:"dependencies_removed"`
	Reasons             []Reason `json:"reasons"`
	Changes             []Delta  `json:"changes"`
	ChangedFunctions    int      `json:"changed_functions"`
	Unmeasured          []string `json:"unmeasured"`
}
