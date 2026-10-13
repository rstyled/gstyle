package gopatterns

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// NativeBuild pins the toolchain and effective target used by package loading.
// It is captured once for an explicit native-conformance run.
type NativeBuild struct {
	GOOS        string
	GOARCH      string
	GOVERSION   string
	GOFLAGS     string
	CGO_ENABLED string
}

func nativeBuild(ctx context.Context, root string) (NativeBuild, error) {
	command := exec.CommandContext(ctx, "go", "env", "-json", "GOOS", "GOARCH", "GOVERSION", "GOFLAGS", "CGO_ENABLED")
	command.Dir = root
	data, err := command.Output()
	if err != nil {
		return NativeBuild{}, fmt.Errorf("capture Go build target: %w", err)
	}
	var build NativeBuild
	if err := json.Unmarshal(data, &build); err != nil {
		return NativeBuild{}, err
	}
	if build.GOOS == "" || build.GOARCH == "" || build.GOVERSION == "" {
		return NativeBuild{}, fmt.Errorf("Go build identity is incomplete")
	}
	return build, nil
}
