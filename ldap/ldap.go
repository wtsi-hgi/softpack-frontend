package ldap

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"text/template"

	ldapapi "github.com/go-ldap/ldap/v3"
)

type ldapConn interface {
	IsClosing() bool
	Search(*ldapapi.SearchRequest) (*ldapapi.SearchResult, error)
}

type ldap struct {
	filter    *template.Template
	url       string
	basedn    string
	groupAttr string

	mu   sync.Mutex
	conn ldapConn
}

var dial func(string) (ldapConn, error) = func(url string) (ldapConn, error) {
	return ldapapi.DialURL(url)
}

func New(url, basedn, filter, groupAttr string) (http.Handler, error) {
	c, err := dial(url)
	if err != nil {
		return nil, err
	}

	filterTemplate, err := template.New("").Parse(filter)
	if err != nil {
		return nil, err
	}

	return &ldap{
		filter:    filterTemplate,
		url:       url,
		basedn:    basedn,
		groupAttr: groupAttr,
		conn:      c,
	}, nil
}

func (l *ldap) getUserGroups(user string) ([]string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var filter strings.Builder

	l.filter.Execute(&filter, user)

	if l.conn.IsClosing() {
		conn, err := dial(l.url)
		if err != nil {
			return nil, err
		}

		l.conn = conn
	}

	results, err := l.conn.Search(&ldapapi.SearchRequest{
		BaseDN:       l.basedn,
		Scope:        ldapapi.ScopeWholeSubtree,
		DerefAliases: ldapapi.NeverDerefAliases,
		SizeLimit:    0,
		TimeLimit:    0,
		TypesOnly:    false,
		Filter:       filter.String(),
	})
	if err != nil {
		return nil, err
	}

	var groups []string

	for _, entry := range results.Entries {
		for _, attr := range entry.Attributes {
			if attr.Name == l.groupAttr {
				groups = append(groups, attr.Values...)
			}
		}
	}

	return groups, nil
}

func (l *ldap) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var user strings.Builder

	io.Copy(&user, io.LimitReader(r.Body, 1024))
	r.Body.Close()

	groups, err := l.getUserGroups(user.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if groups == nil {
		groups = make([]string, 0)
	}

	json.NewEncoder(w).Encode(groups)
}
