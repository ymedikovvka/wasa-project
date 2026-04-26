package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"wasatext/service/database"
)

type userResponse struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Photo *string `json:"photo"`
}

type userListResponse struct {
	Users []userResponse `json:"users"`
}

type setUserNameRequest struct {
	Name string `json:"name"`
}

type setPhotoRequest struct {
	Photo string `json:"photo"`
}

func (rt *router) users(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rt.searchUsers(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (rt *router) myUserName(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		rt.setMyUserName(w, r)
	default:
		w.Header().Set("Allow", http.MethodPut)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (rt *router) myPhoto(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		rt.setMyPhoto(w, r)
	default:
		w.Header().Set("Allow", http.MethodPut)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (rt *router) searchUsers(w http.ResponseWriter, r *http.Request) {
	if _, err := rt.currentUser(r); err != nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}

	users, err := rt.db.ListUsers(r.URL.Query().Get("name"))
	if err != nil {
		rt.internalError(w, r, "users:ListUsers", err)
		return
	}

	response := userListResponse{
		Users: make([]userResponse, 0, len(users)),
	}
	for _, user := range users {
		response.Users = append(response.Users, toUserResponse(user))
	}

	writeJSON(w, http.StatusOK, response)
}

func (rt *router) setMyUserName(w http.ResponseWriter, r *http.Request) {
	currentUser, err := rt.currentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}

	var req setUserNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	updatedUser, err := rt.db.SetUserName(currentUser.ID, req.Name)
	if errors.Is(err, database.ErrInvalidUserName) {
		writeError(w, http.StatusBadRequest, "invalid username")
		return
	}
	if errors.Is(err, database.ErrUserNameTaken) {
		writeError(w, http.StatusConflict, "username already in use")
		return
	}
	if errors.Is(err, database.ErrUserNotFound) {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}
	if err != nil {
		rt.internalError(w, r, "users:SetUserName", err)
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(updatedUser))
}

func (rt *router) setMyPhoto(w http.ResponseWriter, r *http.Request) {
	currentUser, err := rt.currentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}

	var req setPhotoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if !validBase64Photo(req.Photo) {
		writeError(w, http.StatusBadRequest, "invalid photo")
		return
	}

	updatedUser, err := rt.db.SetUserPhoto(currentUser.ID, req.Photo)
	if errors.Is(err, database.ErrUserNotFound) {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}
	if err != nil {
		rt.internalError(w, r, "users:SetUserPhoto", err)
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(updatedUser))
}

// maxPhotoBytes is the hard ceiling for a decoded profile / group photo. The
// cap keeps a single request (and the row it writes) from ballooning the
// SQLite database — 2 MiB comfortably covers a 1024×1024 JPEG without giving
// malicious clients a denial-of-storage vector.
const maxPhotoBytes = 2 * 1024 * 1024

func validBase64Photo(photo string) bool {
	photo = strings.TrimSpace(photo)
	if photo == "" {
		return false
	}

	// Fail fast on the encoded length: base64 inflates bytes by ~4/3, so any
	// payload above 4/3 · maxPhotoBytes is guaranteed to exceed the limit.
	if len(photo) > (maxPhotoBytes*4/3)+4 {
		return false
	}

	decoded, err := base64.StdEncoding.DecodeString(photo)
	if err != nil {
		return false
	}
	return len(decoded) > 0 && len(decoded) <= maxPhotoBytes
}

func toUserResponse(user database.User) userResponse {
	return userResponse{
		ID:    user.ID,
		Name:  user.Name,
		Photo: user.Photo,
	}
}
