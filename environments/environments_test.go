package environments

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wtsi-hgi/softpack-frontend/artefacts"
	"github.com/wtsi-hgi/softpack-frontend/internal/git"
)

func TestEnvironments(t *testing.T) {
	for n, test := range [...]struct {
		Files       map[string]string
		Expectation environments
	}{
		{
			Expectation: environments{},
		},
		{
			Files: map[string]string{
				artefacts.Environments + "/users/userA/envA-1/" + builtBySoftpackFile: "",
				artefacts.Environments + "/users/userA/envA-1/" + readmeFile:          "README",
				artefacts.Environments + "/users/userA/envA-1/" + moduleFile:          "",
				artefacts.Environments + "/users/userA/envA-1/" + environmentsFile: `description: MY DESC
packages:
 - packageA@1
 - packageB@2
`,
			},
			Expectation: environments{
				"users/userA/envA-1": {
					Tags:        []string{},
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

		if a, err := artefacts.New(artefacts.Remote(g.URL())); err != nil {
			t.Errorf("test %d: unexpected error creating artefacts: %s", n+1, err)
		} else if e, err := New(a); err != nil {
			t.Fatalf("test %d: unexpected error creating environments: %s", n+1, err)
		} else if envs, err := loadFromWebsocket("ws" + httptest.NewServer(e).URL[4:] + socketPath); err != nil {
			t.Fatalf("test %d: unexpected error getting environments: %s", n+1, err)
		} else if !reflect.DeepEqual(envs, test.Expectation) {
			t.Errorf("test %d: expecting envs %#v, got %#v", n+1, test.Expectation, envs)
		}
	}
}

func loadFromWebsocket(url string) (environments, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	envsCh := make(chan environments, 1)

	conn.SetReadDeadline(time.Now().Add(time.Second))

	go func() {
		var (
			resp response
			envs environments
		)

		if err = conn.ReadJSON(&resp); err != nil {
			close(envsCh)

			return
		}

		if err = json.Unmarshal(resp.Result, &envs); err != nil {
			close(envsCh)

			return
		}

		envsCh <- envs
	}()

	return <-envsCh, err
}
