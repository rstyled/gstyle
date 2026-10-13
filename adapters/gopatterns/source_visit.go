package gopatterns

import (
	"os"

	"golang.org/x/tools/go/packages"
)

type sourceVisit struct {
	config   *packages.Config
	query    []string
	overlay  map[string][]byte
	selected map[string]bool
	known    map[string]sourcePackageRecord
}

func visitSourcePackages(request sourceVisit, visit func([]*packages.Package) error) error {
	config, overlay := request.config, request.overlay
	if len(overlay) == 0 {
		return visitSourceBatches(config, request, visit)
	}
	directory, err := os.MkdirTemp("", "gstyle-source-overlay-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	path, err := writeBuildOverlay(directory, overlay)
	if err != nil {
		return err
	}
	next := *config
	next.BuildFlags = append(append([]string(nil), config.BuildFlags...), "-overlay="+path)
	next.ParseFile = overlaySourceParser(config.ParseFile, overlay)
	return visitSourceBatches(&next, request, visit)
}

func visitSourceBatches(config *packages.Config, request sourceVisit, visit func([]*packages.Package) error) error {
	query := request.query
	batches, err := sourcePackageBatches(config, query, request)
	if err != nil {
		return err
	}
	for _, batch := range batches {
		pkgs, err := packages.Load(config, batch...)
		if err != nil {
			return err
		}
		if err := patternPackageErrors(pkgs, config.Dir); err != nil {
			return err
		}
		if err := visit(pkgs); err != nil {
			return err
		}
	}
	return nil
}
