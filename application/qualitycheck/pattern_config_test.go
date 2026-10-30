package qualitycheck

import "testing"

func TestSharedConfigLoadsPatternSettingsWithoutChangingQuality(t *testing.T) {
	path := writeTemp(t, "gstyle.toml", "[patterns]\nsimilarity_threshold = 85\nmin_coverage = 90\nmax_holes = 2\n[patterns.measurements]\nmin_source_lines = 3\nmin_function_calls = 2\n")
	settings, err := LoadPatternConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if settings.SimilarityThreshold != 85 || settings.MinCoverage != 90 || settings.MaxHoles != 2 || settings.Measurements.MinFunctionCalls != 2 || settings.Measurements.MinSourceLines != 3 || settings.Measurements.ThinSliceMinNodes != 10 {
		t.Fatalf("pattern configuration loses overrides or defaults: %+v", settings)
	}
	quality, err := LoadConfig(path)
	if err != nil || quality.FnParams.Max != 4 {
		t.Fatalf("pattern settings change the quality policy: %+v, %v", quality, err)
	}
}

func TestPatternConfigurationRejectsTyposAndInvalidValues(t *testing.T) {
	for _, source := range []string{
		"[patterns]\nsimilarity_threshold = 101\n",
		"[patterns]\nsimilarity_threshold = 800\n",
		"[patterns]\nsimilarity_threshold = -1\n",
		"[patterns]\nmin_coverag = 80\n",
		"[patterns.measurements]\ncluster_percent = 101\n",
		"[patterns.measurements]\ncluster_fraction_milli = 250\n",
		"[patterns.measurements]\nthin_slice_min_nodes = 0\n",
		"[patterns.measurements]\nmin_function_calls = -1\n",
		"[[overrides]]\npaths = ['cpp']\n[overrides.patterns]\nsimilarity_threshold = 80\n",
	} {
		path := writeTemp(t, "gstyle.toml", source)
		if _, err := LoadPatternConfig(path); err == nil {
			t.Fatalf("invalid pattern settings accepted: %s", source)
		}
		if _, err := LoadConfig(path); err == nil {
			t.Fatalf("quality command ignores invalid shared settings: %s", source)
		}
	}
}
