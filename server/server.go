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
	environmentsPath = "/environments"
	recipesPath      = "/recipes"
	ldapPath         = "/ldap"
)

//go:embed static
var static embed.FS

func New(s *spack.Spack, e *environments.Environments, l http.Handler) http.Handler {
	sm := mux(s, e, l)

	sm.Handle("/", http.FileServer(http.FS(static)))

	return sm
}

func mux(s *spack.Spack, e *environments.Environments, l http.Handler) *http.ServeMux {
	sm := new(http.ServeMux)

	sm.Handle(environmentsPath, http.StripPrefix(environmentsPath, e))
	sm.Handle(recipesPath, http.StripPrefix(recipesPath, s))
	sm.Handle(ldapPath, http.StripPrefix(ldapPath, l))

	return sm
}

func NewDev(s *spack.Spack, e *environments.Environments, l http.Handler, path string) http.Handler {
	sm := mux(s, e, l)

	sm.Handle("/", http.FileServer(http.FS(tsserver.WrapFS(os.DirFS(path)))))

	return sm
}
