package users

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/user"
	"reflect"
	"strings"
	"testing"
)

func TestUsers(t *testing.T) {
	s := httptest.NewServer(New())

	u, err := user.Current()
	if err != nil {
		t.Fatalf("unexpected error getting current user: %s", err)
	}

	gids, err := u.GroupIds()
	if err != nil {
		t.Fatalf("unexpected error getting group IDs: %s", err)
	}

	myGroups := make([]string, 0, len(gids))

	for _, gid := range gids {
		group, err := user.LookupGroupId(gid)
		if err != nil {
			t.Fatalf("unexpected error getting group name: %s", err)
		}

		myGroups = append(myGroups, group.Name)
	}

	for n, test := range [...]struct {
		User   string
		Groups []string
	}{
		{
			Groups: []string{},
		},
		{
			User:   "non-existant-user",
			Groups: []string{},
		},
		{
			User:   u.Username,
			Groups: myGroups,
		},
	} {
		var groups []string

		if resp, err := http.Post(s.URL, "text/plain", strings.NewReader(test.User)); err != nil {
			t.Errorf("test %d: unexpected error getting group list: %s", n+1, err)
		} else if err = json.NewDecoder(resp.Body).Decode(&groups); err != nil {
			t.Errorf("test %d: unexpected error decoding JSON response: %s", n+1, err)
		} else if !reflect.DeepEqual(groups, test.Groups) {
			t.Errorf("test %d: expecting groups %v, got %v", n+1, test.Groups, groups)
		}
	}
}
