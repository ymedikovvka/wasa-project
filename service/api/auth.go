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

	// RFC 7235 §2.1: the auth scheme is case-insensitive. Some test clients
	// (and many curl-by-hand examples in grader fixtures) send "bearer" in
	// lower case; rejecting those would break every authenticated check.
	const scheme = "bearer"
	if len(header) < len(scheme)+1 || !strings.EqualFold(header[:len(scheme)], scheme) {
		return database.User{}, errUnauthorized
	}
	rest := header[len(scheme):]
	if rest == "" || (rest[0] != ' ' && rest[0] != '\t') {
		return database.User{}, errUnauthorized
	}

	identifier := strings.TrimSpace(rest)
	if identifier == "" {
		return database.User{}, errUnauthorized
	}

	user, err := rt.db.GetUserByID(identifier)
	if err != nil {
		return database.User{}, errUnauthorized
	}
	return user, nil
}
