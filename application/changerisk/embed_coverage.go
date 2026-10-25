package changerisk

import (
	"context"
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

func embeddedDocumentationScope(ctx context.Context, root string, changed []string) string {
	message, err := embeddedDocumentationCoverage(ctx, root, changed)
	if err != nil {
		return "Go embed coverage is unavailable; documentation changes need review: " + err.Error()
	}
	return message
}

func embeddedDocumentationCoverage(ctx context.Context, root string, changed []string) (string, error) {
	documents, err := changedMarkdownPaths(changed)
	if err != nil {
		return "", err
	}
	if len(documents) == 0 {
		return "", nil
	}
	data, err := command(ctx, root, "git", "ls-files", "-z", "--", "*.go")
	if err != nil {
		return "", err
	}
	var affected []string
	for _, name := range fields(data) {
		matches, err := sourceEmbeddedDocuments(root, name, documents)
		if err != nil {
			return "", err
		}
		affected = append(affected, matches...)
	}
	return embeddedDocumentationMessage(affected), nil
}

func changedMarkdownPaths(changed []string) ([]string, error) {
	var documents []string
	for _, name := range changed {
		if path.Ext(name) != ".md" {
			continue
		}
		if !validConfigPath(name) {
			return nil, fmt.Errorf("documentation path is outside the repository scope: %q", name)
		}
		documents = append(documents, name)
	}
	return documents, nil
}

func embeddedDocumentationMessage(affected []string) string {
	if len(affected) == 0 {
		return ""
	}
	slices.Sort(affected)
	return "changed embedded resources need review across Go build configurations: " + strings.Join(slices.Compact(affected), ", ")
}

func sourceEmbeddedDocuments(root, name string, documents []string) ([]string, error) {
	source, err := readEmbedSource(root, name)
	if err != nil {
		return nil, err
	}
	patterns, err := scanEmbedPatterns(name, source)
	if err != nil {
		return nil, err
	}
	return matchingEmbeddedDocuments(name, patterns, documents)
}

func readEmbedSource(root, name string) ([]byte, error) {
	if !validConfigPath(name) || path.Ext(name) != ".go" {
		return nil, fmt.Errorf("invalid Go embed inventory path: %q", name)
	}
	current, err := embedSourcePath(root, name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(current)
}

func embedSourcePath(root, name string) (string, error) {
	current := root
	for _, component := range strings.Split(name, "/") {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return "", err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return "", fmt.Errorf("Go embed inventory has an unsupported file mode: %q", name)
		}
	}
	return current, nil
}

func scanEmbedPatterns(name string, source []byte) ([]string, error) {
	file := token.NewFileSet().AddFile(name, -1, len(source))
	var scan scanner.Scanner
	scan.Init(file, source, nil, scanner.ScanComments)
	var patterns []string
	for {
		_, kind, literal := scan.Scan()
		if kind == token.EOF {
			break
		}
		if kind != token.COMMENT {
			continue
		}
		items, err := embedCommentPatterns(literal)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		patterns = append(patterns, items...)
	}
	if scan.ErrorCount != 0 {
		return nil, fmt.Errorf("cannot scan Go embed comments in %q", name)
	}
	return patterns, nil
}

func embedCommentPatterns(comment string) ([]string, error) {
	arguments, found := strings.CutPrefix(comment, "//go:embed")
	if !found {
		return nil, nil
	}
	if arguments != "" && !unicode.IsSpace([]rune(arguments)[0]) {
		return nil, nil
	}
	return parseEmbedPatterns(arguments)
}

func parseEmbedPatterns(arguments string) ([]string, error) {
	var patterns []string
	for arguments = strings.TrimSpace(arguments); arguments != ""; arguments = strings.TrimSpace(arguments) {
		pattern, rest, err := nextEmbedPattern(arguments)
		if err != nil {
			return nil, err
		}
		pattern = strings.TrimPrefix(pattern, "all:")
		if err := validateEmbedPattern(pattern); err != nil {
			return nil, err
		}
		patterns = append(patterns, pattern)
		arguments = rest
	}
	if len(patterns) == 0 {
		return nil, fmt.Errorf("Go embed directive has no patterns")
	}
	return patterns, nil
}

func nextEmbedPattern(arguments string) (string, string, error) {
	if arguments[0] == '"' || arguments[0] == '`' {
		return quotedEmbedPattern(arguments)
	}
	end := strings.IndexFunc(arguments, unicode.IsSpace)
	if end < 0 {
		end = len(arguments)
	}
	return arguments[:end], arguments[end:], nil
}

func quotedEmbedPattern(arguments string) (string, string, error) {
	quoted, err := strconv.QuotedPrefix(arguments)
	if err != nil {
		return "", "", err
	}
	pattern, err := strconv.Unquote(quoted)
	if err != nil {
		return "", "", err
	}
	rest := arguments[len(quoted):]
	if rest != "" && !unicode.IsSpace([]rune(rest)[0]) {
		return "", "", fmt.Errorf("Go embed patterns must be separated by whitespace")
	}
	return pattern, rest, nil
}

func validateEmbedPattern(pattern string) error {
	if !validConfigPath(pattern) {
		return fmt.Errorf("unsupported Go embed pattern: %q", pattern)
	}
	_, err := path.Match(pattern, "")
	return err
}

func matchingEmbeddedDocuments(source string, patterns, documents []string) ([]string, error) {
	prefix := path.Dir(source) + "/"
	if prefix == "./" {
		prefix = ""
	}
	var matches []string
	for _, name := range documents {
		relative, within := strings.CutPrefix(name, prefix)
		if !within {
			continue
		}
		matched, err := embedPatternsMatch(patterns, relative)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, name)
		}
	}
	return matches, nil
}

func embedPatternsMatch(patterns []string, name string) (bool, error) {
	for candidate := name; candidate != "."; candidate = path.Dir(candidate) {
		for _, pattern := range patterns {
			matched, err := path.Match(pattern, candidate)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
	}
	return false, nil
}
