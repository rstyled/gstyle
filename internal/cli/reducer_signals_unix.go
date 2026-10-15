//go:build unix

package cli

import (
	"os"
	"syscall"
)

func reducerSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
