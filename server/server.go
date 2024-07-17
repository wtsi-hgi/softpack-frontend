package server

import (
	"embed"
	"net/http"
	"os"

	"github.com/wtsi-hgi/softpack-frontend/environments"
	"github.com/wtsi-hgi/softpack-frontend/spack"
	"vimagination.zapto.org/tsserver"
)

const (
	environmentsPath = "environments"
	recipesPath      = "recipes"
)

//go:embed static
var static embed.FS

func New(s *spack.Spack, e *environments.Environments) http.Handler {
	sm := mux(s, e)

	sm.Handle("/", http.FileServer(http.FS(static)))

	return sm
}

func mux(s *spack.Spack, e *environments.Environments) *http.ServeMux {
	var sm http.ServeMux

	sm.Handle(environmentsPath, e)
	sm.Handle(recipesPath, s)

	return &sm
}

func NewDev(s *spack.Spack, e *environments.Environments, path string) http.Handler {
	sm := mux(s, e)

	sm.Handle("/", http.FileServer(http.FS(tsserver.WrapFS(os.DirFS(path)))))

	return sm
}
