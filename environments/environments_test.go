package environments

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/wtsi-hgi/softpack-frontend/artefacts"
	"github.com/wtsi-hgi/softpack-frontend/internal/git"
)

var testFiles = map[string]string{}

func TestEnvironments(t *testing.T) {
	for n, test := range [...]struct {
		Files       map[string]string
		Expectation []environment
	}{
		{
			Expectation: []environment{},
		},
		{
			Files: map[string]string{
				"users/userA/envA-1/" + built_by_softpack_file: "",
				"users/userA/envA-1/" + readme_file:            "README",
				"users/userA/envA-1/" + module_file:            "",
				"users/userA/envA-1/" + environments_file: `description: MY DESC
packages:
 - packageA@1
 - packageB@2
`,
			},
			Expectation: []environment{
				{
					Path:        "users/userA",
					NameVersion: "envA-1",
					Packages:    []string{"packageA@1", "packageB@2"},
					Description: "MY DESC",
					ReadMe:      "README",
					Status:      envReady,
					SoftPack:    true,
				},
			},
		},
	} {
		g := git.New(t)
		g.Add(t, test.Files)

		var envs []environment

		if a, err := artefacts.New(artefacts.Remote(g.URL())); err != nil {
			t.Errorf("test %d: unexpected error creating artefacts: %s", n+1, err)
		} else if e, err := New(a); err != nil {
			t.Fatalf("test %d: unexpected error creating environments: %s", n+1, err)
		} else if resp, err := http.Get(httptest.NewServer(e).URL + "/environments.json"); err != nil {
			t.Fatalf("test %d: unexpected error getting environments: %s", n+1, err)
		} else if err := json.NewDecoder(resp.Body).Decode(&envs); err != nil {
			t.Fatalf("test %d: unexpected error decoding environments: %s", n+1, err)
		} else if !reflect.DeepEqual(envs, test.Expectation) {
			t.Errorf("test %d: expecting envs %v, got %v", n+1, test.Expectation, envs)
		}
	}
}
