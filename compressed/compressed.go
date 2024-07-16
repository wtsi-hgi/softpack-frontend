package compressed

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"hash"
	"io"
	"net/http"
	"sync"
	"time"

	"vimagination.zapto.org/httpencoding"
)

var isGzip = httpencoding.HandlerFunc(func(enc httpencoding.Encoding) bool { return enc == "gzip" })

type File struct {
	name string

	mu                       sync.RWMutex
	hash                     hash.Hash
	compressed, uncompressed []byte
	modTime                  time.Time
}

func New(name string) *File {
	return &File{
		name: name,
	}
}

func (f *File) ReadFrom(r io.Reader) (int64, error) {
	var compressed, uncompressed bytes.Buffer

	g := gzip.NewWriter(&compressed)

	n, err := uncompressed.ReadFrom(io.TeeReader(r, g))
	if err != nil {
		return n, err
	}

	g.Close()

	f.mu.Lock()
	defer f.mu.Unlock()

	if !bytes.Equal(uncompressed.Bytes(), f.uncompressed) {
		f.modTime = time.Now()
		f.compressed = compressed.Bytes()
		f.uncompressed = uncompressed.Bytes()
	}

	return n, nil
}

func (f *File) Encode(v any) {
	var compressed, uncompressed bytes.Buffer

	json.NewEncoder(&uncompressed).Encode(v)

	f.mu.Lock()
	defer f.mu.Unlock()

	if bytes.Equal(uncompressed.Bytes(), f.uncompressed) {
		return
	}

	g := gzip.NewWriter(&compressed)

	g.Write(uncompressed.Bytes())
	g.Close()

	f.modTime = time.Now()
	f.compressed = compressed.Bytes()
	f.uncompressed = uncompressed.Bytes()
}

func (f *File) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var br *bytes.Reader

	f.mu.RLock()

	if httpencoding.HandleEncoding(r, isGzip) {
		br = bytes.NewReader(f.compressed)

		w.Header().Add("Content-Encoding", "gzip")
	} else {
		br = bytes.NewReader(f.uncompressed)
	}

	f.mu.RUnlock()

	http.ServeContent(w, r, f.name, f.modTime, br)
}
