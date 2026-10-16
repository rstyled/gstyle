package precision

import (
	"errors"
	"log"
	"log/slog"
	"testing"
)

func TestLoggingCallbacks(t *testing.T) {
	logger, writer, flags, prefix := slog.Default(), log.Writer(), log.Flags(), log.Prefix()
	t.Cleanup(func() { slog.SetDefault(logger); log.SetOutput(writer); log.SetFlags(flags); log.SetPrefix(prefix) })
	cause := errors.New("failed")
	for _, call := range []func(error) error{WriterPropagates, FormatterPropagates, LogWriterPropagates, SlogHandlerPropagates} {
		if got := call(cause); got != cause {
			t.Fatalf("callback returned %v, want original error", got)
		}
	}
}
func TestWriterStopsBeforeReturn(t *testing.T) {
	cause := errors.New("failed")
	defer func() {
		if got := recover(); got != "writer callback" {
			t.Fatalf("panic=%v,want writer callback", got)
		}
	}()
	_ = WriterStops(cause)
	t.Fatal("nil return is unreachable")
}
