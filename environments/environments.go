package environments

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
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

	socketPath        = "/socket"
	uploadPath        = "/upload"
	resendPendingPath = "/resend-pending-builds"
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

func environmentFromArtefacts(a artefacts.Environment) (*environment, error) {
	e := &environment{
		Tags: make([]string, 0),
	}
	_, e.SoftPack = a[builtBySoftpackFile]
	ef := a[environmentsFile]

	if ef == nil {
		return nil, ErrBadEnvironment
	}

	if err := e.setSoftpackYaml(ef); err != nil {
		return nil, err
	}

	if _, hasModule := a[moduleFile]; hasModule {
		if err := parseReadyEnvironment(a, e); err != nil {
			return nil, err
		}

		e.Status = envReady
	} else if _, hasbuilderOut := a[builderOut]; hasbuilderOut {
		e.Status = envFailed
	}

	return e, nil
}

func parseReadyEnvironment(a artefacts.Environment, e *environment) error {
	meta := a[metaFile]
	readme := a[readmeFile]

	if readme == nil {
		return ErrBadEnvironment
	}

	if err := e.setReadme(readme); err != nil {
		return err
	}

	if meta != nil {
		if err := e.setMeta(meta); err != nil {
			return err
		}
	}

	return nil
}

func (e *environment) setSoftpackYaml(r io.Reader) error {
	var dp descriptionPackages

	if err := yaml.NewDecoder(r).Decode(&dp); err != nil {
		return err
	}

	e.Description = dp.Description
	e.Packages = dp.Packages

	return nil
}

func (e *environment) setReadme(r io.Reader) error {
	var sb strings.Builder

	if _, err := io.Copy(&sb, r); err != nil {
		return err
	}

	e.ReadMe = sb.String()

	return nil
}

func (e *environment) setMeta(r io.Reader) error {
	var metadata meta

	if err := yaml.NewDecoder(r).Decode(&metadata); err != nil {
		return err
	}

	e.Tags = metadata.Tags

	return nil
}

func (e *environment) setStatus(s envStatus) {
	e.Status = s
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

			ep, err := environmentFromArtefacts(as)
			if err != nil {
				slog.Error("failed to load environment", "env", path.Join(base, entry, env), "err", err)

				continue
			}

			e[p] = ep
		}
	}

	return nil
}

type Environments struct {
	artefacts *artefacts.Artefacts
	socket
	http.ServeMux

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
	e.socket.conns = make(map[*conn]struct{})

	e.ServeMux.Handle(socketPath, websocket.Handler(e.socket.ServeConn))
	e.ServeMux.HandleFunc(uploadPath, e.handleUpload)
	e.ServeMux.HandleFunc(resendPendingPath, e.handleResend)

	e.updateJSON()

	return e, nil
}

func (e *Environments) handleUpload(w http.ResponseWriter, r *http.Request) {}

func (e *Environments) handleResend(w http.ResponseWriter, r *http.Request) {}

func (e *Environments) updateJSON() {
	e.mu.Lock()
	defer e.mu.Unlock()

	var buf bytes.Buffer

	json.NewEncoder(&buf).Encode(e.environments)

	e.json = json.RawMessage(buf.Bytes())
}

var ErrBadEnvironment = errors.New("bad environment")
