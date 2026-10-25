package changerisk

import (
	"context"
	"regexp"
	"strings"
)

var githubRemote = regexp.MustCompile(`^(?:https://github[.]com/|git@github[.]com:|ssh://git@github[.]com/)([A-Za-z0-9_-]+)/([A-Za-z0-9_.-]+)$`)

func repositoryURL(ctx context.Context, root string) string {
	data, err := command(ctx, root, "git", "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return githubRepository(strings.TrimSpace(string(data)))
}

func githubRepository(remote string) string {
	parts := githubRemote.FindStringSubmatch(strings.TrimSuffix(remote, ".git"))
	if len(parts) != 3 {
		return ""
	}
	if parts[2] == "." || parts[2] == ".." {
		return ""
	}
	return "https://github.com/" + parts[1] + "/" + parts[2]
}
