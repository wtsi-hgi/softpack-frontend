package environments

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/wtsi-hgi/softpack-frontend/artefacts"
	"github.com/wtsi-hgi/softpack-frontend/internal/git"
	"golang.org/x/net/websocket"
	"vimagination.zapto.org/jsonrpc"
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
				"users/userA/envA-1/" + builtBySoftpackFile: "",
				"users/userA/envA-1/" + readmeFile:          "README",
				"users/userA/envA-1/" + moduleFile:          "",
				"users/userA/envA-1/" + environmentsFile: `description: MY DESC
packages:
 - packageA@1
 - packageB@2
`,
			},
			Expectation: environments{
				"users/userA/envA-1": {
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
			t.Errorf("test %d: expecting envs %v, got %v", n+1, test.Expectation, envs)
		}
	}
}

func loadFromWebsocket(url string) (environments, error) {
	conn, err := websocket.Dial(url, "", path.Dir(url))
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	c := jsonrpc.NewClient(conn)

	defer c.Close()

	envsCh := make(chan environments, 1)

	c.Subscribe(-1, func(rm json.RawMessage) {
		var envs environments

		json.NewDecoder(bytes.NewReader(rm)).Decode(&envs)

		envsCh <- envs
	})

	go func() {
		time.Sleep(time.Second)

		close(envsCh)
	}()

	return <-envsCh, nil
}
