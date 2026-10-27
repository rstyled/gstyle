package main

import "testing"

func testReport(functions int, findings ...diagnostic) checkReport {
	report := checkReport{}
	report.Summary.Functions = functions
	report.Diagnostics = findings
	return report
}

func finding(rule, name string, actual, limit int64) diagnostic {
	return diagnostic{RuleID: rule, Name: name, Actual: actual, Limit: limit}
}

func TestAggregateSumsExcess(t *testing.T) {
	rules := aggregate(testReport(100,
		finding("fn_params", "a", 6, 4), // excess 2
		finding("fn_params", "b", 5, 4), // excess 1
	))
	if got := rules["fn_params"].Excess; got != 3 {
		t.Fatalf("excess = %d, want 3", got)
	}
}

func TestAggregateSumsExcessAcrossPathLimits(t *testing.T) {
	rules := aggregate(testReport(100,
		finding("fn_length", "domain.a", 70, 60),
		finding("fn_length", "legacy.b", 130, 120),
	))
	if got := rules["fn_length"].Excess; got != 20 {
		t.Fatalf("excess = %d, want 20", got)
	}
}

func TestCheckBaselinePassesOnIdenticalReport(t *testing.T) {
	report := testReport(100, finding("fn_params", "a", 6, 4))
	violations := checkBaseline(writeBaseline(report, defaultTolerance), report)
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestCheckBaselineTripsOnNewFinding(t *testing.T) {
	base := writeBaseline(testReport(100, finding("fn_params", "a", 6, 4)), defaultTolerance)
	worse := testReport(100,
		finding("fn_params", "a", 6, 4),
		finding("fn_params", "b", 5, 4),
	)
	violations := checkBaseline(base, worse)
	if len(violations) != 1 || violations[0].rule != "fn_params" {
		t.Fatalf("expected one fn_params violation, got %v", violations)
	}
}

func TestCheckBaselineTripsOnWorsenedFinding(t *testing.T) {
	base := writeBaseline(testReport(100, finding("fn_params", "a", 6, 4)), defaultTolerance)
	worse := testReport(100, finding("fn_params", "a", 9, 4)) // excess 2 -> 5
	violations := checkBaseline(base, worse)
	if len(violations) != 1 {
		t.Fatalf("expected a violation for worsened excess, got %v", violations)
	}
}

func TestCheckBaselineTripsOnImprovement(t *testing.T) {
	// A past improvement (side_effect_density accuracy work) landed without
	// refreshing the baseline, leaving the ratchet loose. The gate must force
	// the refresh into the same PR.
	base := writeBaseline(testReport(100, finding("side_effect_density", "a", 2500, 500)), defaultTolerance)
	better := testReport(100, finding("side_effect_density", "a", 1500, 500)) // excess 2000 -> 1000
	violations := checkBaseline(base, better)
	if len(violations) != 1 || !violations[0].improved || violations[0].rule != "side_effect_density" {
		t.Fatalf("expected an improvement violation, got %v", violations)
	}
}

func TestCheckBaselineIgnoresNoise(t *testing.T) {
	// Small movements inside the tolerance band pass either way.
	base := writeBaseline(testReport(1000, finding("fn_params", "a", 24, 4)), defaultTolerance)
	quieter := testReport(1000, finding("fn_params", "a", 23, 4)) // rate 20 -> 19
	violations := checkBaseline(base, quieter)
	if len(violations) != 0 {
		t.Fatalf("expected no violations inside tolerance, got %v", violations)
	}
}

func TestCheckBaselineScalesWithGrowth(t *testing.T) {
	// Doubling the codebase with proportional findings keeps the rate flat.
	base := writeBaseline(testReport(100,
		finding("fn_params", "a", 6, 4),
		finding("fn_params", "b", 6, 4),
	), defaultTolerance)
	grown := testReport(200,
		finding("fn_params", "a", 6, 4),
		finding("fn_params", "b", 6, 4),
		finding("fn_params", "c", 6, 4),
		finding("fn_params", "d", 6, 4),
	)
	violations := checkBaseline(base, grown)
	if len(violations) != 0 {
		t.Fatalf("proportional growth should pass, got %v", violations)
	}
}

func TestCheckBaselineTripsOnUnbaselinedRule(t *testing.T) {
	base := writeBaseline(testReport(100, finding("fn_params", "a", 6, 4)), defaultTolerance)
	withNewRule := testReport(100,
		finding("fn_params", "a", 6, 4),
		finding("mutated_targets", "a", 5, 3),
	)
	violations := checkBaseline(base, withNewRule)
	if len(violations) != 1 || violations[0].rule != "mutated_targets" {
		t.Fatalf("expected a mutated_targets violation, got %v", violations)
	}
}

func TestCheckBaselinePassesNewFindingUnderPathLimit(t *testing.T) {
	base := writeBaseline(testReport(1000, finding("fn_length", "domain.a", 100, 60)), defaultTolerance)
	withLegacy := testReport(1000,
		finding("fn_length", "domain.a", 100, 60),
		finding("fn_length", "legacy.b", 121, 120),
	)
	if violations := checkBaseline(base, withLegacy); len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestCheckBaselineTripsOnLoosenedLimit(t *testing.T) {
	base := writeBaseline(testReport(100, finding("fn_params", "a", 7, 4)), defaultTolerance)
	loosened := testReport(100, finding("fn_params", "a", 7, 6))
	violations := checkBaseline(base, loosened)
	if len(violations) != 1 || !violations[0].improved {
		t.Fatalf("expected an improvement violation, got %v", violations)
	}
}

func TestWorstOffendersOrderedByExcess(t *testing.T) {
	diagnostics := []diagnostic{
		finding("fn_params", "small", 5, 4),
		finding("fn_params", "big", 9, 4),
		finding("fn_params", "mid", 6, 4),
	}
	worst := worstOffenders(diagnostics, "fn_params", 2)
	if len(worst) != 2 || worst[0].Name != "big" || worst[1].Name != "mid" {
		t.Fatalf("unexpected ordering: %v", worst)
	}
}
