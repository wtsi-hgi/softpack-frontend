package spack

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/wtsi-hgi/softpack-frontend/internal/git"
)

func TestRecipeLoad(t *testing.T) {
	sr := git.New(t)

	sr.Add(t, map[string]string{
		spackPackages + "/abc/package.py": "version(\"1.1\")\nversion(\"1.2\")",
		spackPackages + "/def/package.py": "version(\"dev\")\nversion(\"3.1.3\")",
	})

	spackRepo = sr.URL()

	s, err := New(plumbing.NewBranchReferenceName("master"))
	if err != nil {
		t.Fatalf("unexpected error creating spack object: %s", err)
	}

	cr := git.New(t)
	cr.Add(t, map[string]string{
		customPackages + "/ghi/package.py": "version(\"1\")\nversion(\"2\")\nversion(\"3\")",
		customPackages + "/def/package.py": "version(\"dev\")\nversion(\"3.1.4\")",
	})

	if err := s.WatchRemote(cr.URL(), -1); err != nil {
		t.Fatalf("unexpected error getting remote: %s", err)
	}

	ts := httptest.NewServer(s)

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error getting JSON: %s", err)
	}

	var recipes []recipe

	expectation := []recipe{
		{"abc", []string{"1.1", "1.2"}},
		{"def", []string{"dev", "3.1.4"}},
		{"ghi", []string{"1", "2", "3"}},
	}

	if err := json.NewDecoder(resp.Body).Decode(&recipes); err != nil {
		t.Errorf("unexpected error decoding JSON: %s", err)
	} else if !reflect.DeepEqual(recipes, expectation) {
		t.Errorf("expecting recipes %v, got %v", expectation, recipes)
	}
}
