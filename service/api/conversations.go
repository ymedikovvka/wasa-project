package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"wasatext/service/database"
)

type startConversationRequest struct {
	UserID string `json:"userId"`
}

type conversationResponse struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Name     string            `json:"name"`
	Photo    *string           `json:"photo"`
	Members  []userResponse    `json:"members"`
	Messages []messageResponse `json:"messages"`
}

type conversationListResponse struct {
	Conversations []conversationPreviewResponse `json:"conversations"`
}

type conversationPreviewResponse struct {
	ID                   string  `json:"id"`
	Type                 string  `json:"type"`
	Name                 string  `json:"name"`
	Photo                *string `json:"photo"`
	LatestMessageAt      *string `json:"latestMessageAt"`
	LatestMessagePreview *string `json:"latestMessagePreview"`
	LatestMessageIsPhoto bool    `json:"latestMessageIsPhoto"`
}

type sendMessageRequest struct {
	Type             string  `json:"type"`
	Text             *string `json:"text"`
	Photo            *string `json:"photo"`
	ReplyToMessageID *string `json:"replyToMessageId"`
}

type messageResponse struct {
	ID             string                    `json:"id"`
	ConversationID string                    `json:"conversationId"`
	Sender         userResponse              `json:"sender"`
	Timestamp      string                    `json:"timestamp"`
	Type           string                    `json:"type"`
	Text           *string                   `json:"text"`
	Photo          *string                   `json:"photo"`
	ReplyTo        *messageReferenceResponse `json:"replyTo"`
	IsForwarded    bool                      `json:"isForwarded"`
	ForwardedFrom  string                    `json:"forwardedFrom,omitempty"`
	DeliveryStatus string                    `json:"deliveryStatus"`
	Reactions      []messageReactionResponse `json:"reactions"`
}

type messageReferenceResponse struct {
	ID         string `json:"id"`
	SenderName string `json:"senderName"`
	Preview    string `json:"preview"`
}

type messageReactionResponse struct {
	User     userResponse `json:"user"`
	Emoticon string       `json:"emoticon"`
}

func (rt *router) conversations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rt.getMyConversations(w, r)
	case http.MethodPost:
		rt.startConversation(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (rt *router) conversationByID(w http.ResponseWriter, r *http.Request) {
	conversationID, subresource, ok := conversationPathParts(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}

	switch subresource {
	case "":
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		rt.getConversation(w, r, conversationID)
	case "messages":
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		rt.sendMessage(w, r, conversationID)
	default:
		writeError(w, http.StatusNotFound, "conversation not found")
	}
}

func (rt *router) getMyConversations(w http.ResponseWriter, r *http.Request) {
	currentUser, err := rt.currentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}

	conversations, err := rt.db.ListConversationsForUser(currentUser.ID)
	if errors.Is(err, database.ErrUserNotFound) {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}
	if err != nil {
		rt.internalError(w, r, "conversations:ListConversationsForUser", err)
		return
	}

	response := conversationListResponse{
		Conversations: make([]conversationPreviewResponse, 0, len(conversations)),
	}
	for _, conversation := range conversations {
		response.Conversations = append(response.Conversations, toConversationPreviewResponse(conversation, currentUser.ID))
	}

	writeJSON(w, http.StatusOK, response)
}

func (rt *router) getConversation(w http.ResponseWriter, r *http.Request, conversationID string) {
	currentUser, err := rt.currentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}

	conversation, err := rt.db.GetConversationForUser(currentUser.ID, conversationID)
	if errors.Is(err, database.ErrConversationNotFound) {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	if errors.Is(err, database.ErrConversationForbidden) {
		writeError(w, http.StatusForbidden, "user is not a member of this conversation")
		return
	}
	if errors.Is(err, database.ErrUserNotFound) {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}
	if err != nil {
		rt.internalError(w, r, "conversations:GetConversationForUser", err)
		return
	}

	// Fetching a conversation implies the user has seen its contents; update
	// the per-user read marker so delivery statuses flip from "received" to
	// "read" for older messages. Failures here are non-fatal — the response
	// still carries the conversation payload.
	if err := rt.db.MarkConversationRead(currentUser.ID, conversationID, time.Now().UTC()); err != nil {
		rt.logger.Warn("mark conversation read failed",
			slog.String("user_id", currentUser.ID),
			slog.String("conversation_id", conversationID),
			slog.Any("error", err),
		)
	}

	writeJSON(w, http.StatusOK, toConversationResponse(conversation, currentUser.ID))
}

func (rt *router) startConversation(w http.ResponseWriter, r *http.Request) {
	currentUser, err := rt.currentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}

	var req startConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	conversation, created, err := rt.db.StartPrivateConversation(currentUser.ID, req.UserID)
	if errors.Is(err, database.ErrInvalidConversation) {
		writeError(w, http.StatusBadRequest, "invalid user supplied")
		return
	}
	if errors.Is(err, database.ErrUserNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		rt.internalError(w, r, "conversations:StartPrivateConversation", err)
		return
	}

	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}

	writeJSON(w, status, toConversationResponse(conversation, currentUser.ID))
}

