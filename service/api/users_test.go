package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"wasatext/service/database"
)

func TestUserEndpoints(t *testing.T) {
	handler := newTestHandler(t)

	mediID := loginForTest(t, handler, "Medi")
	_ = loginForTest(t, handler, "Lina")

	t.Run("search users requires identifier", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("search users returns matching users", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users?name=Me", nil)
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body userListResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
		if len(body.Users) != 1 || body.Users[0].Name != "Medi" {
			t.Fatalf("unexpected users response: %+v", body.Users)
		}
	})

	t.Run("rename current user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/users/me/name", jsonBody(t, setUserNameRequest{Name: "Medina"}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body userResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
		if body.ID != mediID || body.Name != "Medina" {
			t.Fatalf("unexpected user response: %+v", body)
		}
	})

	t.Run("rename rejects duplicate username", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/users/me/name", jsonBody(t, setUserNameRequest{Name: "Lina"}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
		}
	})

	t.Run("update current user photo", func(t *testing.T) {
		photo := "ZmFrZS1pbWFnZQ=="
		req := httptest.NewRequest(http.MethodPut, "/users/me/photo", jsonBody(t, setPhotoRequest{Photo: photo}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body userResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decoding response: %v", err)
		}
		if body.Photo == nil || *body.Photo != photo {
			t.Fatalf("unexpected photo response: %+v", body)
		}
	})

	t.Run("photo update rejects invalid base64", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/users/me/photo", jsonBody(t, setPhotoRequest{Photo: "not base64"}))
		req.Header.Set("Authorization", "Bearer "+mediID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()

	router, err := New(Config{Database: database.New()})
	if err != nil {
		t.Fatalf("creating API router: %v", err)
	}
	return router.Handler()
}

func loginForTest(t *testing.T, handler http.Handler, name string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/session", jsonBody(t, loginRequest{Name: name}))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected login status %d, got %d", http.StatusCreated, rec.Code)
	}

	var body loginResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding login response: %v", err)
	}
	if body.Identifier == "" {
		t.Fatal("expected non-empty identifier")
	}
	return body.Identifier
}

func jsonBody(t *testing.T, value interface{}) *bytes.Reader {
	t.Helper()

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(value); err != nil {
		t.Fatalf("encoding JSON body: %v", err)
	}
	return bytes.NewReader(buf.Bytes())
}
