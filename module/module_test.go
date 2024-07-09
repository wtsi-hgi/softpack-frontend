package module

import (
	"bytes"
	"embed"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed files/*
var files embed.FS

func TestToSoftPack(t *testing.T) {
	entries, _ := files.ReadDir("files")

	var n int

	for _, entry := range entries {
		if name := entry.Name(); strings.HasSuffix(entry.Name(), ".mod") {
			n++

			tName := strings.TrimSuffix(name, ".mod")

			contents, _ := files.ReadFile(filepath.Join("files", name))
			expected, _ := files.ReadFile(filepath.Join("files", tName+".yml"))

			if yml, _ := io.ReadAll(ToSoftpackYML(tName, string(contents))); !bytes.Equal(yml, expected) {
				t.Errorf("test %d: expecting contents %q, got %q", n, expected, yml)
			}
		}
	}
}

func TestGenerateEnvReadme(t *testing.T) {
	readme, _ := io.ReadAll(GenerateEnvReadme("HGI/common/some_environment"))
	expected, _ := files.ReadFile(filepath.Join("files", "shpc.readme"))

	if !bytes.Equal(readme, expected) {
		t.Errorf("expecting readme %q, got %q", expected, readme)
	}
}
