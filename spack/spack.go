package spack

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/wtsi-hgi/softpack-frontend/compressed"
	"vimagination.zapto.org/parser"
	"vimagination.zapto.org/python"
)

var spackRepo = "https://github.com/spack/spack.git" //nolint:gochecknoglobals

const (
	spackPackages  = "var/spack/repos/builtin/packages"
	customPackages = "packages"
)

type Spack struct {
	builtIn  map[string]recipe
	cacheDir string
	*compressed.File
}

func New(spackVersion plumbing.ReferenceName, opts ...Option) (*Spack, error) {
	var o options

	for _, opt := range opts {
		opt(&o)
	}

	var builtinRecipes map[string]recipe

	if o.cacheDir != "" {
		builtinRecipes = loadBuiltinFromCache(spackVersion, o.cacheDir)
	}

	if len(builtinRecipes) == 0 {
		var err error

		if builtinRecipes, err = loadBuiltinFromRepo(spackVersion, o.cacheDir); err != nil {
			return nil, err
		}

		slog.Debug("loaded builtin recipes from repo", "recipeCount", len(builtinRecipes))
	} else {
		slog.Debug("loaded builtin recipes from cache", "recipeCount", len(builtinRecipes))
	}

	s := &Spack{
		builtIn:  builtinRecipes,
		cacheDir: o.cacheDir,
		File:     compressed.New("recipes.json"),
	}

	if o.remote != "" {
		s.watchRemote(o.remote, o.remoteUpdateFrequency)
	}

	return s, nil
}

func loadBuiltinFromCache(spackVersion plumbing.ReferenceName, cacheDir string) map[string]recipe {
	builtinRecipes, _ := loadFromCache(cachePath(cacheDir, string(spackVersion)))

	return builtinRecipes
}

func loadFromCache(cacheFile string) (map[string]recipe, error) {
	f, err := os.Open(cacheFile)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	var recipes map[string]recipe

	if err = json.NewDecoder(f).Decode(&recipes); err != nil {
		return nil, err
	}

	return recipes, nil
}

func cachePath(cacheDir, identifier string) string {
	return filepath.Join(cacheDir, base64.RawURLEncoding.EncodeToString([]byte(identifier)))
}

func loadBuiltinFromRepo(spackVersion plumbing.ReferenceName, cacheDir string) (map[string]recipe, error) {
	builtinFS := memfs.New()

	if _, err := git.Clone(memory.NewStorage(), builtinFS, &git.CloneOptions{
		URL:           spackRepo,
		ReferenceName: spackVersion,
		SingleBranch:  true,
		Depth:         1,
	}); err != nil {
		return nil, err
	}

	builtinRecipes, err := readRecipes(builtinFS, spackPackages)
	if err != nil {
		return nil, err
	}

	if cacheDir != "" {
		slog.Debug("writing builting recipes to cache", "recipeCount", len(builtinRecipes))

		if err := writeToCache(cachePath(cacheDir, string(spackVersion)), builtinRecipes); err != nil {
			return nil, err
		}
	}

	return builtinRecipes, nil
}

func writeToCache(cacheFile string, recipes map[string]recipe) error {
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		return err
	}

	f, err := os.Create(cacheFile)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(f).Encode(recipes); err != nil {
		return err
	}

	return nil
}

func readRecipes(bfs billy.Filesystem, base string) (map[string]recipe, error) {
	recipePaths, _ := bfs.ReadDir(base) //nolint:errcheck

	recipes := make(map[string]recipe, len(recipePaths))

	for _, r := range recipePaths {
		name := r.Name()

		f, err := bfs.Open(path.Join(base, name, "package.py"))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, err
		}

		versions := parseRecipeVersions(f)

		f.Close()

		if versions != nil {
			recipes[name] = recipe{
				Name:     name,
				Versions: versions,
			}
		}
	}

	return recipes, nil
}

func parseRecipeVersions(r io.Reader) []string {
	tk := parser.NewReaderTokeniser(r)

	python.SetTokeniser(&tk)

	var versions []string

	p := parser.New(tk)

	for {
		if p.ExceptRun(python.TokenIdentifier) < 0 {
			break
		}

		if p.AcceptToken(parser.Token{Type: python.TokenIdentifier, Data: "version"}) {
			p.AcceptRun(python.TokenWhitespace)

			if p.AcceptToken(parser.Token{Type: python.TokenDelimiter, Data: "("}) {
				p.AcceptRun(python.TokenWhitespace, python.TokenLineTerminator)

				if p.Accept(python.TokenStringLiteral) {
					tokens := p.Get()

					ver, err := python.Unquote(tokens[len(tokens)-1].Data)
					if err == nil {
						versions = append(versions, ver)
					}
				}
			}
		} else {
			p.Except()
		}
	}

	return versions
}

func (s *Spack) watchRemote(url string, timeout time.Duration) error {
	if s.cacheDir != "" {
		if err := s.loadRemoteCache(url); err == nil {
			slog.Debug("loaded remote recipes from cache")

			go s.getRemote(url, timeout)

			return nil
		}
	}

	slog.Debug("loading remote recipes from repo")

	return s.getRemote(url, timeout)
}

func (s *Spack) getRemote(url string, timeout time.Duration) error {
	fs := memfs.New()

	r, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	if err := s.updateRemote(url, fs); err != nil {
		return err
	}

	if timeout > 0 {
		go func() {
			for {
				time.Sleep(timeout)
				if err := w.Pull(&git.PullOptions{
					Force: true,
				}); err != nil {
					slog.Debug("error pulling remote recipes", "err", err)
				} else if err = s.updateRemote(url, fs); err != nil {
					slog.Debug("error parsing remote recipes", "err", err)
				}
			}
		}()
	}

	return nil
}

func (s *Spack) loadRemoteCache(url string) error {
	recipes, err := loadFromCache(cachePath(s.cacheDir, url))
	if err != nil {
		return err
	}

	s.mergeRecipes(recipes)

	return nil
}

func (s *Spack) updateRemote(url string, fs billy.Filesystem) error {
	recipes, err := readRecipes(fs, customPackages)
	if err != nil {
		return err
	}

	if s.cacheDir != "" {
		s.cacheRemoteRecipes(url, recipes)
	}

	s.mergeRecipes(recipes)

	return nil
}

func (s *Spack) cacheRemoteRecipes(url string, recipes map[string]recipe) {
	slog.Debug("saving remote recipes to cache", "recipeCount", len(recipes))
	writeToCache(cachePath(s.cacheDir, url), recipes)
}

type recipe struct {
	Name     string
	Versions []string
}

func (s *Spack) mergeRecipes(recipes map[string]recipe) {
	recipeList := make([]recipe, 0, len(recipes)+len(s.builtIn))

	for name, recipe := range s.builtIn {
		if _, ok := recipes[name]; !ok {
			recipeList = append(recipeList, recipe)
		}
	}

	for _, recipe := range recipes {
		recipeList = append(recipeList, recipe)
	}

	sort.Slice(recipeList, func(i, j int) bool {
		return recipeList[i].Name < recipeList[j].Name
	})

	s.File.Encode(recipeList)
}
