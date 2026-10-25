package cli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/application/repoguard"
)

func TestGuardCommandDispatchAndExitCodes(t *testing.T) {
	root := t.TempDir()
	command := exec.Command("git", "init", root)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, output)
	}
	if err := os.WriteFile(filepath.Join(root, "gstyle.toml"), []byte("[[guard.files]]\nid='python'\nextensions=['.py']\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tool.py"), []byte("pass\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	var output bytes.Buffer
	err := run([]string{"guard"}, &output)
	if !errors.Is(err, repoguard.ErrFindings) || commandExitCode([]string{"guard"}, err) != 1 || !strings.Contains(output.String(), "tool.py") {
		t.Fatalf("guard: %v %s", err, output.String())
	}
	if err := run([]string{"guard", "--config", "missing.toml"}, &output); err == nil || commandExitCode([]string{"guard"}, err) != 2 {
		t.Fatalf("configuration error: %v", err)
	}
}
