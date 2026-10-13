package pdg

import (
	"crypto/sha256"
	"fmt"
	"sort"
)

// ByteRange indexes the original bytes of a source snapshot. End is exclusive.
type ByteRange struct{ Start, End uint32 }

// SourceCommentOwner keeps declaration ownership without copying comment text.
type SourceCommentOwner struct {
	Range ByteRange
	Owner string
}

// SourceSnapshot owns one immutable copy of a file, shared by all its graphs.
// Comments contain actual parser comment ranges, never comment-like literals.
type SourceSnapshot struct {
	SyntaxProfile string `json:"-"`
	Syntax        string `json:"-"`
	Owners        []SourceCommentOwner
	Path          string
	Content       string
	Identity      [32]byte
	Comments      []ByteRange
}

// HasComment reports intersection with parser-owned comment ranges.
func (s SourceSnapshot) HasComment(span ByteRange) bool {
	for _, comment := range s.Comments {
		if comment.Start < span.End && span.Start < comment.End {
			return true
		}
	}
	return false
}

// SourceContext contains source evidence for one extraction revision.
// It does not retain compiler objects. File indexes are one-based.
type SourceContext struct {
	// Preview inputs retain the original disk evidence and cumulative edits.
	Disk       *SourceContext    `json:"-"`
	Overlay    map[string][]byte `json:"-"`
	Functions  []SourceFunction
	References []SourceReference
	Root       string
	Queries    []string
	files      []SourceSnapshot
	paths      map[string]Ref
	missing    map[string]bool
	vacancies  map[string]Ref
}

func NewSourceContext(root string, queries []string) *SourceContext {
	return &SourceContext{Root: root, Queries: append([]string(nil), queries...), paths: make(map[string]Ref), missing: make(map[string]bool)}
}

// Add publishes a snapshot. Construction must finish before downstream use.
func (c *SourceContext) Add(path string, content []byte, comments []ByteRange) Ref {
	return c.AddOwned(path, content, comments, nil)
}

func (c *SourceContext) AddOwned(path string, content []byte, comments []ByteRange, owners []SourceCommentOwner) Ref {
	if ref := c.paths[path]; ref != 0 {
		return ref
	}
	snapshot := SourceSnapshot{Path: path, Content: string(content), Identity: sha256.Sum256(content), Comments: append([]ByteRange(nil), comments...), Owners: append([]SourceCommentOwner(nil), owners...)}
	if ref := c.vacancies[path]; ref != 0 {
		c.files[ref-1] = snapshot
		c.paths[path] = ref
		delete(c.vacancies, path)
		return ref
	}
	c.files = append(c.files, snapshot)
	ref := Ref(len(c.files))
	c.paths[path] = ref
	return ref
}

func (c *SourceContext) Reference(path string) Ref { return c.paths[path] }

func (c *SourceContext) Snapshot(ref Ref) (SourceSnapshot, bool) {
	if ref == 0 || uint64(ref) > uint64(len(c.files)) {
		return SourceSnapshot{}, false
	}
	snapshot := c.files[ref-1]
	snapshot.Comments = append([]ByteRange(nil), snapshot.Comments...)
	snapshot.Owners = append([]SourceCommentOwner(nil), snapshot.Owners...)
	return snapshot, true
}

func (c *SourceContext) Paths() []string {
	paths := make([]string, len(c.files))
	for i, file := range c.files {
		paths[i] = file.Path
	}
	return paths
}

func (c *SourceContext) Check(path string, content []byte) error {
	ref := c.Reference(path)
	if ref == 0 {
		return fmt.Errorf("source evidence is missing: %s", path)
	}
	if sha256.Sum256(content) != c.files[ref-1].Identity {
		return fmt.Errorf("source changed after extraction: %s", path)
	}
	return nil
}

// Source returns immutable text without copying range tables.
func (c *SourceContext) Source(ref Ref) (string, string, bool) {
	if ref == 0 || uint64(ref) > uint64(len(c.files)) {
		return "", "", false
	}
	file := c.files[ref-1]
	return file.Path, file.Content, true
}

func (c *SourceContext) HasComment(ref Ref, span ByteRange) bool {
	if ref == 0 || uint64(ref) > uint64(len(c.files)) {
		return false
	}
	return c.files[ref-1].HasComment(span)
}

// RecordMissing records build inputs whose absence affects this revision.
func (c *SourceContext) RecordMissing(path string) { c.missing[path] = true }

func (c *SourceContext) MissingPaths() []string {
	paths := make([]string, 0, len(c.missing))
	for path := range c.missing {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

// SetSyntax publishes a language-specific compact syntax projection. It is
// shared by a revision and never retains parser or compiler objects.
func (c *SourceContext) SetSyntax(ref Ref, profile, data string) error {
	if ref == 0 || uint64(ref) > uint64(len(c.files)) {
		return fmt.Errorf("source evidence is missing")
	}
	if c.files[ref-1].Syntax != "" {
		return fmt.Errorf("source syntax is already published")
	}
	c.files[ref-1].SyntaxProfile, c.files[ref-1].Syntax = profile, data
	return nil
}
