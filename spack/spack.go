package spack

import (
	"encoding/json"
	"os/exec"
)

type Spack struct {
	exe  string
	args []string
}

func New() (*Spack, error) {
	cmd, err := exec.LookPath("spack")
	if err != nil {
		return nil, err
	}

	return NewWithPath(cmd), nil
}

func NewWithPath(cmd string, args ...string) *Spack {
	return &Spack{
		exe:  cmd,
		args: append(args, "list"),
	}
}

type Recipe struct {
	Name          string   `json:"name"`
	LatestVersion string   `json:"latest_version"`
	Version       []string `json:"versions"`
}

func (s *Spack) ListRecipes() ([]Recipe, error) {
	cmd := exec.Command(s.exe, s.args...)

	pr, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err = cmd.Start(); err != nil {
		return nil, err
	}

	var recipes []Recipe

	if err = json.NewDecoder(pr).Decode(&recipes); err != nil {
		return nil, err
	}

	if err = cmd.Wait(); err != nil {
		return nil, err
	}

	return recipes, nil
}
