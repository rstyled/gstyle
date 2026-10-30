package patterns

import (
	"reflect"
	"testing"
)

func TestPatternConfigDefaultsAndValidation(t *testing.T) {
	defaults := DefaultConfig()
	if err := defaults.Validate(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(defaults.MiningParams(), DefaultParams()) {
		t.Fatal("configuration changes existing mining defaults")
	}
	for _, invalid := range []func(*Config){
		func(c *Config) { c.SimilarityThreshold = 101 },
		func(c *Config) { c.MinCoverage = 101 },
		func(c *Config) { c.MaxHoles = -1 },
		func(c *Config) { c.Measurements.ThinSliceMinNodes = 0 },
		func(c *Config) { c.Measurements.BarrierSliceMinNodes = -1 },
		func(c *Config) { c.Measurements.ChopMinNodes = 0 },
		func(c *Config) { c.Measurements.ClusterMinNodes = 1 },
		func(c *Config) { c.Measurements.ClusterPercent = 101 },
		func(c *Config) { c.Measurements.MinSourceLines = -1 },
		func(c *Config) { c.Measurements.MinFunctionCalls = -1 },
	} {
		config := defaults
		invalid(&config)
		if config.Validate() == nil {
			t.Fatalf("invalid pattern config accepted: %+v", config)
		}
	}
}

func TestPatternPercentagesPreserveInternalScoring(t *testing.T) {
	for _, percent := range []uint32{0, 60, 80, 100} {
		config := DefaultConfig()
		config.SimilarityThreshold, config.MinCoverage = percent, percent
		config.Measurements.ClusterPercent = int(percent)
		if err := config.Validate(); err != nil {
			t.Fatal(err)
		}
		params := config.MiningParams()
		if params.ThresholdMilli != percent*10 || params.MinCoverageMilli != percent*10 {
			t.Fatalf("public percent changes internal precision: %+v", params)
		}
	}
	config := DefaultConfig()
	config.SimilarityThreshold, config.MinCoverage = 80, 80
	params := config.MiningParams()
	if params.ThresholdMilli != 800 || params.MinCoverageMilli != 800 || config.Measurements.ClusterPercent != 25 {
		t.Fatal("80% profile changes its effective thresholds")
	}
}

func TestMeasurementThresholdsUseCallsAndSourceLines(t *testing.T) {
	fact := sliceTestFact("chain", chainPdg(25))
	fact.Pdg.Nodes[1].Kind = Call
	limits := DefaultConfig().Measurements
	limits.ThinSliceMinNodes = 25
	limits.MinFunctionCalls, limits.MinSourceLines = 1, 25
	if len(dependencyMeasurementsWithLimits(fact, ThinSliceKind, limits)) != 1 {
		t.Fatal("inclusive measurement thresholds reject a matching slice")
	}
	for _, raise := range []func(*MeasurementLimits){
		func(l *MeasurementLimits) { l.ThinSliceMinNodes = 26 },
		func(l *MeasurementLimits) { l.MinFunctionCalls = 2 },
		func(l *MeasurementLimits) { l.MinSourceLines = 26 },
	} {
		stricter := limits
		raise(&stricter)
		if len(dependencyMeasurementsWithLimits(fact, ThinSliceKind, stricter)) != 0 {
			t.Fatalf("measurement ignores configured thresholds: %+v", stricter)
		}
	}
}

func TestClusterThresholdsAreConfigurable(t *testing.T) {
	graph := chainPdg(10)
	graph.Edges = append(graph.Edges, PdgEdge{From: 2, To: 0, Kind: Data})
	fact := sliceTestFact("cluster", graph)
	limits := DefaultConfig().Measurements
	if len(dependencyMeasurementsWithLimits(fact, DependenceCluster, limits)) != 1 {
		t.Fatal("fixture does not exercise the default relative cluster threshold")
	}
	limits.ClusterPercent = 50
	if len(dependencyMeasurementsWithLimits(fact, DependenceCluster, limits)) != 0 {
		t.Fatal("relative cluster threshold is ignored")
	}
	limits.ClusterPercent, limits.ClusterMinNodes = 0, 3
	if len(dependencyMeasurementsWithLimits(fact, DependenceCluster, limits)) != 1 {
		t.Fatal("absolute cluster threshold is ignored")
	}
}
