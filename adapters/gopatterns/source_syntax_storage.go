package gopatterns

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"sync"
)

type syntaxCompressor struct {
	output bytes.Buffer
	writer *flate.Writer
}

// Reuse the compressor workspace across files. The retained source inventory
// contains only compact bytes; it never owns this encoder workspace.
var syntaxCompressors = sync.Pool{New: func() any {
	compressor := &syntaxCompressor{}
	// BestSpeed is a constant supported by compress/flate.
	compressor.writer, _ = flate.NewWriter(&compressor.output, flate.BestSpeed)
	return compressor
}}

func compactSourceSyntax(data []byte) (string, error) {
	compressor := syntaxCompressors.Get().(*syntaxCompressor)
	defer syntaxCompressors.Put(compressor)
	compressor.output.Reset()
	compressor.writer.Reset(&compressor.output)
	if _, err := compressor.writer.Write(data); err != nil {
		return "", err
	}
	if err := compressor.writer.Close(); err != nil {
		return "", err
	}
	header := binary.AppendUvarint(nil, uint64(len(data)))
	return string(header) + compressor.output.String(), nil
}

func inflateSourceSyntax(data string, sourceSize int) (string, error) {
	size, header := binary.Uvarint([]byte(data))
	if header <= 0 || size > uint64(sourceSize)*256+4096 {
		return "", fmt.Errorf("invalid syntax storage size")
	}
	reader := flate.NewReader(strings.NewReader(data[header:]))
	defer reader.Close()
	result, err := io.ReadAll(io.LimitReader(reader, int64(size)+1))
	if err != nil {
		return "", err
	}
	if uint64(len(result)) != size {
		return "", fmt.Errorf("syntax storage size differs")
	}
	return string(result), nil
}
