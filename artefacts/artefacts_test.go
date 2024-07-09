package artefacts

import (
	_ "embed"
	"io"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func setupRemoteGit(t *testing.T) string {
	t.Helper()

	dir := createGitRepo(t)

	cmd := exec.Command("git", "--exec-path")
	ep, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("error finding git http backend: %s", err)
	}

	return httptest.NewServer(&cgi.Handler{
		Path:   filepath.Join(strings.TrimSpace(string(ep)), "git-http-backend"),
		Env:    []string{"GIT_HTTP_EXPORT_ALL=true", "GIT_PROJECT_ROOT=" + dir, "REMOTE_USER=test"},
		Stderr: io.Discard,
	}).URL
}

func execGit(t *testing.T, dir string, command ...string) {
	t.Helper()

	cmd := exec.Command("git", append(append(make([]string, 0, len(command)), "-C", dir), command...)...)
	cmd.Dir = dir

	if err := cmd.Run(); err != nil {
		t.Fatalf("error creating test repo (%v): %s", cmd.Args, err)
	}
}

var testFiles = map[string]string{
	"users/userA/env-1/a-file":   "1",
	"users/userA/env-1/b-file":   "contents",
	"users/userA/env-2/a-file":   "2",
	"users/userA/env-3/a-file":   "3",
	"users/userB/env-4/a-file":   "4",
	"users/userB/env-5/a-file":   "5",
	"users/userC/env-1/a-file":   "6",
	"groups/groupD/env-6/a-file": "7",
	"groups/groupE/env-1/a-file": "AAA",
	"groups/groupE/env-1/b-file": "BBB",
	"groups/groupE/env-1/c-file": "CCC",
}

func createGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	execGit(t, dir, "init")

	for name, contents := range testFiles {
		file := filepath.Join(dir, name)

		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatalf("error creating repo dir: %s", err)
		} else if f, err := os.Create(file); err != nil {
			t.Fatalf("error creating repo file: %s", err)
		} else if _, err = io.WriteString(f, contents); err != nil {
			t.Fatalf("error writing to repo file: %s", err)
		} else if err = f.Close(); err != nil {
			t.Fatalf("error closing repo file: %s", err)
		}

		execGit(t, dir, "add", file)
		execGit(t, dir, "commit", "-m", "Added "+name)
	}

	execGit(t, dir, "config", "--bool", "core.bare", "true")

	return filepath.Join(dir, ".git")
}

func TestList(t *testing.T) {
	url := setupRemoteGit(t)

	r, err := New(Remote(url))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	for n, test := range [...]struct{ Path, Expectation []string }{
		{Path: []string{userDirectory}, Expectation: []string{"userA", "userB", "userC"}},
		{Path: []string{groupDirectory}, Expectation: []string{"groupD", "groupE"}},
		{Path: []string{userDirectory, "userA"}, Expectation: []string{"env-1", "env-2", "env-3"}},
		{Path: []string{userDirectory, "userB"}, Expectation: []string{"env-4", "env-5"}},
		{Path: []string{userDirectory, "userC"}, Expectation: []string{"env-1"}},
		{Path: []string{groupDirectory, "groupD"}, Expectation: []string{"env-6"}},
		{Path: []string{groupDirectory, "groupE"}, Expectation: []string{"env-1"}},
	} {
		out, err := r.List(test.Path...)
		if err != nil {
			t.Fatalf("test %d: unexpected error: %s", n+1, err)
		} else if !slices.Equal(out, test.Expectation) {
			t.Errorf("test %d: expecting result %v, got %v", n+1, test.Expectation, out)
		}
	}
}
