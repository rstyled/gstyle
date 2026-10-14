package quality

import (
	"slices"
	"strings"
)

var ruleIDs = []string{"fn_length", "fn_params", "mutation_per_target", "mutated_targets", "side_effect_density", "invalid_suppression", "aggregate", "repository", "mutable_identity", "nilerr", "forcetypeassert", "typednil", "nil_guard", "negative_index", "literal_index"}

func suppression(comment string) ([]string, string) {
	comment = strings.TrimSpace(comment)
	// Support both // gstyle-allow(rules): reason and //lint:ignore rules reason.
	// The lint:ignore form is the standard Go ecosystem convention.
	if strings.HasPrefix(comment, "//lint:ignore") {
		return parseLintIgnore(comment)
	}
	if !strings.Contains(comment, "gstyle-allow") {
		return nil, ""
	}
	if !strings.HasPrefix(comment, "// gstyle-allow(") {
		return nil, "invalid suppression: expected // gstyle-allow(rules): reason"
	}
	rules, reason, ok := strings.Cut(strings.TrimPrefix(comment, "// gstyle-allow("), "):")
	if !ok || strings.TrimSpace(reason) == "" {
		return nil, "invalid suppression: reason is required"
	}
	return validateSuppressedRules(rules)
}

// parseLintIgnore parses `//lint:ignore rule1, rule2 reason text`.
func parseLintIgnore(comment string) ([]string, string) {
	rest := strings.TrimSpace(strings.TrimPrefix(comment, "//lint:ignore"))
	parts := strings.Fields(rest)
	rules, reasonParts := splitRulesReason(parts)
	if len(rules) == 0 {
		return nil, "invalid suppression: expected rule IDs"
	}
	if len(reasonParts) == 0 {
		return nil, "invalid suppression: reason is required"
	}
	return validateSuppressedRules(strings.Join(rules, ","))
}

// splitRulesReason splits fields into rule IDs (leading) and reason (rest).
func splitRulesReason(parts []string) ([]string, []string) {
	if len(parts) == 0 {
		return nil, nil
	}
	rules := strings.Split(strings.TrimSuffix(parts[0], ","), ",")
	i := 1
	for strings.HasSuffix(parts[i-1], ",") && i < len(parts) {
		rules = append(rules, strings.Split(strings.TrimSuffix(parts[i], ","), ",")...)
		i++
	}
	return rules, parts[i:]
}

func validateSuppressedRules(rules string) ([]string, string) {
	allowed := []string{}
	for _, id := range strings.Split(rules, ",") {
		id = strings.TrimSpace(id)
		if !slices.Contains(ruleIDs, id) {
			return nil, "invalid suppression: unknown rule id"
		}
		if id == "invalid_suppression" {
			return nil, "invalid suppression: invalid_suppression cannot be suppressed"
		}
		allowed = append(allowed, id)
	}
	return allowed, ""
}
