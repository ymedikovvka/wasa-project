package api

import (
	"errors"
	"net/http"
	"strings"
	"wasatext/service/database"
)

var errUnauthorized = errors.New("missing or invalid user identifier")

func (rt *router) currentUser(r *http.Request) (database.User, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return database.User{}, errUnauthorized
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return database.User{}, errUnauthorized
	}

	identifier := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if identifier == "" {
		return database.User{}, errUnauthorized
	}

	user, err := rt.db.GetUserByID(identifier)
	if err != nil {
		return database.User{}, errUnauthorized
	}
	return user, nil
}
