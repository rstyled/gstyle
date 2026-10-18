package callflow

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/deferflow"
)

// Saved capture contracts and invocation receiving contracts describe the same
// registration. Neither record may silently override a contradictory record.
func (calls *CallSites) validateDeferredInputs() error {
	index, err := deferflow.NewIndex(calls.graph)
	if err != nil {
		return err
	}
	calls.deferred = index
	for _, site := range index.Sites() {
		shape, found := calls.paths.CallShapeAt(site.Invocation)
		if !found {
			continue
		}
		if shape.Spread != site.Spread || (shape.Tuple != 0) != site.Tuple {
			return fmt.Errorf("call shape contradicts deferred capture contract")
		}
	}
	return nil
}
