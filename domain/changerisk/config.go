package changerisk

import (
	"fmt"
	"slices"
	"strings"
)

func (c Config) Validate() error {
	if c.PDGLowRiskMaxFunctions < 0 {
		return fmt.Errorf("risk.pdg_low_risk_max_functions must not be negative")
	}
	if c.PDGLowRiskMaxFunctions > 0 && c.PDGChange == "" {
		return fmt.Errorf("risk.pdg_low_risk_max_functions requires pdg_change")
	}
	for _, level := range []Level{c.PDGChange, c.DependencyChange, c.APIBreaking} {
		if !validLevel(level) {
			return fmt.Errorf("risk levels must be low, medium, high, or omitted")
		}
	}
	return validateCallRules(c.Calls)
}

func validLevel(level Level) bool {
	return slices.Contains([]Level{"", Low, Medium, High}, level)
}

func validateCallRules(rules []CallRule) error {
	seen := map[string]bool{}
	for _, rule := range rules {
		if !validCallee(rule.Callee) || seen[rule.Callee] {
			return fmt.Errorf("risk.calls require distinct nonempty callee identities")
		}
		if !validLevel(rule.Level) || rule.Level == "" {
			return fmt.Errorf("risk.calls level must be low, medium, or high")
		}
		seen[rule.Callee] = true
	}
	return nil
}

func validCallee(callee string) bool {
	return callee != "" && strings.TrimSpace(callee) == callee
}

func configured(level Level) Level {
	if level == "" {
		return Unrated
	}
	return level
}

func strongest(a, b Level) Level {
	if levelScore(b) > levelScore(a) {
		return b
	}
	return a
}

func levelScore(level Level) int {
	return map[Level]int{Low: 1, Medium: 2, Unrated: 3, Unknown: 4, High: 5}[level]
}

func (c Config) enabled() bool {
	return c.PDGChange != "" || c.DependencyChange != "" || c.APIBreaking != "" || len(c.Calls) != 0
}
