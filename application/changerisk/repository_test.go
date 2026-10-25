package changerisk

import "testing"

func TestGitHubRepositoryAcceptsSafeRemotes(t *testing.T) {
	for _, remote := range []string{"https://github.com/owner/repo.git", "git@github.com:owner/repo.git", "ssh://git@github.com/owner/repo.git", "https://github.com/owner/repo"} {
		if got := githubRepository(remote); got != "https://github.com/owner/repo" {
			t.Errorf("%q: %q", remote, got)
		}
	}
	for _, remote := range []string{"https://token@github.com/owner/repo.git", "https://github.com/owner/repo?token=secret", "http://github.com/owner/repo", "https://example.com/owner/repo", "https://github.com/owner/..", "https://github.com/owner/repo\n<script>"} {
		if got := githubRepository(remote); got != "" {
			t.Errorf("unsafe remote %q: %q", remote, got)
		}
	}
}
