package gopatterns

type outputDecision struct {
	Formatted  bool
	Memory     bool
	Standalone bool
	Values     int
	Results    int
	ErrorBlank bool
}

// @ ensures result == (input.Formatted && !input.Memory && (input.Standalone || (input.Values == 1 && input.Results == 2 && input.ErrorBlank)))
// @ decreases
func discardsOutputDecision(input outputDecision) (result bool) {
	if !input.Formatted || input.Memory {
		return false
	}
	if input.Standalone {
		return true
	}
	return input.Values == 1 && input.Results == 2 && input.ErrorBlank
}
