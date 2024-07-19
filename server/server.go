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
	environmentsPath = "/envs"
	recipesPath      = "/recipes"
	ldapPath         = "/ldap"
	aboutPage        = "/about"
	environmentsPage = "/environments"
	tagsPage         = "/tags"
	createPage       = "/create"
)

//go:embed static
var static embed.FS

func New(s *spack.Spack, e *environments.Environments, l http.Handler) http.Handler {
	return mux(s, e, l, http.FileServer(&virtualPages{http.FS(static)}))
}

func mux(s, e, l, files http.Handler) http.Handler {
	sm := new(http.ServeMux)

	sm.Handle(environmentsPath+"/", http.StripPrefix(environmentsPath, e))
	sm.Handle(recipesPath, http.StripPrefix(recipesPath, s))
	sm.Handle(ldapPath, http.StripPrefix(ldapPath, l))
	sm.Handle("/", files)

	return sm
}

func NewDev(s *spack.Spack, e *environments.Environments, l http.Handler, path string) http.Handler {
	return mux(s, e, l, http.FileServer(&virtualPages{http.FS(tsserver.WrapFS(os.DirFS(path)))}))
}

type virtualPages struct {
	http.FileSystem
}

func (v *virtualPages) Open(path string) (http.File, error) {
	switch path {
	case aboutPage, environmentsPage, tagsPage, createPage:
		path = "/index.html"
	}

	return v.FileSystem.Open(path)
}
