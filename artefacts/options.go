package artefacts

import (
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

type Option func(*cloneOptions)

type cloneOptions struct {
	git.CloneOptions
	fs      billy.Filesystem
	storage storage.Storer
}

func Remote(url string) Option {
	return func(o *cloneOptions) {
		o.URL = url
	}
}

func UserPass(username, password string) Option {
	return func(o *cloneOptions) {
		o.Auth = &http.BasicAuth{
			Username: username,
			Password: password,
		}
	}
}

func Token(token string) Option {
	return func(o *cloneOptions) {
		o.Auth = &http.BasicAuth{
			Password: token,
		}
	}
}

func FS(path string) Option {
	return func(o *cloneOptions) {
		o.fs = osfs.New(path)
		o.storage = filesystem.NewStorage(osfs.New(filepath.Join(path, ".git")), cache.NewObjectLRUDefault())
	}
}
