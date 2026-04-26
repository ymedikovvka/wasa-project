package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStartConversation(t *testing.T) {
	handler := newTestHandler(t)

	mediID := loginForTest(t, handler, "Medi")
	linaID := loginForTest(t, handler, "Lina")

	t.Run("requires identifier", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/conversations", jsonBody(t, startConversationRequest{UserID: linaID}))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("creates private conversation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/conversations", jsonBody(t, startConversationRequest{UserID: linaID}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
		}

		var body conversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
		if body.ID == "" || body.Type != "private" || body.Name != "Lina" {
			t.Fatalf("unexpected conversation response: %+v", body)
		}
		if len(body.Members) != 2 || len(body.Messages) != 0 {
			t.Fatalf("unexpected conversation details: %+v", body)
		}
	})

	t.Run("returns existing private conversation", func(t *testing.T) {
		first := startConversationForTest(t, handler, mediID, linaID)

		req := httptest.NewRequest(http.MethodPost, "/conversations", jsonBody(t, startConversationRequest{UserID: mediID}))
		req.Header.Set("Authorization", "Bearer "+linaID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var second conversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&second); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
		if second.ID != first.ID {
			t.Fatalf("expected existing conversation %q, got %q", first.ID, second.ID)
		}
		if second.Name != "Medi" {
			t.Fatalf("expected conversation name from Lina's point of view, got %q", second.Name)
		}
	})

	t.Run("rejects conversation with self", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/conversations", jsonBody(t, startConversationRequest{UserID: mediID}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("rejects missing user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/conversations", jsonBody(t, startConversationRequest{UserID: "missing"}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func TestGetMyConversations(t *testing.T) {
	handler := newTestHandler(t)

	mediID := loginForTest(t, handler, "Medi")
	linaID := loginForTest(t, handler, "Lina")
	noraID := loginForTest(t, handler, "Nora")

	t.Run("requires identifier", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/conversations", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("returns current user's conversations", func(t *testing.T) {
		_ = startConversationForTest(t, handler, mediID, linaID)
		_ = startConversationForTest(t, handler, mediID, noraID)

		req := httptest.NewRequest(http.MethodGet, "/conversations", nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationListResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
		if len(body.Conversations) != 2 {
			t.Fatalf("expected 2 conversations, got %+v", body.Conversations)
		}

		names := map[string]bool{}
		for _, conversation := range body.Conversations {
			names[conversation.Name] = true
			if conversation.Type != "private" {
				t.Fatalf("expected private conversation, got %+v", conversation)
			}
			if conversation.LatestMessageAt != nil || conversation.LatestMessagePreview != nil || conversation.LatestMessageIsPhoto {
				t.Fatalf("expected empty latest-message preview, got %+v", conversation)
			}
		}
		if !names["Lina"] || !names["Nora"] {
			t.Fatalf("unexpected conversation names: %+v", names)
		}
	})
}

func TestGetConversation(t *testing.T) {
	handler := newTestHandler(t)

	mediID := loginForTest(t, handler, "Medi")
	linaID := loginForTest(t, handler, "Lina")
	noraID := loginForTest(t, handler, "Nora")
	conversation := startConversationForTest(t, handler, mediID, linaID)

	t.Run("requires identifier", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/conversations/"+conversation.ID, nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("returns conversation for member", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/conversations/"+conversation.ID, nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
		if body.ID != conversation.ID || body.Name != "Lina" || len(body.Members) != 2 || len(body.Messages) != 0 {
			t.Fatalf("unexpected conversation response: %+v", body)
		}
	})

	t.Run("forbids non-member", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/conversations/"+conversation.ID, nil)
		req.Header.Set("Authorization", "Bearer "+noraID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("returns not found for missing conversation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/conversations/missing", nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})

	t.Run("rejects unsupported get on messages path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/conversations/"+conversation.ID+"/messages", nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
		}
	})
}

