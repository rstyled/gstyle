package changerisk

import "testing"

func TestDeclarationCoverageChangesRequireReview(t *testing.T) {
	for _, test := range []struct {
		name          string
		before, after map[string]string
	}{
		{"modified", map[string]string{"source.go": "before"}, map[string]string{"source.go": "after"}},
		{"added", nil, map[string]string{"source.go": "after"}},
		{"removed", map[string]string{"source.go": "before"}, nil},
		{"empty addition", nil, map[string]string{"source.go": ""}},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := Snapshot{Declarations: test.before}
			head := Snapshot{Declarations: test.after}
			report := Evaluate(base, head, Config{PDGChange: Low}, APIReport{})
			if report.Risk != Unknown || report.PDGRisk != Unknown || report.ChangedFunctions != 0 {
				t.Fatalf("declaration gap received a conclusive rating: %+v", report)
			}
			if len(report.Reasons) != 1 || report.Reasons[0].Rule != "analysis_unavailable" || report.Reasons[0].Path != "source.go" {
				t.Fatalf("declaration evidence missing: %+v", report.Reasons)
			}
			if report := Evaluate(base, head, Config{}, APIReport{}); report.Risk != Unrated || report.PDGRisk != Unknown {
				t.Fatalf("missing policy was treated as configured: %+v", report)
			}
		})
	}
}

func TestDeclarationCoverageKeepsConfiguredBodyRatings(t *testing.T) {
	base := Snapshot{Declarations: map[string]string{"source.go": "same"},
		Functions: []Function{{Name: "p.Count", Path: "source.go", Body: "old", Graph: Graph{Shape: "old"}}}}
	head := Snapshot{Declarations: map[string]string{"source.go": "same"},
		Functions: []Function{{Name: "p.Count", Path: "source.go", Body: "new", Graph: Graph{Shape: "new"}}}}
	report := Evaluate(base, head, Config{PDGChange: Low}, APIReport{})
	if report.Risk != Low || report.ChangedFunctions != 1 || len(report.Reasons) != 1 || report.Reasons[0].Rule != "pdg_change" {
		t.Fatalf("covered body change lost its configured rating: %+v", report)
	}
	if report := Evaluate(base, head, Config{}, APIReport{}); report.Risk != Unrated {
		t.Fatalf("unconfigured body change received a rating: %+v", report)
	}
}

func TestDeclarationCoverageReasonsAreStableAcrossPaths(t *testing.T) {
	report := Evaluate(Snapshot{Declarations: map[string]string{"old.go": "same"}},
		Snapshot{Declarations: map[string]string{"new.go": "same"}}, Config{PDGChange: Low}, APIReport{})
	if report.Risk != Unknown || len(report.Reasons) != 2 || report.Reasons[0].Path != "new.go" || report.Reasons[1].Path != "old.go" {
		t.Fatalf("rename coverage disappeared or became unstable: %+v", report)
	}
}
