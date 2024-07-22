package ldap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	ldapapi "github.com/go-ldap/ldap/v3"
)

type mockLDAP struct {
	closing bool

	results map[string][]string
}

func (m *mockLDAP) IsClosing() bool {
	c := m.closing

	return c
}

func (m *mockLDAP) Search(r *ldapapi.SearchRequest) (*ldapapi.SearchResult, error) {
	var groups []*ldapapi.Entry

	for _, group := range m.results[r.Filter] {
		groups = append(groups, &ldapapi.Entry{
			Attributes: []*ldapapi.EntryAttribute{
				{
					Name:   "group",
					Values: []string{group},
				},
			},
		})
	}

	return &ldapapi.SearchResult{Entries: groups}, nil
}

func TestLDAP(t *testing.T) {
	ml := mockLDAP{
		results: map[string][]string{
			"user1": {"group1", "group2", "group3"},
			"user2": {"group1", "group4"},
			"user3": {},
		},
	}
	dial = func(_ string) (ldapConn, error) {
		ml.closing = false

		return &ml, nil
	}

	l, err := New("", "", "{{ . }}", "group")
	if err != nil {
		t.Fatalf("unexpected error creating LDAP connection: %s", err)
	}

	s := httptest.NewServer(l)

	for n, test := range [...]struct {
		User    string
		Closing bool
		Groups  []string
	}{
		{
			User:   "",
			Groups: []string{},
		},
		{
			User:   "unknown",
			Groups: []string{},
		},
		{
			User:   "user1",
			Groups: []string{"group1", "group2", "group3"},
		},
		{
			User:    "user2",
			Groups:  []string{"group1", "group4"},
			Closing: true,
		},
		{
			User:   "user3",
			Groups: []string{},
		},
	} {
		ml.closing = test.Closing

		var groups []string

		if resp, err := http.Post(s.URL, "text/plain", strings.NewReader(test.User)); err != nil {
			t.Errorf("test %d: unexpected error getting group list: %s", n+1, err)
		} else if err = json.NewDecoder(resp.Body).Decode(&groups); err != nil {
			t.Errorf("test %d: unexpected error decoding JSON response: %s", n+1, err)
		} else if !reflect.DeepEqual(groups, test.Groups) {
			t.Errorf("test %d: expecting groups %v, got %v", n+1, test.Groups, groups)
		} else if ml.closing {
			t.Errorf("test %d: expecting closing state to be false, got true", n+1)
		}
	}
}
