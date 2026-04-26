package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateGroup(t *testing.T) {
	handler := newTestHandler(t)

	mediID := loginForTest(t, handler, "Medi")
	linaID := loginForTest(t, handler, "Lina")
	noraID := loginForTest(t, handler, "Nora")

	t.Run("requires identifier", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/groups", jsonBody(t, createGroupRequest{
			Name:      "Study Group",
			MemberIDs: []string{linaID},
		}))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("creates group with current user and selected members", func(t *testing.T) {
		group := createGroupForTest(t, handler, mediID, createGroupRequest{
			Name:      "Study Group",
			MemberIDs: []string{linaID, noraID},
		})

		if group.ID == "" || group.Type != "group" || group.Name != "Study Group" || group.Photo != nil {
			t.Fatalf("unexpected group response: %+v", group)
		}
		if len(group.Members) != 3 || len(group.Messages) != 0 {
			t.Fatalf("unexpected group members/messages: %+v", group)
		}

		members := map[string]bool{}
		for _, member := range group.Members {
			members[member.Name] = true
		}
		if !members["Medi"] || !members["Lina"] || !members["Nora"] {
			t.Fatalf("unexpected members: %+v", group.Members)
		}
	})

	t.Run("group appears in conversation list", func(t *testing.T) {
		group := createGroupForTest(t, handler, mediID, createGroupRequest{
			Name:      "Project Team",
			MemberIDs: []string{linaID},
		})

		req := httptest.NewRequest(http.MethodGet, "/conversations", nil)
		req.Header.Set("Authorization", "Bearer "+linaID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationListResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding conversation list response: %v", err)
		}

		found := false
		for _, conversation := range body.Conversations {
			if conversation.ID == group.ID && conversation.Name == "Project Team" && conversation.Type == "group" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected group in conversation list, got %+v", body.Conversations)
		}
	})

	t.Run("allows messages in group conversation", func(t *testing.T) {
		group := createGroupForTest(t, handler, mediID, createGroupRequest{
			Name:      "Messages Group",
			MemberIDs: []string{linaID},
		})

		message := sendMessageForTest(t, handler, linaID, group.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("Hello group"),
		})

		if message.ConversationID != group.ID || message.Text == nil || *message.Text != "Hello group" {
			t.Fatalf("unexpected group message: %+v", message)
		}
	})

	t.Run("rejects invalid group details", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/groups", jsonBody(t, createGroupRequest{
			Name:      " ",
			MemberIDs: []string{linaID},
		}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("rejects group without another member", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/groups", jsonBody(t, createGroupRequest{
			Name:      "Only Medi",
			MemberIDs: []string{mediID},
		}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("rejects missing user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/groups", jsonBody(t, createGroupRequest{
			Name:      "Missing User",
			MemberIDs: []string{"missing"},
		}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func TestSetGroupName(t *testing.T) {
	handler := newTestHandler(t)

	mediID := loginForTest(t, handler, "Medi")
	linaID := loginForTest(t, handler, "Lina")
	noraID := loginForTest(t, handler, "Nora")
	group := createGroupForTest(t, handler, mediID, createGroupRequest{
		Name:      "Old Name",
		MemberIDs: []string{linaID},
	})

	t.Run("requires identifier", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/groups/"+group.ID+"/name", jsonBody(t, setGroupNameRequest{
			Name: "New Name",
		}))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("renames group for member", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/groups/"+group.ID+"/name", jsonBody(t, setGroupNameRequest{
			Name: "Exam Team",
		}))
		req.Header.Set("Authorization", "Bearer "+linaID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding group response: %v", err)
		}
		if body.ID != group.ID || body.Name != "Exam Team" || body.Type != "group" {
			t.Fatalf("unexpected renamed group: %+v", body)
		}
	})

	t.Run("conversation list shows renamed group", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/conversations", nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationListResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding conversation list response: %v", err)
		}

		found := false
		for _, conversation := range body.Conversations {
			if conversation.ID == group.ID && conversation.Name == "Exam Team" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected renamed group in conversation list, got %+v", body.Conversations)
		}
	})

	t.Run("rejects invalid group name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/groups/"+group.ID+"/name", jsonBody(t, setGroupNameRequest{
			Name: " ",
		}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("forbids non-member", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/groups/"+group.ID+"/name", jsonBody(t, setGroupNameRequest{
			Name: "Nora Name",
		}))
		req.Header.Set("Authorization", "Bearer "+noraID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("returns not found for missing group", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/groups/missing/name", jsonBody(t, setGroupNameRequest{
			Name: "Missing",
		}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func TestSetGroupPhoto(t *testing.T) {
	handler := newTestHandler(t)

	mediID := loginForTest(t, handler, "Medi")
	linaID := loginForTest(t, handler, "Lina")
	noraID := loginForTest(t, handler, "Nora")
	group := createGroupForTest(t, handler, mediID, createGroupRequest{
		Name:      "Photo Group",
		MemberIDs: []string{linaID},
	})

	t.Run("requires identifier", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/groups/"+group.ID+"/photo", jsonBody(t, setPhotoRequest{
			Photo: "ZmFrZS1pbWFnZQ==",
		}))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("sets group photo for member", func(t *testing.T) {
		photo := "ZmFrZS1pbWFnZQ=="
		req := httptest.NewRequest(http.MethodPut, "/groups/"+group.ID+"/photo", jsonBody(t, setPhotoRequest{
			Photo: photo,
		}))
		req.Header.Set("Authorization", "Bearer "+linaID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding group response: %v", err)
		}
		if body.ID != group.ID || body.Photo == nil || *body.Photo != photo {
			t.Fatalf("unexpected group photo response: %+v", body)
		}
	})

	t.Run("conversation list shows group photo", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/conversations", nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationListResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding conversation list response: %v", err)
		}

		found := false
		for _, conversation := range body.Conversations {
			if conversation.ID == group.ID && conversation.Photo != nil && *conversation.Photo == "ZmFrZS1pbWFnZQ==" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected group photo in conversation list, got %+v", body.Conversations)
		}
	})

	t.Run("rejects invalid photo", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/groups/"+group.ID+"/photo", jsonBody(t, setPhotoRequest{
			Photo: "not base64",
		}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("forbids non-member", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/groups/"+group.ID+"/photo", jsonBody(t, setPhotoRequest{
			Photo: "ZmFrZS1pbWFnZQ==",
		}))
		req.Header.Set("Authorization", "Bearer "+noraID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("returns not found for missing group", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/groups/missing/photo", jsonBody(t, setPhotoRequest{
			Photo: "ZmFrZS1pbWFnZQ==",
		}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func TestAddToGroup(t *testing.T) {
	handler := newTestHandler(t)

	mediID := loginForTest(t, handler, "Medi")
	linaID := loginForTest(t, handler, "Lina")
	noraID := loginForTest(t, handler, "Nora")
	ottoID := loginForTest(t, handler, "Otto")
	group := createGroupForTest(t, handler, mediID, createGroupRequest{
		Name:      "Add Group",
		MemberIDs: []string{linaID},
	})

	t.Run("requires identifier", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/groups/"+group.ID+"/members/"+noraID, nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("adds user when requester is a member", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/groups/"+group.ID+"/members/"+noraID, nil)
		req.Header.Set("Authorization", "Bearer "+linaID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding group response: %v", err)
		}
		if len(body.Members) != 3 {
			t.Fatalf("expected 3 members, got %+v", body.Members)
		}

		members := map[string]bool{}
		for _, member := range body.Members {
			members[member.ID] = true
		}
		if !members[mediID] || !members[linaID] || !members[noraID] {
			t.Fatalf("unexpected members after add: %+v", body.Members)
		}
	})

	t.Run("added user can see and use group", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/conversations", nil)
		req.Header.Set("Authorization", "Bearer "+noraID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationListResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding conversation list response: %v", err)
		}

		found := false
		for _, conversation := range body.Conversations {
			if conversation.ID == group.ID {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected added user to see group, got %+v", body.Conversations)
		}

		message := sendMessageForTest(t, handler, noraID, group.ID, sendMessageRequest{
			Type: "text",
			Text: stringPointer("I joined"),
		})
		if message.Sender.ID != noraID || message.ConversationID != group.ID {
			t.Fatalf("unexpected message from added user: %+v", message)
		}
	})

	t.Run("adding existing member is idempotent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/groups/"+group.ID+"/members/"+noraID, nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding group response: %v", err)
		}
		if len(body.Members) != 3 {
			t.Fatalf("expected duplicate add to keep 3 members, got %+v", body.Members)
		}
	})

	t.Run("forbids non-member requester", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/groups/"+group.ID+"/members/"+ottoID, nil)
		req.Header.Set("Authorization", "Bearer "+ottoID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("returns not found for missing user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/groups/"+group.ID+"/members/missing", nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})

	t.Run("returns not found for missing group", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/groups/missing/members/"+ottoID, nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func TestLeaveGroup(t *testing.T) {
	handler := newTestHandler(t)

	mediID := loginForTest(t, handler, "Medi")
	linaID := loginForTest(t, handler, "Lina")
	noraID := loginForTest(t, handler, "Nora")
	group := createGroupForTest(t, handler, mediID, createGroupRequest{
		Name:      "Leave Group",
		MemberIDs: []string{linaID},
	})

	t.Run("requires identifier", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/groups/"+group.ID+"/members/me", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("removes current user from group", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/groups/"+group.ID+"/members/me", nil)
		req.Header.Set("Authorization", "Bearer "+linaID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
		}
	})

	t.Run("left user no longer sees group", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/conversations", nil)
		req.Header.Set("Authorization", "Bearer "+linaID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body conversationListResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding conversation list response: %v", err)
		}
		for _, conversation := range body.Conversations {
			if conversation.ID == group.ID {
				t.Fatalf("left user should not see group, got %+v", body.Conversations)
			}
		}
	})

	t.Run("remaining member still sees group without left user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/conversations/"+group.ID, nil)
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
		if len(body.Members) != 1 || body.Members[0].ID != mediID {
			t.Fatalf("expected only Medi to remain, got %+v", body.Members)
		}
	})

	t.Run("forbids user who is not a member", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/groups/"+group.ID+"/members/me", nil)
		req.Header.Set("Authorization", "Bearer "+noraID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("returns not found for missing group", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/groups/missing/members/me", nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func createGroupForTest(t *testing.T, handler http.Handler, currentUserID string, request createGroupRequest) conversationResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/groups", jsonBody(t, request))
	req.Header.Set("Authorization", "Bearer "+currentUserID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected group status %d, got %d", http.StatusCreated, rec.Code)
	}

	var body conversationResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding group response: %v", err)
	}
	return body
}