func TestSendMessage(t *testing.T) {
	handler := newTestHandler(t)

	mediID := loginForTest(t, handler, "Medi")
	linaID := loginForTest(t, handler, "Lina")
	noraID := loginForTest(t, handler, "Nora")
	conversation := startConversationForTest(t, handler, mediID, linaID)

	t.Run("sends text message", func(t *testing.T) {
		message := sendMessageForTest(t, handler, mediID, conversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("Hello Lina"),
		})

		if message.ID == "" || message.Type != "text" || message.Text == nil || *message.Text != "Hello Lina" {
			t.Fatalf("unexpected message response: %+v", message)
		}
		if message.Photo != nil || message.ReplyTo != nil || message.DeliveryStatus != "received" || len(message.Reactions) != 0 {
			t.Fatalf("unexpected message details: %+v", message)
		}
	})

	t.Run("sends photo message", func(t *testing.T) {
		photo := "ZmFrZS1pbWFnZQ=="
		message := sendMessageForTest(t, handler, linaID, conversation.ID, sendMessageRequest{
			Type:  "photo",
			Photo: &photo,
		})

		if message.Type != "photo" || message.Photo == nil || *message.Photo != photo || message.Text != nil {
			t.Fatalf("unexpected photo message response: %+v", message)
		}
	})

	t.Run("sends reply message and conversation returns chronological messages", func(t *testing.T) {
		first := sendMessageForTest(t, handler, mediID, conversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("Question"),
		})
		reply := sendMessageForTest(t, handler, linaID, conversation.ID, sendMessageRequest{
			Type:             "text",
			Text:             stringPointer("Answer"),
			ReplyToMessageID: &first.ID,
		})

		req := httptest.NewRequest(http.MethodGet, "/conversations/"+conversation.ID, nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
		if len(body.Messages) < 2 || body.Messages[len(body.Messages)-1].ID != reply.ID {
			t.Fatalf("expected newest message last, got %+v", body.Messages)
		}
		newest := body.Messages[len(body.Messages)-1]
		if newest.ReplyTo == nil || newest.ReplyTo.ID != first.ID || newest.ReplyTo.Preview != "Question" {
			t.Fatalf("unexpected reply reference: %+v", newest.ReplyTo)
		}
	})

	t.Run("conversation list includes latest text preview", func(t *testing.T) {
		latest := "Latest preview"
		_ = sendMessageForTest(t, handler, mediID, conversation.ID, sendMessageRequest{
			Type: "text",
			Text: &latest,
		})

		req := httptest.NewRequest(http.MethodGet, "/conversations", nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationListResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
		if len(body.Conversations) != 1 {
			t.Fatalf("expected one conversation, got %+v", body.Conversations)
		}
		preview := body.Conversations[0]
		if preview.LatestMessagePreview == nil || *preview.LatestMessagePreview != latest || preview.LatestMessageAt == nil || preview.LatestMessageIsPhoto {
			t.Fatalf("unexpected latest preview: %+v", preview)
		}
	})

	t.Run("rejects invalid message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/conversations/"+conversation.ID+"/messages", jsonBody(t, sendMessageRequest{
			Type: "text",
			Text: stringPointer(" "),
		}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("forbids non-member", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/conversations/"+conversation.ID+"/messages", jsonBody(t, sendMessageRequest{
			Type: "text",
			Text: stringPointer("Hello"),
		}))
		req.Header.Set("Authorization", "Bearer "+noraID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("rejects missing reply message", func(t *testing.T) {
		missing := "missing"
		req := httptest.NewRequest(http.MethodPost, "/conversations/"+conversation.ID+"/messages", jsonBody(t, sendMessageRequest{
			Type:             "text",
			Text:             stringPointer("Reply"),
			ReplyToMessageID: &missing,
		}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestDeleteMessage(t *testing.T) {
	handler := newTestHandler(t)

	mediID := loginForTest(t, handler, "Medi")
	linaID := loginForTest(t, handler, "Lina")
	conversation := startConversationForTest(t, handler, mediID, linaID)

	t.Run("deletes own message", func(t *testing.T) {
		message := sendMessageForTest(t, handler, mediID, conversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("Delete me"),
		})

		req := httptest.NewRequest(http.MethodDelete, "/messages/"+message.ID, nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/conversations/"+conversation.ID, nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec = httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
		for _, existing := range body.Messages {
			if existing.ID == message.ID {
				t.Fatalf("deleted message still present: %+v", existing)
			}
		}
	})

	t.Run("forbids deleting another user's message", func(t *testing.T) {
		message := sendMessageForTest(t, handler, mediID, conversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("Mine"),
		})

		req := httptest.NewRequest(http.MethodDelete, "/messages/"+message.ID, nil)
		req.Header.Set("Authorization", "Bearer "+linaID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("returns not found for missing message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/messages/missing", nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})

	t.Run("requires identifier", func(t *testing.T) {
		message := sendMessageForTest(t, handler, mediID, conversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("Auth please"),
		})

		req := httptest.NewRequest(http.MethodDelete, "/messages/"+message.ID, nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}

func TestForwardMessage(t *testing.T) {
	handler := newTestHandler(t)

	mediID := loginForTest(t, handler, "Medi")
	linaID := loginForTest(t, handler, "Lina")
	noraID := loginForTest(t, handler, "Nora")
	ottoID := loginForTest(t, handler, "Otto")
	sourceConversation := startConversationForTest(t, handler, mediID, linaID)
	destinationConversation := startConversationForTest(t, handler, mediID, noraID)
	otherConversation := startConversationForTest(t, handler, noraID, ottoID)

	t.Run("forwards text message to accessible conversation", func(t *testing.T) {
		sourceMessage := sendMessageForTest(t, handler, linaID, sourceConversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("Forward this"),
		})

		forwarded := forwardMessageForTest(t, handler, mediID, sourceMessage.ID, destinationConversation.ID)

		if forwarded.ID == sourceMessage.ID || forwarded.Type != "text" || forwarded.Text == nil || *forwarded.Text != "Forward this" {
			t.Fatalf("unexpected forwarded message: %+v", forwarded)
		}
		if forwarded.ConversationID != destinationConversation.ID || forwarded.Sender.ID != mediID {
			t.Fatalf("unexpected forwarded destination/sender: %+v", forwarded)
		}
		if !forwarded.IsForwarded || forwarded.ForwardedFrom != "Lina" {
			t.Fatalf("expected forwarded metadata, got %+v", forwarded)
		}
	})

	t.Run("forwards photo message", func(t *testing.T) {
		photo := "ZmFrZS1pbWFnZQ=="
		sourceMessage := sendMessageForTest(t, handler, mediID, sourceConversation.ID, sendMessageRequest{
			Type:  "photo",
			Photo: &photo,
		})

		forwarded := forwardMessageForTest(t, handler, mediID, sourceMessage.ID, destinationConversation.ID)

		if forwarded.Type != "photo" || forwarded.Photo == nil || *forwarded.Photo != photo || forwarded.Text != nil {
			t.Fatalf("unexpected forwarded photo message: %+v", forwarded)
		}
	})

	t.Run("rejects inaccessible destination conversation", func(t *testing.T) {
		sourceMessage := sendMessageForTest(t, handler, mediID, sourceConversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("No access"),
		})

		req := httptest.NewRequest(http.MethodPost, "/messages/"+sourceMessage.ID+"/forward", jsonBody(t, forwardMessageRequest{
			ConversationID: otherConversation.ID,
		}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("rejects inaccessible source message", func(t *testing.T) {
		sourceMessage := sendMessageForTest(t, handler, noraID, otherConversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("Private elsewhere"),
		})

		req := httptest.NewRequest(http.MethodPost, "/messages/"+sourceMessage.ID+"/forward", jsonBody(t, forwardMessageRequest{
			ConversationID: destinationConversation.ID,
		}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("returns not found for missing source message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/messages/missing/forward", jsonBody(t, forwardMessageRequest{
			ConversationID: destinationConversation.ID,
		}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})

	t.Run("rejects missing destination conversation", func(t *testing.T) {
		sourceMessage := sendMessageForTest(t, handler, mediID, sourceConversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("Missing destination"),
		})

		req := httptest.NewRequest(http.MethodPost, "/messages/"+sourceMessage.ID+"/forward", jsonBody(t, forwardMessageRequest{
			ConversationID: "missing",
		}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func TestCommentMessage(t *testing.T) {
	handler := newTestHandler(t)

	mediID := loginForTest(t, handler, "Medi")
	linaID := loginForTest(t, handler, "Lina")
	noraID := loginForTest(t, handler, "Nora")
	ottoID := loginForTest(t, handler, "Otto")
	conversation := startConversationForTest(t, handler, mediID, linaID)
	otherConversation := startConversationForTest(t, handler, noraID, ottoID)

	t.Run("adds reaction to accessible message", func(t *testing.T) {
		message := sendMessageForTest(t, handler, mediID, conversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("React here"),
		})

		reaction := commentMessageForTest(t, handler, linaID, message.ID, ":thumbsup:")

		if reaction.User.ID != linaID || reaction.Emoticon != "👍" {
			t.Fatalf("unexpected reaction response: %+v", reaction)
		}

		req := httptest.NewRequest(http.MethodGet, "/conversations/"+conversation.ID, nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding conversation response: %v", err)
		}
		found := false
		for _, existing := range body.Messages {
			if existing.ID == message.ID && len(existing.Reactions) == 1 && existing.Reactions[0].Emoticon == "👍" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected reaction in conversation response: %+v", body.Messages)
		}
	})

	t.Run("replaces current user's reaction", func(t *testing.T) {
		message := sendMessageForTest(t, handler, mediID, conversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("Replace reaction"),
		})

		_ = commentMessageForTest(t, handler, linaID, message.ID, ":heart:")
		replaced := commentMessageForTest(t, handler, linaID, message.ID, ":ok:")

		if replaced.Emoticon != "👌" {
			t.Fatalf("expected replacement reaction, got %+v", replaced)
		}

		req := httptest.NewRequest(http.MethodGet, "/conversations/"+conversation.ID, nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding conversation response: %v", err)
		}
		for _, existing := range body.Messages {
			if existing.ID == message.ID {
				if len(existing.Reactions) != 1 || existing.Reactions[0].Emoticon != "👌" {
					t.Fatalf("expected one replaced reaction, got %+v", existing.Reactions)
				}
			}
		}
	})

	t.Run("normalizes legacy heart alias", func(t *testing.T) {
		message := sendMessageForTest(t, handler, mediID, conversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("Legacy heart"),
		})

		reaction := commentMessageForTest(t, handler, linaID, message.ID, "<3")

		if reaction.Emoticon != "❤️" {
			t.Fatalf("expected canonical heart emoji, got %+v", reaction)
		}
	})

	t.Run("rejects invalid emoticon", func(t *testing.T) {
		message := sendMessageForTest(t, handler, mediID, conversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("Invalid reaction"),
		})

		req := httptest.NewRequest(http.MethodPut, "/messages/"+message.ID+"/comments/me", jsonBody(t, commentMessageRequest{
			Emoticon: " ",
		}))
		req.Header.Set("Authorization", "Bearer "+linaID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("forbids inaccessible message", func(t *testing.T) {
		message := sendMessageForTest(t, handler, noraID, otherConversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("Elsewhere"),
		})

		req := httptest.NewRequest(http.MethodPut, "/messages/"+message.ID+"/comments/me", jsonBody(t, commentMessageRequest{
			Emoticon: ":wave:",
		}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("returns not found for missing message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/messages/missing/comments/me", jsonBody(t, commentMessageRequest{
			Emoticon: ":wave:",
		}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})

	t.Run("removes own reaction", func(t *testing.T) {
		message := sendMessageForTest(t, handler, mediID, conversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("Remove reaction"),
		})
		_ = commentMessageForTest(t, handler, linaID, message.ID, ":thumbsup:")

		req := httptest.NewRequest(http.MethodDelete, "/messages/"+message.ID+"/comments/me", nil)
		req.Header.Set("Authorization", "Bearer "+linaID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/conversations/"+conversation.ID, nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec = httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding conversation response: %v", err)
		}
		for _, existing := range body.Messages {
			if existing.ID == message.ID && len(existing.Reactions) != 0 {
				t.Fatalf("expected reaction to be removed, got %+v", existing.Reactions)
			}
		}
	})

	t.Run("returns not found when removing missing reaction", func(t *testing.T) {
		message := sendMessageForTest(t, handler, mediID, conversation.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("No reaction"),
		})

		req := httptest.NewRequest(http.MethodDelete, "/messages/"+message.ID+"/comments/me", nil)
		req.Header.Set("Authorization", "Bearer "+linaID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func startConversationForTest(t *testing.T, handler http.Handler, currentUserID string, otherUserID string) conversationResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/conversations", jsonBody(t, startConversationRequest{UserID: otherUserID}))
	req.Header.Set("Authorization", "Bearer "+currentUserID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("expected conversation status 200 or 201, got %d", rec.Code)
	}

	var body conversationResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding conversation response: %v", err)
	}
	return body
}

func sendMessageForTest(t *testing.T, handler http.Handler, currentUserID string, conversationID string, request sendMessageRequest) messageResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/conversations/"+conversationID+"/messages", jsonBody(t, request))
	req.Header.Set("Authorization", "Bearer "+currentUserID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected message status %d, got %d", http.StatusCreated, rec.Code)
	}

	var body messageResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding message response: %v", err)
	}
	return body
}

func forwardMessageForTest(t *testing.T, handler http.Handler, currentUserID string, sourceMessageID string, destinationConversationID string) messageResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/messages/"+sourceMessageID+"/forward", jsonBody(t, forwardMessageRequest{
		ConversationID: destinationConversationID,
	}))
	req.Header.Set("Authorization", "Bearer "+currentUserID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected forward status %d, got %d", http.StatusCreated, rec.Code)
	}

	var body messageResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding forward response: %v", err)
	}
	return body
}

func commentMessageForTest(t *testing.T, handler http.Handler, currentUserID string, messageID string, emoticon string) messageReactionResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodPut, "/messages/"+messageID+"/comments/me", jsonBody(t, commentMessageRequest{
		Emoticon: emoticon,
	}))
	req.Header.Set("Authorization", "Bearer "+currentUserID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected comment status %d, got %d", http.StatusOK, rec.Code)
	}

	var body messageReactionResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding comment response: %v", err)
	}
	return body
}

func stringPointer(value string) *string {
	return &value
}
