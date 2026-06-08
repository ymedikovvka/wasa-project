package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"wasatext/service/database"
)

type forwardMessageRequest struct {
	ConversationID string `json:"conversationId"`
}

type commentMessageRequest struct {
	Emoticon string `json:"emoticon"`
}

func (rt *_router) messageByID(w http.ResponseWriter, r *http.Request) {
	messageID, action, ok := messagePathParts(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}

	switch action {
	case "":
		switch r.Method {
		case http.MethodDelete:
			rt.deleteMessage(w, r, messageID)
		default:
			w.Header().Set("Allow", http.MethodDelete)
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	case "forward":
		switch r.Method {
		case http.MethodPost:
			rt.forwardMessage(w, r, messageID)
		default:
			w.Header().Set("Allow", http.MethodPost)
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	case "comments/me":
		switch r.Method {
		case http.MethodPut:
			rt.commentMessage(w, r, messageID)
		case http.MethodDelete:
			rt.uncommentMessage(w, r, messageID)
		default:
			w.Header().Set("Allow", http.MethodPut+", "+http.MethodDelete)
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	default:
		writeError(w, http.StatusNotFound, "message not found")
	}
}

func (rt *_router) deleteMessage(w http.ResponseWriter, r *http.Request, messageID string) {
	currentUser, err := rt.currentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}

	err = rt.db.DeleteMessage(currentUser.ID, messageID)
	if errors.Is(err, database.ErrMessageNotFound) {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}
	if errors.Is(err, database.ErrMessageForbidden) {
		writeError(w, http.StatusForbidden, "user cannot delete messages sent by other users")
		return
	}
	if errors.Is(err, database.ErrUserNotFound) {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}
	if err != nil {
		rt.internalError(w, r, "messages:DeleteMessage", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) forwardMessage(w http.ResponseWriter, r *http.Request, messageID string) {
	currentUser, err := rt.currentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}

	var req forwardMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.ConversationID) == "" {
		writeError(w, http.StatusBadRequest, "invalid destination conversation supplied")
		return
	}

	message, err := rt.db.ForwardMessage(currentUser.ID, messageID, strings.TrimSpace(req.ConversationID))
	if errors.Is(err, database.ErrMessageNotFound) {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}
	if errors.Is(err, database.ErrConversationNotFound) {
		writeError(w, http.StatusNotFound, "destination conversation not found")
		return
	}
	if errors.Is(err, database.ErrConversationForbidden) {
		writeError(w, http.StatusForbidden, "user cannot access the source or destination conversation")
		return
	}
	if errors.Is(err, database.ErrUserNotFound) {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}
	if err != nil {
		rt.internalError(w, r, "messages:ForwardMessage", err)
		return
	}

	writeJSON(w, http.StatusCreated, toMessageResponse(message, nil))
}

func (rt *_router) commentMessage(w http.ResponseWriter, r *http.Request, messageID string) {
	currentUser, err := rt.currentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}

	var req commentMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	reaction, err := rt.db.CommentMessage(currentUser.ID, messageID, req.Emoticon)
	if errors.Is(err, database.ErrInvalidReaction) {
		writeError(w, http.StatusBadRequest, "invalid emoticon supplied")
		return
	}
	if errors.Is(err, database.ErrMessageNotFound) {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}
	if errors.Is(err, database.ErrConversationForbidden) {
		writeError(w, http.StatusForbidden, "user cannot access this message")
		return
	}
	if errors.Is(err, database.ErrUserNotFound) {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}
	if err != nil {
		rt.internalError(w, r, "messages:CommentMessage", err)
		return
	}

	writeJSON(w, http.StatusOK, toMessageReactionResponse(reaction))
}

func (rt *_router) uncommentMessage(w http.ResponseWriter, r *http.Request, messageID string) {
	currentUser, err := rt.currentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}

	err = rt.db.UncommentMessage(currentUser.ID, messageID)
	if errors.Is(err, database.ErrMessageNotFound) || errors.Is(err, database.ErrReactionNotFound) {
		writeError(w, http.StatusNotFound, "message or reaction not found")
		return
	}
	if errors.Is(err, database.ErrConversationForbidden) {
		writeError(w, http.StatusForbidden, "user cannot access this message")
		return
	}
	if errors.Is(err, database.ErrUserNotFound) {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}
	if err != nil {
		rt.internalError(w, r, "messages:UncommentMessage", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func messagePathParts(path string) (string, string, bool) {
	const prefix = "/messages/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}

	remaining := strings.TrimPrefix(path, prefix)
	parts := strings.Split(remaining, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", "", false
	}
	if len(parts) == 1 {
		return parts[0], "", true
	}
	if len(parts) == 2 && parts[1] == "forward" {
		return parts[0], parts[1], true
	}
	if len(parts) == 3 && parts[1] == "comments" && parts[2] == "me" {
		return parts[0], "comments/me", true
	}

	return "", "", false
}
