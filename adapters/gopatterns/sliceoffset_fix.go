package gopatterns

import (
	"crypto/sha256"
	"fmt"
	"go/format"
	"strconv"

	"github.com/rstyled/gstyle/domain/patterns"
)

func applySliceOffsetFix(spec *patterns.FixSpec, src []byte) ([]byte, error) {
	out, err := sliceOffsetRewrite(spec, src)
	if err != nil {
		return nil, err
	}
	return format.Source(out)
}

func sliceOffsetRewrite(spec *patterns.FixSpec, src []byte) ([]byte, error) {
	if spec.Params["source_sha256"] != fmt.Sprintf("%x", sha256.Sum256(src)) {
		return nil, fmt.Errorf("slice_offset_fallback: source changed; re-mine before fixing")
	}
	var edits []textEdit
	for _, name := range []string{"condition", "index"} {
		edit, err := sliceOffsetEdit(spec.Params, name, len(src))
		if err != nil {
			return nil, err
		}
		edits = append(edits, edit)
	}
	if edits[0].end > edits[1].start {
		return nil, fmt.Errorf("slice_offset_fallback: overlapping edits")
	}
	return applyEdits(src, edits), nil
}

func sliceOffsetEdit(params map[string]string, name string, size int) (textEdit, error) {
	start, err := strconv.Atoi(params[name+"_start"])
	if err != nil {
		return textEdit{}, err
	}
	end, err := strconv.Atoi(params[name+"_end"])
	if err != nil {
		return textEdit{}, err
	}
	if start < 0 || start >= end || end > size {
		return textEdit{}, fmt.Errorf("slice_offset_fallback: invalid %s edit bounds", name)
	}
	return textEdit{start: start, end: end, replacement: []byte(params[name])}, nil
}
