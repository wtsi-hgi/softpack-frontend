package spack

import (
	_ "embed"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	//go:embed spack
	spackStr string

	SpackExe  string
	SpackArgs []string
)

func Setup() (func(), error) {
	SpackExe = os.Getenv("SPACK_EXE")

	json.Unmarshal([]byte(os.Getenv("SPACK_ARGS")), &SpackArgs)

	if SpackExe == "" {
		SpackExe, _ = exec.LookPath("spack")
	}

	return buildFakeSpack()
}

func buildFakeSpack() (func(), error) {
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, err
	}

	spack := filepath.Join(tmp, "spack")

	f, err := os.Create(spack)
	if err != nil {
		return nil, err
	}

	if _, err = io.WriteString(f, spackStr); err != nil {
		return nil, err
	}

	if err = f.Close(); err != nil {
		return nil, err
	}

	os.Setenv("PATH", tmp+":"+os.Getenv("PATH"))

	if err = os.Chmod(spack, 0o755); err != nil {
		return nil, err
	}

	return func() {
		os.RemoveAll(tmp)
	}, nil
}
