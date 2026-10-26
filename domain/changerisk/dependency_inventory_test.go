package changerisk

import "testing"

func TestDependencyInventoryTracksPackagesPerModuleWithoutCountingCallers(t *testing.T) {
	for _, test := range []struct {
		name          string
		before, after []string
		changed       bool
	}{
		{"already used", []string{".:import:old -> reflect"}, []string{".:import:old -> reflect", ".:import:new -> reflect"}, false},
		{"caller moved", []string{".:import:old -> reflect"}, []string{".:import:new -> reflect"}, false},
		{"test build", []string{".:import:p -> reflect"}, []string{".:import:p -> reflect", ".:import:p [p.test] -> reflect"}, false},
		{"one caller removed", []string{".:import:a -> reflect", ".:import:b -> reflect"}, []string{".:import:b -> reflect"}, false},
		{"order and duplicates", []string{".:import:a -> reflect", ".:import:b -> fmt"}, []string{".:import:z -> fmt", ".:import:z -> reflect", ".:import:z -> reflect"}, false},
		{"first use", nil, []string{".:import:p -> example.com/lib"}, true},
		{"last use removed", []string{".:import:p -> example.com/lib"}, nil, true},
		{"other module", []string{".:import:p -> reflect"}, []string{".:import:p -> reflect", "tools:import:p -> reflect"}, true},
		{"module version", []string{`.:module:"example.com/lib":"v1.0.0":"1.26"`}, []string{`.:module:"example.com/lib":"v1.1.0":"1.26"`}, true},
		{"module graph", []string{".:main old@v1"}, []string{".:main new@v1"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			report := Evaluate(Snapshot{Dependencies: test.before}, Snapshot{Dependencies: test.after}, Config{DependencyChange: High}, APIReport{})
			want := Low
			if test.changed {
				want = High
			}
			if report.DependenciesChanged != test.changed || report.DependencyRisk != want {
				t.Fatalf("wrong inventory delta: %+v", report)
			}
			if !test.changed && len(report.DependenciesAdded)+len(report.DependenciesRemoved) != 0 {
				t.Fatalf("caller changes leaked as added dependencies: %+v", report)
			}
		})
	}
}

func TestDependencyDeltasExcludeExistingImportsWhenAModuleVersionChanges(t *testing.T) {
	before := []string{".:import:old -> reflect", `.:module:"example.com/lib":"v1.0.0":"1.26"`}
	after := []string{".:import:old -> reflect", ".:import:new -> reflect", `.:module:"example.com/lib":"v1.1.0":"1.26"`}
	report := Evaluate(Snapshot{Dependencies: before}, Snapshot{Dependencies: after}, Config{DependencyChange: High}, APIReport{})
	if !report.DependenciesChanged || len(report.DependenciesAdded) != 1 || len(report.DependenciesRemoved) != 1 {
		t.Fatalf("existing import usage counted as a new dependency: %+v", report)
	}
}
