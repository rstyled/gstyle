package changerisk

import (
	"html"
	"strings"
	"testing"
)

func TestMermaidQuotedSourceUsesMermaidEntities(t *testing.T) {
	label := `data, err := command(ctx, root, "git", "diff", "--no-ext-diff")`
	delta := Delta{Name: "p.changedPaths", Path: "git.go", Added: []Node{{Key: "call", Label: label}, {Key: "return", Label: "return data, err"}},
		Edges: []ChangedEdge{{Edge: Edge{Key: "edge", From: "call", To: "return", Label: `value "data" & err`}, Change: "added"}}}
	diagram := mermaid(delta)
	if !strings.Contains(diagram, `command(ctx, root, #34;git#34;, #34;diff#34;, #34;--no-ext-diff#34;)`) {
		t.Fatalf("quoted Go source cannot survive Mermaid parsing: %s", diagram)
	}
	if !strings.Contains(diagram, `-->|"+ value #34;data#34; #38; err"|`) {
		t.Fatalf("edge label uses the wrong escaping context: %s", diagram)
	}
	if strings.Contains(diagram, "&#34;") || strings.Contains(diagram, "&amp;") {
		t.Fatalf("HTML entities leaked into Mermaid syntax: %s", diagram)
	}
	if decoded := html.UnescapeString(diagram); decoded != diagram {
		t.Fatalf("GitHub HTML preprocessing changes the Mermaid source: %s", decoded)
	}
}

func TestMermaidEntityTextCannotBecomeSyntax(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{`#quot;`, `#35;quot;`},
		{`&#34;`, `#38;#35;34;`},
		{"\"'&<>`\\|", "#34;#39;#38;#60;#62;#96;#92;#124;"},
		{"code\n\tcontinues", "code continues"},
	} {
		if got := mermaidText(tc.source); got != tc.want {
			t.Errorf("%q: got %q, want %q", tc.source, got, tc.want)
		}
	}
}
