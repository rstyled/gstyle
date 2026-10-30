package cli

import (
	"github.com/rstyled/gstyle/application/qualitycheck"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/quality"
)

type runConfig struct {
	quality  quality.Config
	patterns patterns.Config
}

func loadRunConfig(options runOptions) (runConfig, error) {
	qualityConfig, err := qualitycheck.LoadAnalysisConfig(options.config, options.cpg)
	if err != nil {
		return runConfig{}, err
	}
	patternConfig, err := qualitycheck.LoadPatternConfig(options.config)
	return runConfig{quality: qualityConfig, patterns: patternConfig}, err
}
