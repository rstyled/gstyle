package quality

import "testing"

func TestAuditLintIgnoreReasons(t *testing.T) {
	for _, line := range []string{"//lint:ignore fn_params public api contract", "//lint:ignore fn_params,fn_length Public API contract"} {
		if _, err := suppression(line); err != "" {
			t.Errorf("%q: %s", line, err)
		}
	}
}
