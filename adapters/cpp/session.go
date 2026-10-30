package cpp

import (
	"context"
	"sync"

	"github.com/rstyled/gstyle/domain/patterns"
)

type extractionIdentity struct {
	path, root string
}

type extractionSession struct {
	mu       sync.Mutex
	identity extractionIdentity
	snapshot *Extraction
}

func (s *extractionSession) store(identity extractionIdentity, snapshot *Extraction) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.identity, s.snapshot = identity, snapshot
}

func (s *extractionSession) take(identity extractionIdentity) *Extraction {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.identity != identity {
		return nil
	}
	snapshot := s.snapshot
	s.snapshot = nil
	return snapshot
}

func (a Analyzer) Session() Analyzer {
	a.session = &extractionSession{}
	return a
}

func (a Analyzer) PatternScan(ctx context.Context, paths []string, config patterns.Config) func() (*patterns.PatternsReport, error) {
	snapshot := a.session.take(extractionKey(paths, a.SourceRoot))
	return func() (*patterns.PatternsReport, error) {
		if err := config.Validate(); err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		selected := snapshot
		if selected == nil {
			var err error
			selected, err = ExtractFrom(ctx, paths, a.SourceRoot)
			if err != nil {
				return nil, err
			}
		}
		return selected.PatternsWithConfig(config, nil), nil
	}
}

func (a Analyzer) patternExtraction(ctx context.Context, paths []string) (*Extraction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if snapshot := a.session.take(extractionKey(paths, a.SourceRoot)); snapshot != nil {
		return snapshot, nil
	}
	return ExtractFrom(ctx, paths, a.SourceRoot)
}

func extractionKey(paths []string, root string) extractionIdentity {
	key := extractionIdentity{root: root}
	if len(paths) == 1 {
		key.path = paths[0]
	}
	return key
}
