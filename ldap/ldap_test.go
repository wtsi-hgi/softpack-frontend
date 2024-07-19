package ldap

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLDAP(t *testing.T) {
	params := map[string]string{
		"SERVER": "",
		"BASE":   "",
		"FILTER": "",
		"ATTR":   "",
		"GROUPS": "",
	}

	for key := range params {
		keyname := "TEST_LDAP_" + key
		value := os.Getenv(keyname)
		if value == "" {
			t.Skip("set " + keyname + " to enable test")
		}

		params[key] = value
	}

	l, err := New(params["SERVER"], params["BASE"], params["FILTER"], params["ATTR"])
	if err != nil {
		t.Fatalf("unexpected error creating ldap connection: %s", err)
	}

	s := httptest.NewServer(l)

	resp, err := http.Post(s.URL, "text/plain", strings.NewReader("mw31"))
	if err != nil {
		t.Fatalf("unexpected error getting groups: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		io.Copy(os.Stdout, resp.Body)
		t.Fatalf("expecting status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var groups []string

	if err = json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		t.Fatalf("unexpected error decoding JSON: %s", err)
	}

	expectation := strings.Split(params["GROUPS"], "|")

	if !reflect.DeepEqual(groups, expectation) {
		t.Errorf("expecting groups %v, got %v", expectation, groups)
	}
}
