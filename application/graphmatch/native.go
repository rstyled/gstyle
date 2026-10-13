package graphmatch

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/domain/ccgraph"
)

func runNative(ctx context.Context, opts options, output io.Writer) error {
	config, err := loadConfig(opts.config)
	if err != nil {
		return err
	}
	input, err := gopatterns.ExtractMatching(ctx, opts.source, []string{"."})
	if err != nil {
		return err
	}
	if err := writeNativeContext(opts.contextOutput, input); err != nil {
		return err
	}
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(struct {
		Type     string                     `json:"type"`
		Manifest *gopatterns.NativeMatching `json:"manifest"`
	}{"native-manifest", input}); err != nil {
		return err
	}
	run, matchErr := ccgraph.Evaluate(ctx, input.Functions, config, func(pair ccgraph.Pair) error { return encoder.Encode(record{Type: "pair", Pair: &pair}) })
	run.InputScope = "validated_native_source"
	run.Ineligible = append(input.Ineligible, run.Ineligible...)
	return finish(encoder, run, matchErr)
}

func writeNativeContext(path string, input *gopatterns.NativeMatching) error {
	if path == "" {
		return nil
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if err := input.WriteContext(file); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
