package compressed

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"vimagination.zapto.org/httpencoding"
)

var isGzip = httpencoding.HandlerFunc(func(enc httpencoding.Encoding) bool { return enc == "gzip" }) //nolint:gochecknoglobals

type File struct {
	name string

	mu                       sync.RWMutex
	compressed, uncompressed []byte
	modTime                  time.Time
}

func New(name string) *File {
	return &File{
		name: name,
	}
}

func (f *File) ReadFrom(r io.Reader) (int64, error) {
	var uncompressed bytes.Buffer

	n, err := uncompressed.ReadFrom(r)
	if err != nil {
		return n, err
	}

	f.writeData(uncompressed.Bytes())

	return n, nil
}

func (f *File) Encode(v any) {
	var uncompressed bytes.Buffer

	json.NewEncoder(&uncompressed).Encode(v) //nolint:errcheck

	f.writeData(uncompressed.Bytes())
}

func (f *File) writeData(p []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if bytes.Equal(p, f.uncompressed) {
		return
	}

	var compressed bytes.Buffer

	g := gzip.NewWriter(&compressed)

	g.Write(p) //nolint:errcheck
	g.Close()

	f.modTime = time.Now()
	f.compressed = compressed.Bytes()
	f.uncompressed = p
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
