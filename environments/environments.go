package environments

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/wtsi-hgi/softpack-frontend/artefacts"
	"golang.org/x/net/websocket"
	"gopkg.in/yaml.v3"
)

const (
	environmentsFile        = "softpack.yml"
	builderOut              = "builder.out"
	moduleFile              = "module"
	readmeFile              = "README.md"
	metaFile                = "meta.yml"
	builtBySoftpackFile     = ".built_by_softpack"
	generatedFromModuleFile = ".generated_from_module"
)

type envStatus byte

const (
	envBuilding envStatus = iota
	envFailed
	envReady
)

type descriptionPackages struct {
	Description string
	Packages    []string
}

type meta struct {
	Tags []string
}

type environment struct {
	Tags        []string
	Packages    []string
	Description string
	ReadMe      string
	Status      envStatus
	SoftPack    bool
}

func environmentFromArtefacts(a artefacts.Environment, p string) (*environment, error) {
	e := &environment{}
	_, e.SoftPack = a[builtBySoftpackFile]

	if _, hasModule := a[moduleFile]; hasModule {
		if err := parseEnvironment(a, e); err != nil {
			return nil, err
		}

		e.Status = envReady
	} else if _, hasbuilderOut := a[builderOut]; hasbuilderOut {
		e.Status = envFailed
	}

	return e, nil
}

func parseEnvironment(a artefacts.Environment, e *environment) error {
	ef := a[environmentsFile]
	m := a[metaFile]
	readme := a[readmeFile]

	if ef == nil || readme == nil {
		return ErrBadEnvironment
	}

	var (
		dp       descriptionPackages
		sb       strings.Builder
		metadata meta
	)

	if err := yaml.NewDecoder(ef).Decode(&dp); err != nil {
		return err
	}

	if m != nil {
		if err := yaml.NewDecoder(m).Decode(&metadata); err != nil {
			return err
		}

		e.Tags = metadata.Tags
	}

	if _, err := io.Copy(&sb, readme); err != nil {
		return err
	}

	e.Description = dp.Description
	e.Packages = dp.Packages
	e.ReadMe = sb.String()

	return nil
}

type environments map[string]*environment

func (e environments) LoadFrom(a *artefacts.Artefacts, base string) error {
	entries, err := a.List(base)
	if errors.Is(err, object.ErrDirectoryNotFound) {
		return nil
	} else if err != nil {
		return err
	}

	for _, entry := range entries {
		envs, err := a.List(base, entry)
		if err != nil {
			return err
		}

		for _, env := range envs {
			as, err := a.GetEnv(base, entry, env)
			if err != nil {
				return err
			}

			p := path.Join(base, entry, env)

			e[p], err = environmentFromArtefacts(as, p)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type Environments struct {
	artefacts *artefacts.Artefacts
	socket
	http.Handler

	mu           sync.RWMutex
	environments map[string]*environment
	json         json.RawMessage
}

func New(a *artefacts.Artefacts) (*Environments, error) {
	envs := make(environments)

	if err := envs.LoadFrom(a, artefacts.UserDirectory); err != nil {
		return nil, err
	}

	if err := envs.LoadFrom(a, artefacts.GroupDirectory); err != nil {
		return nil, err
	}

	e := &Environments{
		artefacts:    a,
		environments: envs,
	}

	e.socket.Environments = e
	e.Handler = websocket.Handler(e.socket.ServeConn)

	e.updateJSON()

	return e, nil
}

func (e *Environments) updateJSON() {
	e.mu.Lock()
	defer e.mu.Unlock()

	var buf bytes.Buffer

	json.NewEncoder(&buf).Encode(e)

	e.json = json.RawMessage(buf.Bytes())
}

var ErrBadEnvironment = errors.New("bad environment")
