package application_test

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/rstyled/gstyle/adapters/cpp"
	"github.com/rstyled/gstyle/application"
)

func TestCPGTUI(t *testing.T) {
	model := application.NewModel(cpp.Analyzer{Context: context.Background()}, []string{"../adapters/cpp/testdata/sample.graphson.json"})
	loaded, _ := model.Update(model.Init()())
	sized, _ := loaded.Update(tea.WindowSizeMsg{Width: 160, Height: 48})
	view := ansi.Strip(sized.View().Content)
	for _, text := range []string{"sample.cpp", "CC", "COG"} {
		if !strings.Contains(view, text) {
			t.Fatalf("missing %q: %s", text, view)
		}
	}
	quality, _ := sized.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	if strings.Contains(ansi.Strip(quality.View().Content), "Quality unavailable") {
		t.Fatal("TUI does not run shared quality checks for C++")
	}
}
