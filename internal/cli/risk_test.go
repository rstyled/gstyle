package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestRiskHelpAndOperationalExit(t *testing.T) {
	var output bytes.Buffer
	if err := run([]string{"risk", "--help"}, &output); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--base", "--head", "--config", "--format"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("missing %s in help: %s", want, output.String())
		}
	}
	if got := commandExitCode([]string{"risk"}, errors.New("failure")); got != 2 {
		t.Fatalf("operational failure exit=%d", got)
	}
}
