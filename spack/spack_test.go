package spack

import (
	_ "embed"
	"fmt"
	"os"
	"testing"

	ispack "github.com/wtsi-hgi/softpack-frontend/internal/spack"
)

func TestMain(m *testing.M) {
	cleanup, err := ispack.Setup()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating fake spack: %s", err)

		os.Exit(1)
	}

	code := m.Run()

	cleanup()

	os.Exit(code)
}

func TestRecipeLoad(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatalf("unexpected error creating spack object: %s", err)
	}

	r, err := s.ListRecipes()
	if err != nil {
		t.Fatalf("unexpected error getting recipes: %s", err)
	}

	const (
		numRecipes        = 7469
		firstName         = "3dtk"
		firstNumVersions  = 2
		firstfirstVersion = "trunk"
	)

	if len(r) != numRecipes {
		t.Errorf("expecting %d recipes, got %d", numRecipes, len(r))
	} else if r[0].Name != firstName {
		t.Errorf("expecting first recipe to be %s, got %s", firstName, r[0].Name)
	} else if len(r[0].Version) != firstNumVersions {
		t.Errorf("expecting first recipe to have %d recipes, got %d", firstNumVersions, len(r[0].Version))
	} else if r[0].Version[0] != firstfirstVersion {
		t.Errorf("expecting first recipes first version to be %s, got %s", firstfirstVersion, r[0].Version[0])
	}
}
