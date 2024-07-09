package artefacts

import (
	"io"
	"io/fs"
	"time"
)

type Environment map[string]fs.File

func EnvironmentFromReaders(readers map[string]io.Reader) Environment {
	e := make(Environment, len(readers))

	for n, r := range readers {
		var rc io.ReadCloser

		if c, ok := r.(io.ReadCloser); ok {
			rc = c
		} else {
			rc = io.NopCloser(r)
		}

		e[n] = &environmentFile{
			name:       n,
			size:       -1,
			mode:       fs.ModePerm,
			mtime:      time.Now(),
			ReadCloser: rc,
		}
	}

	return e
}

func (e Environment) Close() error {
	for _, f := range e {
		f.Close()
	}

	return nil
}

type environmentFile struct {
	name  string
	size  int64
	mode  fs.FileMode
	mtime time.Time
	io.ReadCloser
}

func (e *environmentFile) Stat() (fs.FileInfo, error) {
	return e, nil
}

func (e *environmentFile) Name() string {
	return e.name
}

func (e *environmentFile) Size() int64 {
	return e.size
}

func (e *environmentFile) Mode() fs.FileMode {
	return e.mode
}

func (e *environmentFile) ModTime() time.Time {
	return e.mtime
}

func (e *environmentFile) IsDir() bool {
	return false
}

func (e *environmentFile) Sys() any {
	return e
}
