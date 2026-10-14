package pdg

import "maps"

// EvidenceOnly keeps immutable file evidence without retaining obsolete typed
// function inventories. It shares immutable source strings and range arrays.
func (c *SourceContext) EvidenceOnly() *SourceContext {
	evidence := c.copyEvidence()
	for i := range evidence.files {
		evidence.files[i].SyntaxProfile, evidence.files[i].Syntax = "", ""
		evidence.files[i].Imports = nil
	}
	return evidence
}

// NextRevision preserves snapshot references for unchanged graphs. Reloaded
// slots are filled by the parser before the new revision is published.
func (c *SourceContext) NextRevision(reload map[string]bool) *SourceContext {
	next := c.copyEvidence()
	next.Disk = c.Disk
	next.vacancies = make(map[string]Ref)
	for path := range reload {
		if ref := next.paths[path]; ref != 0 {
			next.vacancies[path] = ref
			next.files[ref-1] = SourceSnapshot{}
			delete(next.paths, path)
		}
	}
	return next
}

func (c *SourceContext) copyEvidence() *SourceContext {
	return &SourceContext{Root: c.Root, Queries: append([]string(nil), c.Queries...), files: append([]SourceSnapshot(nil), c.files...), paths: maps.Clone(c.paths), missing: maps.Clone(c.missing)}
}
