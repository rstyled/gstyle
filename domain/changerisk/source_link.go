package changerisk

import (
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/url"
	"regexp"
	"strings"
)

var githubURL = regexp.MustCompile(`^https://github[.]com/[A-Za-z0-9_-]+/[A-Za-z0-9_.-]+$`)

func apiSourceLabel(report Report, declaration *APIDeclaration, revision, label string) string {
	label = codeSpan(label)
	link := sourceURL(report.RepositoryURL, declaration, revision)
	if link == "" {
		return label
	}
	return "[" + label + "](" + link + ")"
}

func codeSpan(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	length := 0
	for _, run := range strings.FieldsFunc(value, func(r rune) bool { return r != '`' }) {
		length = max(length, len(run))
	}
	delimiter := strings.Repeat("`", length+1)
	if length != 0 {
		value = " " + value + " "
	}
	return delimiter + value + delimiter
}

func reasonLocation(report Report, reason Reason) string {
	revision := report.Head
	if reason.Removed {
		revision = report.Base
	}
	label := reason.Path
	if reason.Line > 0 {
		label += fmt.Sprintf(":%d", reason.Line)
	}
	label = codeSpan(label)
	link := pinnedURL(report.RepositoryURL, revision, "blob", reason.Path, reason.Line)
	if link == "" {
		return label
	}
	return "[" + label + "](" + link + ")"
}

func sourceURL(repository string, declaration *APIDeclaration, revision string) string {
	if declaration == nil {
		return ""
	}
	kind := "tree"
	if declaration.Line > 0 {
		kind = "blob"
	}
	return pinnedURL(repository, revision, kind, declaration.Path, declaration.Line)
}

func pinnedURL(repository, revision, kind, path string, line int) string {
	if !githubURL.MatchString(repository) || !fs.ValidPath(path) || !fullRevision(revision) {
		return ""
	}
	fragment := ""
	if line > 0 {
		fragment = fmt.Sprintf("L%d", line)
	}
	link := url.URL{Scheme: "https", Host: "github.com", Path: strings.TrimPrefix(repository, "https://github.com") + "/" + kind + "/" + revision + "/" + path, Fragment: fragment}
	return strings.NewReplacer("(", "%28", ")", "%29").Replace(link.String())
}

func fullRevision(revision string) bool {
	if len(revision) != 40 && len(revision) != 64 {
		return false
	}
	_, err := hex.DecodeString(revision)
	return err == nil
}
