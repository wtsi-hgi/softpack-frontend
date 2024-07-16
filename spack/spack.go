package spack

import (
	"io"
	"path"
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

var (
	spackRepo      = "https://github.com/spack/spack.git"
	spackPackages  = "var/spack/repos/builtin/packages"
	customPackages = "packages"
)

type Spack struct {
	builtIn map[string]recipe
	*compressed.File
}

func New(spackVersion plumbing.ReferenceName) (*Spack, error) {
	builtinFS := memfs.New()
	_, err := git.Clone(memory.NewStorage(), builtinFS, &git.CloneOptions{
		URL:           spackRepo,
		ReferenceName: spackVersion,
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		return nil, err
	}

	builtinRecipes, err := readRecipes(builtinFS, spackPackages)
	if err != nil {
		return nil, err
	}

	return &Spack{
		builtIn: builtinRecipes,
		File:    compressed.New("recipes.json"),
	}, nil
}

func readRecipes(fs billy.Filesystem, base string) (map[string]recipe, error) {
	recipePaths, _ := fs.ReadDir(base)

	recipes := make(map[string]recipe, len(recipePaths))

	for _, r := range recipePaths {
		name := r.Name()

		f, err := fs.Open(path.Join(base, name, "package.py"))
		if err != nil {
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
					ver, err := python.Unescape(tokens[len(tokens)-1].Data)

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

func (s *Spack) WatchRemote(url string, timeout time.Duration) error {
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

	s.updateRemote(fs)

	if timeout > 0 {
		go func() {
			for {
				time.Sleep(timeout)
				w.Pull(&git.PullOptions{
					Force: true,
				})
				s.updateRemote(fs)
			}
		}()
	}

	return nil
}

func (s *Spack) updateRemote(fs billy.Filesystem) {
	recipes, err := readRecipes(fs, customPackages)
	if err == nil {
		s.mergeRecipes(recipes)
	}
}

type recipe struct {
	Name     string
	Versions []string
}

func (s *Spack) mergeRecipes(recipes map[string]recipe) {
	var recipeList []recipe

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
