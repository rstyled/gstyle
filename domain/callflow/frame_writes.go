package callflow

import "github.com/rstyled/gstyle/domain/bindingflow"

func directFrameWrite(write bindingflow.Write) bool {
	return !write.Unresolved && write.Symbol != 0 && write.Location == 0
}
