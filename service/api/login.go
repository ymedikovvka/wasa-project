package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"wasatext/service/database"
)

type loginRequest struct {
	Name string `json:"name"`
}

type loginResponse struct {
	Identifier string `json:"identifier"`
}

func (rt *router) doLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	user, err := rt.db.LoginOrCreateUser(req.Name)
	if errors.Is(err, database.ErrInvalidUserName) {
		writeError(w, http.StatusBadRequest, "invalid username")
		return
	}
	if err != nil {
		rt.internalError(w, r, "login:LoginOrCreateUser", err)
		return
	}

	writeJSON(w, http.StatusCreated, loginResponse{Identifier: user.ID})
}
