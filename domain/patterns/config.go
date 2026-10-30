package patterns

import "fmt"

type Config struct {
	SimilarityThreshold uint32            `toml:"similarity_threshold" json:"similarity_threshold"`
	MinCoverage         uint32            `toml:"min_coverage" json:"min_coverage"`
	MaxHoles            int               `toml:"max_holes" json:"max_holes"`
	Measurements        MeasurementLimits `toml:"measurements" json:"measurements"`
}

type MeasurementLimits struct {
	ThinSliceMinNodes    int `toml:"thin_slice_min_nodes" json:"thin_slice_min_nodes"`
	BarrierSliceMinNodes int `toml:"barrier_slice_min_nodes" json:"barrier_slice_min_nodes"`
	ChopMinNodes         int `toml:"chop_min_nodes" json:"chop_min_nodes"`
	ClusterMinNodes      int `toml:"cluster_min_nodes" json:"cluster_min_nodes"`
	ClusterPercent       int `toml:"cluster_percent" json:"cluster_percent"`
	MinSourceLines       int `toml:"min_source_lines" json:"min_source_lines"`
	MinFunctionCalls     int `toml:"min_function_calls" json:"min_function_calls"`
}

func DefaultConfig() Config {
	params := DefaultParams()
	return Config{SimilarityThreshold: params.ThresholdMilli / 10, MinCoverage: params.MinCoverageMilli / 10, MaxHoles: params.MaxHoles,
		Measurements: MeasurementLimits{ThinSliceMinNodes: minThinSliceNodes, BarrierSliceMinNodes: minBarrierSliceNodes, ChopMinNodes: minChopNodes, ClusterMinNodes: minClusterSize, ClusterPercent: int(clusterSizeFraction * 100)}}
}

func (config Config) MiningParams() Params {
	params := DefaultParams()
	params.ThresholdMilli, params.MinCoverageMilli, params.MaxHoles = config.SimilarityThreshold*10, config.MinCoverage*10, config.MaxHoles
	return params
}

func (config Config) Validate() error {
	if config.SimilarityThreshold > 100 || config.MinCoverage > 100 {
		return fmt.Errorf("patterns similarity_threshold and min_coverage must be between 0 and 100 percent")
	}
	if config.MaxHoles < 0 {
		return fmt.Errorf("patterns max_holes must be nonnegative")
	}
	return config.Measurements.Validate()
}

func (limits MeasurementLimits) Validate() error {
	for _, field := range []struct {
		name           string
		value, minimum int
	}{
		{"thin_slice_min_nodes", limits.ThinSliceMinNodes, 1},
		{"barrier_slice_min_nodes", limits.BarrierSliceMinNodes, 1},
		{"chop_min_nodes", limits.ChopMinNodes, 1},
		{"cluster_min_nodes", limits.ClusterMinNodes, 2},
		{"min_source_lines", limits.MinSourceLines, 0},
		{"min_function_calls", limits.MinFunctionCalls, 0},
	} {
		if field.value < field.minimum {
			return fmt.Errorf("patterns.measurements.%s must be at least %d", field.name, field.minimum)
		}
	}
	if limits.ClusterPercent < 0 || limits.ClusterPercent > 100 {
		return fmt.Errorf("patterns.measurements.cluster_percent must be between 0 and 100")
	}
	return nil
}

func defaultMeasurementLimits(limits MeasurementLimits) MeasurementLimits {
	if limits == (MeasurementLimits{}) {
		return DefaultConfig().Measurements
	}
	return limits
}
