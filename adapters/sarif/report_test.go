package sarif

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestReportLocations(t *testing.T) {
	var output bytes.Buffer
	findings := []Finding{{RuleID: "nilerr", Message: "bad return", Locations: []Location{
		{Path: "dir/a #é.go", Line: 3},
		{Path: "/tmp/other.go", Line: 5},
	}}}
	if err := Write(&output, findings); err != nil {
		t.Fatal(err)
	}
	var report log
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	item := report.Runs[0].Results[0]
	if item.RuleID != "nilerr" || item.Message.Text != "bad return" {
		t.Fatalf("lost finding: %+v", item)
	}
	source := item.Locations[0].PhysicalLocation
	if source.Artifact.URI != "dir/a%20%23%C3%A9.go" || source.Region.Line != 3 {
		t.Fatalf("bad primary location: %+v", source)
	}
	if len(item.Related) != 1 || item.Related[0].PhysicalLocation.Artifact.URI != "file:///tmp/other.go" {
		t.Fatalf("bad related locations: %+v", item.Related)
	}
}

func TestEmptyReportAndUnknownPosition(t *testing.T) {
	for _, findings := range [][]Finding{nil, {{RuleID: "rule", Message: "message", Locations: []Location{{Path: "a.go"}}}}} {
		var output bytes.Buffer
		if err := Write(&output, findings); err != nil {
			t.Fatal(err)
		}
		var report log
		if err := json.Unmarshal(output.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
		if report.Runs[0].Results == nil {
			t.Fatal("results must be an array")
		}
		for _, result := range report.Runs[0].Results {
			if result.Locations[0].PhysicalLocation.Region != nil {
				t.Fatal("unknown position must omit region")
			}
		}
	}
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }
func TestWriterError(t *testing.T) {
	if err := Write(failedWriter{}, nil); err == nil {
		t.Fatal("missing IO error")
	}
}
