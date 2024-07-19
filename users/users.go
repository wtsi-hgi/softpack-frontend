package users

import (
	"encoding/json"
	"io"
	"net/http"
	"os/user"
	"strings"
)

func New() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var username strings.Builder

		io.Copy(&username, io.LimitReader(r.Body, 1024))

		u, err := user.Lookup(username.String())
		if err != nil {
			noGroups(w)

			return
		}

		gids, err := u.GroupIds()
		if err != nil || len(gids) == 0 {
			noGroups(w)

			return
		}

		var groups []string

		for _, gid := range gids {
			g, err := user.LookupGroupId(gid)
			if err != nil {
				noGroups(w)

				return
			}

			groups = append(groups, g.Name)
		}

		json.NewEncoder(w).Encode(groups)
	})
}

func noGroups(w io.Writer) {
	io.WriteString(w, "[]")
}
