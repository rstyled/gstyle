package changerisk

type APIStatus string

const APIScope = "root_module_current_build"

const (
	APIUnchanged   APIStatus = "unchanged"
	APICompatible  APIStatus = "compatible"
	APIBreaking    APIStatus = "breaking"
	APIPartial     APIStatus = "partial"
	APIUnavailable APIStatus = "unavailable"
)

type APIChange struct {
	Package    string          `json:"package"`
	Message    string          `json:"message"`
	Compatible bool            `json:"compatible"`
	Before     *APIDeclaration `json:"before,omitempty"`
	After      *APIDeclaration `json:"after,omitempty"`
}

type APIDeclaration struct {
	Path        string `json:"path,omitempty"`
	Line        int    `json:"line,omitempty"`
	Declaration string `json:"declaration"`
	Exports     int    `json:"exports,omitempty"`
}

type APIReport struct {
	Scope          string      `json:"scope"`
	Status         APIStatus   `json:"status"`
	PackagesBefore int         `json:"packages_before"`
	PackagesAfter  int         `json:"packages_after"`
	Changes        []APIChange `json:"changes"`
	Limits         []string    `json:"limits"`
	Error          string      `json:"error,omitempty"`
}

func apiReasons(api APIReport, config Config) []Reason {
	if api.Status == APIUnavailable {
		return []Reason{{Rule: "go_api_unavailable", Level: Unknown, Message: api.Error}}
	}
	var reasons []Reason
	if len(api.Limits) != 0 {
		reasons = append(reasons, Reason{Rule: "go_api_partial", Level: Unknown, Message: api.Limits[0]})
	}
	for _, change := range api.Changes {
		if !change.Compatible {
			reasons = append(reasons, Reason{Rule: "go_api_breaking", Level: configured(config.APIBreaking), Message: change.Package + ": " + change.Message})
		}
	}
	return reasons
}

func unmeasuredFields(api APIReport) []string {
	if api.Status == "" {
		return []string{"go_api", "cli", "checks", "test_coverage"}
	}
	return []string{"cli", "checks", "test_coverage"}
}
