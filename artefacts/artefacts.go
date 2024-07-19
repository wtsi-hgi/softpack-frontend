package artefacts

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"path"
	"path/filepath"
	"sync"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
)

const (
	UserDirectory  = "users"
	GroupDirectory = "groups"
)

type Artefacts struct {
	mu   sync.RWMutex
	repo *git.Repository
	head *plumbing.Reference
}

func New(opts ...Option) (*Artefacts, error) {
	var o cloneOptions

	for _, opt := range opts {
		opt(&o)
	}

	var head *plumbing.Reference

	if o.storage == nil {
		o.storage = memory.NewStorage()
	}

	r, err := git.Clone(o.storage, memfs.New(), &o.CloneOptions)
	if errors.Is(err, git.ErrRepositoryAlreadyExists) {
		slog.Debug("opening cached artefact repo")

		r, err = git.Open(o.storage, memfs.New())
		if err != nil {
			return nil, err
		}

		head, err = r.Head()
		if err != nil {
			return nil, err
		}

		w, err := r.Worktree()
		if err != nil {
			return nil, err
		}

		slog.Debug("updating artefact repo")

		if err = w.Pull(&git.PullOptions{
			Force: true,
		}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return nil, err
		}
	} else if !errors.Is(err, transport.ErrEmptyRemoteRepository) {
		if err != nil {
			return nil, err
		}

		head, err = r.Head()
		if err != nil {
			return nil, err
		}
	}

	return &Artefacts{
		repo: r,
		head: head,
	}, nil
}

func (a *Artefacts) getTree(path ...string) (*object.Tree, error) {
	if a.head == nil {
		return &object.Tree{}, nil
	}

	c, err := a.repo.CommitObject(a.head.Hash())
	if err != nil {
		return nil, err
	}

	tree, err := c.Tree()
	if err != nil {
		return nil, err
	}

	for _, p := range path {
		tree, err = tree.Tree(p)
		if err != nil {
			return nil, err
		}
	}

	return tree, nil
}

func entriesToNames(entries []object.TreeEntry) ([]string, error) {
	names := make([]string, len(entries))

	for n, entry := range entries {
		names[n] = entry.Name
	}

	return names, nil
}

func (a *Artefacts) List(parts ...string) ([]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	f, err := a.getTree(parts...)
	if err != nil {
		return nil, err
	}

	return entriesToNames(f.Entries)
}

type Package struct {
	Name     string
	Versions []string
}

func (a *Artefacts) GetEnv(usersOrGroups, userOrGroup, env string) (Environment, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	f, err := a.getTree(usersOrGroups, userOrGroup, env)
	if err != nil {
		return nil, err
	}

	return a.entriesToEnvironment(path.Join(usersOrGroups, userOrGroup, env), f.Entries)
}

func (a *Artefacts) entriesToEnvironment(base string, entries []object.TreeEntry) (Environment, error) {
	e := make(Environment, len(entries))

	for _, entry := range entries {
		f, err := a.createFileFromEntry(base, entry)
		if err != nil {
			return nil, err
		}

		e[entry.Name] = f
	}

	return e, nil
}

func (a *Artefacts) createFileFromEntry(base string, entry object.TreeEntry) (fs.File, error) {
	filename := path.Join(base, entry.Name)

	c, err := a.getLatestCommitFromPath(filename)
	if err != nil {
		return nil, err
	}

	f, err := c.File(filename)
	if err != nil {
		return nil, err
	}

	r, err := f.Reader()
	if err != nil {
		return nil, err
	}

	return &environmentFile{
		name:       entry.Name,
		mtime:      c.Author.When,
		size:       f.Size,
		ReadCloser: r,
	}, nil
}

func (a *Artefacts) getLatestCommitFromPath(path string) (*object.Commit, error) {
	log, err := a.repo.Log(&git.LogOptions{
		From:     a.head.Hash(),
		Order:    git.LogOrderCommitterTime,
		FileName: &path,
	})
	if err != nil {
		return nil, err
	}

	defer log.Close()

	return log.Next()
}

func (a *Artefacts) AddFilesToEnv(usersOrGroups, userOrGroup, env string, files map[string]io.Reader) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	w, err := a.repo.Worktree()
	if err != nil {
		return err
	}

	for name, file := range files {
		if err = addFileToWorktree(w, filepath.Join(usersOrGroups, userOrGroup, env, name), file); err != nil {
			return err
		}
	}

	if _, err = w.Commit("Successfully written artefact(s)", &git.CommitOptions{All: true}); err != nil {
		return err
	}

	if a.head, err = a.repo.Head(); err != nil {
		return err
	}

	return a.repo.Push(&git.PushOptions{
		Force: true,
	})
}

func addFileToWorktree(w *git.Worktree, path string, file io.Reader) error {
	f, err := w.Filesystem.Create(path)
	if err != nil {
		return err
	}

	if _, err = io.Copy(f, file); err != nil {
		return err
	}

	if c, ok := file.(io.Closer); ok {
		c.Close()
	}

	if _, err = w.Add(path); err != nil {
		return err
	}

	return nil
}

func (a *Artefacts) RemoveEnvironment(usersOrGroups, userOrGroup, env string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	w, err := a.repo.Worktree()
	if err != nil {
		return err
	}

	if err = w.RemoveGlob(filepath.Join(usersOrGroups, userOrGroup, env, "*")); err != nil {
		return err
	}

	if _, err = w.Commit("Removed environment", &git.CommitOptions{All: true}); err != nil {
		return err
	}

	if a.head, err = a.repo.Head(); err != nil {
		return err
	}

	return a.repo.Push(&git.PushOptions{
		Force: true,
	})
}
