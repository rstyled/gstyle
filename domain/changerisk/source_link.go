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

func sourceURL(repository string, declaration *APIDeclaration, revision string) string {
	if declaration == nil || !githubURL.MatchString(repository) || !fs.ValidPath(declaration.Path) {
		return ""
	}
	if !fullRevision(revision) {
		return ""
	}
	kind, fragment := "tree", ""
	if declaration.Line > 0 {
		kind, fragment = "blob", fmt.Sprintf("L%d", declaration.Line)
	}
	link := url.URL{Scheme: "https", Host: "github.com", Path: strings.TrimPrefix(repository, "https://github.com") + "/" + kind + "/" + revision + "/" + declaration.Path, Fragment: fragment}
	return strings.NewReplacer("(", "%28", ")", "%29").Replace(link.String())
}

func fullRevision(revision string) bool {
	if len(revision) != 40 && len(revision) != 64 {
		return false
	}
	_, err := hex.DecodeString(revision)
	return err == nil
}
