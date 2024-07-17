package git

import (
	"io"
	"net/http/cgi" //nolint:gosec
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type Remote struct {
	path, url string
}

func New(t *testing.T) *Remote {
	t.Helper()

	dir := t.TempDir()

	execGit(t, dir, "init")
	execGit(t, dir, "config", "--bool", "core.bare", "true")

	cmd := exec.Command("git", "--exec-path")

	ep, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("error finding git http backend: %s", err)
	}

	return &Remote{
		path: dir,
		url: httptest.NewServer(&cgi.Handler{
			Path:   filepath.Join(strings.TrimSpace(string(ep)), "git-http-backend"),
			Env:    []string{"GIT_HTTP_EXPORT_ALL=true", "GIT_PROJECT_ROOT=" + dir, "REMOTE_USER=test"},
			Stderr: io.Discard,
		}).URL,
	}
}

func execGit(t *testing.T, dir string, command ...string) {
	t.Helper()

	cmd := exec.Command("git", append(append(make([]string, 0, len(command)), "-C", dir), command...)...)
	cmd.Dir = dir

	if err := cmd.Run(); err != nil {
		t.Fatalf("error creating test repo (%v): %s", cmd.Args, err)
	}
}

func (r *Remote) Add(t *testing.T, files map[string]string) {
	t.Helper()

	execGit(t, r.path, "config", "--bool", "core.bare", "false")
	defer execGit(t, r.path, "config", "--bool", "core.bare", "true")

	for name, contents := range files {
		file := filepath.Join(r.path, name)

		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatalf("error creating repo dir: %s", err)
		} else if f, err := os.Create(file); err != nil {
			t.Fatalf("error creating repo file: %s", err)
		} else if _, err = io.WriteString(f, contents); err != nil {
			t.Fatalf("error writing to repo file: %s", err)
		} else if err = f.Close(); err != nil {
			t.Fatalf("error closing repo file: %s", err)
		}

		execGit(t, r.path, "add", file)
		execGit(t, r.path, "commit", "-m", "Added "+name)
	}

	execGit(t, r.path, "update-server-info")
}

func (r *Remote) URL() string {
	return r.url
}
