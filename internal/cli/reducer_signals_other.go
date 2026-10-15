//go:build !unix

package cli

import "os"

func reducerSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}
