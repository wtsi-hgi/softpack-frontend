package artefacts

import (
	"sync"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

const (
	userDirectory  = "users"
	groupDirectory = "groups"
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

	if o.storage == nil {
		o.storage = memory.NewStorage()
	}

	r, err := git.Clone(o.storage, memfs.New(), &o.CloneOptions)
	if err != nil {
		return nil, err
	}

	head, err := r.Head()
	if err != nil {
		return nil, err
	}

	return &Artefacts{
		repo: r,
		head: head,
	}, nil
}

func (a *Artefacts) getTree(path ...string) (*object.Tree, error) {
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