func (rt *router) sendMessage(w http.ResponseWriter, r *http.Request, conversationID string) {
	currentUser, err := rt.currentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}

	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if !validMessageRequest(req) {
		writeError(w, http.StatusBadRequest, "invalid message")
		return
	}

	trimmedText := trimmedStringPointer(req.Text)
	trimmedPhoto := trimmedStringPointer(req.Photo)
	trimmedReplyTo := trimmedStringPointer(req.ReplyToMessageID)
	message, err := rt.db.SendMessage(currentUser.ID, conversationID, req.Type, trimmedText, trimmedPhoto, trimmedReplyTo)
	if errors.Is(err, database.ErrConversationNotFound) {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	if errors.Is(err, database.ErrConversationForbidden) {
		writeError(w, http.StatusForbidden, "user is not a member of this conversation")
		return
	}
	if errors.Is(err, database.ErrMessageNotFound) {
		writeError(w, http.StatusBadRequest, "reply message not found")
		return
	}
	if errors.Is(err, database.ErrInvalidMessage) {
		writeError(w, http.StatusBadRequest, "invalid message")
		return
	}
	if errors.Is(err, database.ErrUserNotFound) {
		writeError(w, http.StatusUnauthorized, "missing or invalid user identifier")
		return
	}
	if err != nil {
		rt.internalError(w, r, "conversations:SendMessage", err)
		return
	}

	writeJSON(w, http.StatusCreated, toMessageResponse(message, nil))
}

func toConversationResponse(conversation database.Conversation, currentUserID string) conversationResponse {
	response := conversationResponse{
		ID:       conversation.ID,
		Type:     conversation.Type,
		Name:     conversation.Name,
		Photo:    conversation.Photo,
		Members:  make([]userResponse, 0, len(conversation.Members)),
		Messages: make([]messageResponse, 0, len(conversation.Messages)),
	}

	for _, member := range conversation.Members {
		response.Members = append(response.Members, toUserResponse(member))
		if conversation.Type == "private" && member.ID != currentUserID {
			response.Name = member.Name
			response.Photo = member.Photo
		}
	}

	for _, message := range conversation.Messages {
		response.Messages = append(response.Messages, toMessageResponse(message, messageReferenceFor(conversation, message.ReplyToMessageID)))
	}

	return response
}

func toConversationPreviewResponse(conversation database.Conversation, currentUserID string) conversationPreviewResponse {
	full := toConversationResponse(conversation, currentUserID)
	response := conversationPreviewResponse{
		ID:                   full.ID,
		Type:                 full.Type,
		Name:                 full.Name,
		Photo:                full.Photo,
		LatestMessageAt:      nil,
		LatestMessagePreview: nil,
		LatestMessageIsPhoto: false,
	}

	if len(conversation.Messages) > 0 {
		latestMessage := conversation.Messages[len(conversation.Messages)-1]
		timestamp := latestMessage.Timestamp.Format(timeFormat)
		response.LatestMessageAt = &timestamp
		response.LatestMessageIsPhoto = latestMessage.Type == "photo"
		if latestMessage.Type == "text" {
			response.LatestMessagePreview = latestMessage.Text
		}
	}

	return response
}

func toMessageResponse(message database.Message, replyTo *messageReferenceResponse) messageResponse {
	response := messageResponse{
		ID:             message.ID,
		ConversationID: message.ConversationID,
		Sender:         toUserResponse(message.Sender),
		Timestamp:      message.Timestamp.Format(timeFormat),
		Type:           message.Type,
		Text:           message.Text,
		Photo:          message.Photo,
		ReplyTo:        replyTo,
		IsForwarded:    message.IsForwarded,
		ForwardedFrom:  message.ForwardedFrom,
		DeliveryStatus: deliveryStatusOrDefault(message.DeliveryStatus),
		Reactions:      []messageReactionResponse{},
	}

	for _, reaction := range message.Reactions {
		response.Reactions = append(response.Reactions, toMessageReactionResponse(reaction))
	}

	return response
}

func toMessageReactionResponse(reaction database.MessageReaction) messageReactionResponse {
	return messageReactionResponse{
		User:     toUserResponse(reaction.User),
		Emoticon: reaction.Emoticon,
	}
}

func messageReferenceFor(conversation database.Conversation, messageID *string) *messageReferenceResponse {
	if messageID == nil {
		return nil
	}

	for _, message := range conversation.Messages {
		if message.ID == *messageID {
			return &messageReferenceResponse{
				ID:         message.ID,
				SenderName: message.Sender.Name,
				Preview:    messagePreview(message),
			}
		}
	}

	return nil
}

func messagePreview(message database.Message) string {
	if message.Type == "photo" {
		return "Photo"
	}
	if message.Text == nil {
		return ""
	}
	return *message.Text
}

func validMessageRequest(req sendMessageRequest) bool {
	switch req.Type {
	case "text":
		return trimmedStringPointer(req.Text) != nil
	case "photo":
		photo := trimmedStringPointer(req.Photo)
		return photo != nil && validBase64Photo(*photo)
	default:
		return false
	}
}

func trimmedStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

const timeFormat = "2006-01-02T15:04:05Z"

// deliveryStatusOrDefault falls back to the "received" marker when the
// database layer has not yet annotated a message (e.g. outbound sends that
// bypass ListConversationsForUser / GetConversationForUser).
func deliveryStatusOrDefault(status string) string {
	if status == "" {
		return "received"
	}
	return status
}

func conversationPathParts(path string) (string, string, bool) {
	const prefix = "/conversations/"
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
	if len(parts) == 2 && parts[1] == "messages" {
		return parts[0], parts[1], true
	}

	return "", "", false
}
