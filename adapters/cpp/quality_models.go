package cpp

import (
	"slices"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/quality"
)

var ctypeNames = strings.Fields("isalnum isalpha isblank iscntrl isdigit isgraph islower isprint ispunct isspace isupper isxdigit tolower toupper")

func (p *projection) modeledCtype(node *vertex) string {
	name := standardCtypeName(node)
	for _, target := range p.doc.Callables[name+":int(int)"] {
		if !target.flag("IS_EXTERNAL") {
			return ""
		}
	}
	return name
}

func cppEffectPrefixes() []quality.Prefix {
	models := []struct {
		kind  quality.Kind
		names string
	}{
		{quality.IO, "printf fprintf sprintf snprintf puts putchar getchar fopen fclose fread fwrite fflush open close read write remove rename"},
		{quality.Network, "socket connect accept bind listen send recv sendto recvfrom"},
		{quality.Time, "time clock gettimeofday clock_gettime"},
		{quality.Random, "rand srand random srandom"},
		{quality.Unsafe, "system execve fork"},
		{quality.Panic, "abort exit _Exit terminate std.terminate std::terminate"},
	}
	var result []quality.Prefix
	for _, model := range models {
		for _, name := range strings.Fields(model.names) {
			result = append(result, quality.Prefix{Path: qualityName(name) + ":", Kind: model.kind, ForeignOnly: true})
		}
	}
	for _, name := range ctypeNames {
		for _, prefix := range []string{"std.", "std::"} {
			result = append(result, quality.Prefix{Path: qualityName(prefix+name) + ":int(int)", Kind: quality.GlobalRead, ForeignOnly: true})
		}
	}
	return result
}

func standardCtypeName(node *vertex) string {
	if importedCallForm(node) != pdg.DirectCall || argument(node, 1) == nil || argument(node, 2) != nil {
		return ""
	}
	if !standardCtypeSource(node) {
		return ""
	}
	return "std." + node.text("NAME")
}

func standardCtypeSource(node *vertex) bool {
	head, _, found := strings.Cut(node.text("CODE"), "(")
	return found && slices.Contains(ctypeNames, node.text("NAME")) &&
		strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(head), "::")) == "std::"+node.text("NAME")
}
