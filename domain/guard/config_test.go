package guard

import "testing"

func TestDefaultGuardIsDisabled(t *testing.T) {
	config := Config{}
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := config.CheckPath("tool.py"); len(got) != 0 {
		t.Fatalf("default file restrictions: %+v", got)
	}
	if got := config.Comments.Check("file.go", []Comment{{Text: "// allowed"}}, nil); len(got) != 0 {
		t.Fatalf("default comment restrictions: %+v", got)
	}
}

func TestFileRulesUseDirectoryBoundaries(t *testing.T) {
	config := Config{Files: []FileRule{{ID: "python", Extensions: []string{".py", ".pyi"}, AllowDirectories: []string{"verification/lean"}}}}
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"build.py", "nested/build.PY", "verification/lean-copy/build.py", "verification/lean.py", "stub.pyi"} {
		if got := config.CheckPath(name); len(got) != 1 || got[0].RuleID != "python" {
			t.Errorf("%s: %+v", name, got)
		}
	}
	for _, name := range []string{"verification/lean/build.py", "verification/lean/tools/build.py", "build.go"} {
		if got := config.CheckPath(name); len(got) != 0 {
			t.Errorf("%s: %+v", name, got)
		}
	}
}

func TestInvalidGuardPolicies(t *testing.T) {
	for _, config := range []Config{
		{Comments: Comments{Mode: "typo"}},
		{Files: []FileRule{{Extensions: []string{".py"}}}},
		{Files: []FileRule{{ID: "source_comment", Extensions: []string{".py"}}}},
		{Files: []FileRule{{ID: "python"}}},
		{Files: []FileRule{{ID: "python", Extensions: []string{"*.py"}}}},
		{Files: []FileRule{{ID: "python", Extensions: []string{".tar.gz"}}}},
		{Files: []FileRule{{ID: "python", Extensions: []string{".py"}, AllowDirectories: []string{".."}}}},
		{Files: []FileRule{{ID: "python", Extensions: []string{".py"}, AllowDirectories: []string{"../outside"}}}},
		{Files: []FileRule{{ID: "python", Extensions: []string{".py"}, AllowDirectories: []string{"/absolute"}}}},
		{Files: []FileRule{{ID: "python", Extensions: []string{".py"}}, {ID: "python", Extensions: []string{".pyi"}}}},
	} {
		if err := config.Validate(); err == nil {
			t.Errorf("invalid policy accepted: %+v", config)
		}
	}
}

func TestCommentsPreserveExistingAndRejectExpansion(t *testing.T) {
	old := []Comment{{Text: "// original", Line: 1}, {Text: "/* block */", Line: 3}}
	current := []Comment{{Text: "// original", Line: 10}, {Text: "/* block expanded */", Line: 12}, {Text: "// original", Line: 15}}
	got := (Comments{Mode: "added"}).Check("file.go", current, old)
	if len(got) != 2 || got[0].Line != 12 || got[1].Line != 15 {
		t.Fatalf("new comment tokens: %+v", got)
	}
	if got := (Comments{Mode: "all"}).Check("file.go", current, old); len(got) != 3 {
		t.Fatalf("all comments: %+v", got)
	}
	if got := (Comments{}).Check("file.go", current, nil); len(got) != 0 {
		t.Fatalf("disabled comments: %+v", got)
	}
}
