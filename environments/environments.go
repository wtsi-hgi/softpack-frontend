package environments

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/wtsi-hgi/softpack-frontend/artefacts"
	"github.com/wtsi-hgi/softpack-frontend/compressed"
	"gopkg.in/yaml.v3"
)

const (
	environments_root          = "environments"
	environments_file          = "softpack.yml"
	builder_out                = "builder.out"
	module_file                = "module"
	readme_file                = "README.md"
	meta_file                  = "meta.yml"
	built_by_softpack_file     = ".built_by_softpack"
	generated_from_module_file = ".generated_from_module"
)

type envStatus byte

const (
	envBuilding envStatus = iota
	envFailed
	envReady
)

const zeros = "00000000000000000000"

var (
	digitsRegexp    = regexp.MustCompile(`\d+`)
	nondigitsRegexp = regexp.MustCompile(`[^\d]+`)
	idEncoding      = base64.NewEncoding("-0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz").WithPadding(base64.NoPadding)
)

type descriptionPackages struct {
	Description string
	Packages    []string
}

type meta struct {
	Tags []string
}

type environment struct {
	id          string
	Path        string
	NameVersion string
	Tags        []string
	Packages    []string
	Description string
	ReadMe      string
	Status      envStatus
	SoftPack    bool
}

func environmentFromArtefacts(a artefacts.Environment, p string) (*environment, error) {
	path, nameVer := splitLastOnce(p, '/')
	e := &environment{
		Path:        path,
		NameVersion: nameVer,
	}

	_, e.SoftPack = a[built_by_softpack_file]

	if _, hasModule := a[module_file]; hasModule {
		if err := parseEnvironment(a, e); err != nil {
			return nil, err
		}

		e.Status = envReady
	} else if _, hasbuilderOut := a[builder_out]; hasbuilderOut {
		e.Status = envFailed
	}

	e.id = idFromPath(p)

	return e, nil
}

func idFromPath(path string) string {
	base, env := splitLastOnce(path, '/')
	name, version := splitLastOnce(env, '-')
	numberParts := digitsRegexp.FindAllString(version, -1)
	stringParts := nondigitsRegexp.FindAllString(version, -1)

	var sb bytes.Buffer

	sb.WriteString(strings.ToLower(name))
	sb.WriteRune('-')

	for _, n := range numberParts {
		if len(n) < 20 {
			sb.WriteString(zeros[:len(zeros)-len(n)])
		}

		sb.WriteString(n)
	}

	for _, s := range stringParts {
		sb.WriteString(s)
	}

	sb.WriteRune(0)
	sb.WriteString(base)

	return idEncoding.EncodeToString(sb.Bytes())
}

func splitLastOnce(str string, char byte) (string, string) {
	i := strings.LastIndexByte(str, char)
	if i == -1 {
		return str, ""
	}

	return str[:i], str[i+1:]
}

func parseEnvironment(a artefacts.Environment, e *environment) error {
	ef := a[environments_file]
	m := a[meta_file]
	readme := a[readme_file]

	if ef == nil || readme == nil {
		return ErrBadEnvironment
	}

	var dp descriptionPackages

	if err := yaml.NewDecoder(ef).Decode(&dp); err != nil {
		return err
	}

	e.Description = dp.Description
	e.Packages = dp.Packages

	if m != nil {
		var meta meta

		if err := yaml.NewDecoder(m).Decode(&meta); err != nil {
			return err
		}

		e.Tags = meta.Tags
	}

	var sb strings.Builder

	if _, err := io.Copy(&sb, readme); err != nil {
		return err
	}

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
			ea, err := environmentFromArtefacts(as, p)
			if err != nil {
				return err
			}

			e[p] = ea
		}
	}

	return nil
}

type Environments struct {
	artefacts        *artefacts.Artefacts
	environmentsJSON *compressed.File

	mu           sync.RWMutex
	environments map[string]*environment
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
		artefacts:        a,
		environments:     envs,
		environmentsJSON: compressed.New("environments.json"),
	}

	e.updateJSON()

	return e, nil
}

func (e *Environments) updateJSON() {
	e.mu.RLock()
	defer e.mu.RUnlock()

	envs := make([]*environment, 0, len(e.environments))

	for _, env := range e.environments {
		envs = append(envs, env)
	}

	sort.Slice(envs, func(i, j int) bool {
		return envs[i].id < envs[j].id
	})

	e.environmentsJSON.Encode(envs)
}

func (e *Environments) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch path.Base(r.URL.Path) {
	case "environments.json":
		e.environmentsJSON.ServeHTTP(w, r)
	default:
		http.NotFound(w, r)
	}
}

var ErrBadEnvironment = errors.New("bad environment")
