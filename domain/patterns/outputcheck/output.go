package outputcheck

type Decision struct {
	Formatted  bool
	Memory     bool
	Standalone bool
	Values     int
	Results    int
	ErrorBlank bool
}

func Discards(input Decision) (result bool) {
	if !input.Formatted || input.Memory {
		return false
	}
	if input.Standalone {
		return true
	}
	return input.Values == 1 && input.Results == 2 && input.ErrorBlank
}

type Site struct {
	FuncID          string
	Line            int
	Callee          string
	Checked         bool
	DiscardedOutput bool `json:",omitempty"`
}

func Select(sites []Site, suppressed bool) []Site {
	var out []Site
	if suppressed {
		return out
	}
	for i := 0; i < len(sites); i++ {
		site := sites[i]
		if site.DiscardedOutput {
			out = append(out, site)
		}
	}
	return out
}

func Formatted(pkg, name string) bool {
	return pkg == "fmt" && (name == "Fprint" || name == "Fprintf" || name == "Fprintln")
}

func Memory(pkg, name string) bool {
	return (pkg == "bytes" && name == "Buffer") || (pkg == "strings" && name == "Builder")
}

func hasSuppression(kinds []string) bool {
	for i := 0; i < len(kinds); i++ {
		if kinds[i] == "unchecked_output" {
			return true
		}
	}
	return false
}

func Find(sites []Site, suppressions []string) []Site {
	return Select(sites, hasSuppression(suppressions))
}

type Function struct {
	Path string
	ID   string
	Name string
}

type Advisory struct {
	Path            string
	Line            int
	EndLine         int
	ID              string
	Name            string
	Kind            string
	ScoreMilli      uint32
	Observation     string
	Inference       string
	CounterEvidence string
}

func describe(function Function, site Site) Advisory {
	return Advisory{
		Path: function.Path, Line: site.Line, EndLine: site.Line,
		ID: function.ID, Name: function.Name,
		Kind: "unchecked_output", ScoreMilli: 800,
		Observation:     "formatted output error is discarded: " + site.Callee,
		Inference:       "a failed write can be reported as success; propagate or explicitly handle the output error",
		CounterEvidence: "best-effort output may deliberately ignore failures; no automatic fix",
	}
}

func Report(function Function, sites []Site, suppressions []string) []Advisory {
	selected := Find(sites, suppressions)
	var out []Advisory
	for i := 0; i < len(selected); i++ {
		out = append(out, describe(function, selected[i]))
	}
	return out
}
