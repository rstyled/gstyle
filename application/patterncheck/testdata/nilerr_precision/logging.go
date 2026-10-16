package precision

import (
	"context"
	"fmt"
	"log"
	"log/slog"
)

type repairWriter struct {
	out   *error
	cause error
}

func (w repairWriter) Write(p []byte) (int, error) { *w.out = w.cause; return len(p), nil }

type stopWriter struct{}

func (stopWriter) Write([]byte) (int, error) { panic("writer callback") }

type repairFormatter struct {
	out   *error
	cause error
}

func (v repairFormatter) Format(fmt.State, rune) { *v.out = v.cause }

type repairHandler struct {
	out   *error
	cause error
}

func (h repairHandler) Enabled(context.Context, slog.Level) bool  { return true }
func (h repairHandler) Handle(context.Context, slog.Record) error { *h.out = h.cause; return nil }
func (h repairHandler) WithAttrs([]slog.Attr) slog.Handler        { return h }
func (h repairHandler) WithGroup(string) slog.Handler             { return h }

func WriterPropagates(cause error) (out error) {
	w := repairWriter{&out, cause}
	if cause != nil {
		fmt.Fprintln(w, cause)
		return
	}
	return nil
}
func WriterStops(cause error) error {
	if cause != nil {
		fmt.Fprintln(stopWriter{}, cause)
		return nil
	}
	return nil
}
func FormatterPropagates(cause error) (out error) {
	value := repairFormatter{&out, cause}
	if cause != nil {
		fmt.Printf("%v", value)
		return
	}
	return nil
}
func LogWriterPropagates(cause error) (out error) {
	log.SetOutput(repairWriter{&out, cause})
	if cause != nil {
		log.Print("failed")
		return
	}
	return nil
}
func SlogHandlerPropagates(cause error) (out error) {
	slog.SetDefault(slog.New(repairHandler{&out, cause}))
	if cause != nil {
		slog.Error("failed")
		return
	}
	return nil
}
