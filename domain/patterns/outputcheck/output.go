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
