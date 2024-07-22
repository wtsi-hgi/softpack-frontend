package artefacts

import (
	_ "embed"
	"errors"
	"io"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/wtsi-hgi/softpack-frontend/internal/git"
)

var testFiles = map[string]string{
	Environments + "/" + UserDirectory + "/userA/env-1/a-file":   "1",
	Environments + "/" + UserDirectory + "/userA/env-1/b-file":   "contents",
	Environments + "/" + UserDirectory + "/userA/env-2/a-file":   "2",
	Environments + "/" + UserDirectory + "/userA/env-3/a-file":   "3",
	Environments + "/" + UserDirectory + "/userB/env-4/a-file":   "4",
	Environments + "/" + UserDirectory + "/userB/env-5/a-file":   "5",
	Environments + "/" + UserDirectory + "/userC/env-1/a-file":   "6",
	Environments + "/" + GroupDirectory + "/groupD/env-6/a-file": "7",
	Environments + "/" + GroupDirectory + "/groupE/env-1/a-file": "AAA",
	Environments + "/" + GroupDirectory + "/groupE/env-1/b-file": "BBB",
	Environments + "/" + GroupDirectory + "/groupE/env-1/c-file": "CCC",
}

func TestList(t *testing.T) {
	g := git.New(t)
	g.Add(t, testFiles)

	r, err := New(Remote(g.URL()))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	for n, test := range [...]struct{ Path, Expectation []string }{
		{Path: []string{UserDirectory}, Expectation: []string{"userA", "userB", "userC"}},
		{Path: []string{GroupDirectory}, Expectation: []string{"groupD", "groupE"}},
		{Path: []string{UserDirectory, "userA"}, Expectation: []string{"env-1", "env-2", "env-3"}},
		{Path: []string{UserDirectory, "userB"}, Expectation: []string{"env-4", "env-5"}},
		{Path: []string{UserDirectory, "userC"}, Expectation: []string{"env-1"}},
		{Path: []string{GroupDirectory, "groupD"}, Expectation: []string{"env-6"}},
		{Path: []string{GroupDirectory, "groupE"}, Expectation: []string{"env-1"}},
	} {
		out, err := r.List(test.Path...)
		if err != nil {
			t.Fatalf("test %d: unexpected error: %s", n+1, err)
		} else if !slices.Equal(out, test.Expectation) {
			t.Errorf("test %d: expecting result %v, got %v", n+1, test.Expectation, out)
		}
	}
}

func TestGetEnv(t *testing.T) {
	g := git.New(t)
	g.Add(t, testFiles)

	r, err := New(Remote(g.URL()))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	for n, test := range [...]struct {
		Path        [3]string
		Expectation map[string]string
	}{
		{Path: [3]string{UserDirectory, "userA", "env-1"}, Expectation: map[string]string{"a-file": "1", "b-file": "contents"}},
	} {
		env, err := r.GetEnv(test.Path[0], test.Path[1], test.Path[2])
		if err != nil {
			t.Fatalf("test %d: unexpected error: %s", n+1, err)
		}

		for name, file := range env {
			contents, err := io.ReadAll(file)
			if err != nil {
				t.Fatalf("test %d: unexpected error reading file (%s): %s", n+1, name, err)
			} else if expectation := test.Expectation[name]; string(contents) != expectation {
				t.Errorf("test %d: expecting to read %q from %s, got %q", n+1, expectation, name, contents)
			}
		}
	}
}

func TestAddFilesToEnv(t *testing.T) {
	g := git.New(t)
	g.Add(t, testFiles)

	r, err := New(Remote(g.URL()))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	const (
		newFileName     = "newFile"
		newFileContents = "BRAND NEW"
	)

	if err = r.AddFilesToEnv(UserDirectory, "userC", "env-1", map[string]io.Reader{
		newFileName: strings.NewReader(newFileContents),
	}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err = checkFile(t, r, UserDirectory, "userC", "env-1", newFileName, newFileContents); err != nil {
		t.Fatal(err)
	}

	r, err = New(Remote(g.URL()))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err = checkFile(t, r, UserDirectory, "userC", "env-1", newFileName, newFileContents); err != nil {
		t.Fatal(err)
	}
}

func checkFile(t *testing.T, r *Artefacts, usersOrGroups, userOrGroup, envP, filename, contents string) error {
	t.Helper()

	env, err := r.GetEnv(usersOrGroups, userOrGroup, envP)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	file, ok := env[filename]
	if !ok {
		return errors.New("failed to read new file")
	}

	c, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if str := string(c); str != contents {
		t.Fatalf("expected to read contents %q, got %q", contents, str)
	}

	return nil
}

func TestRemoveEnvironment(t *testing.T) {
	g := git.New(t)
	g.Add(t, testFiles)

	r, err := New(Remote(g.URL()))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err = r.RemoveEnvironment(UserDirectory, "userA", "env-1"); err != nil {
		t.Fatalf("unexpected error while removing environment: %s", err)
	}

	if _, err = r.GetEnv(UserDirectory, "userA", "env-1"); !errors.Is(err, object.ErrDirectoryNotFound) {
		t.Errorf("expecting error %q, got %q", object.ErrDirectoryNotFound, err)
	}
}

func TestCache(t *testing.T) {
	g := git.New(t)
	g.Add(t, testFiles)

	cacheDir := t.TempDir()

	var messages []string

	debug = func(msg string, args ...any) {
		messages = append(messages, msg)
	}

	defer func() { debug = slog.Debug }()

	_, err := New(Remote(g.URL()), FS(cacheDir))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	} else if len(messages) != 0 {
		t.Fatalf("read unexpected debug messages: %v", messages)
	}

	_, err = New(Remote(g.URL()), FS(cacheDir))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	} else if len(messages) != 2 {
		t.Fatalf("expected to read 2 debug messages, got: %v", messages)
	}
}
