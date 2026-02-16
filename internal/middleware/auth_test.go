package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/allen/fishscale/internal/model"
)

func TestUserFromContext_Present(t *testing.T) {
	user := &model.User{
		ID:          42,
		TailscaleID: "test-ts-id",
		DisplayName: "Test User",
	}

	ctx := WithUser(context.Background(), user)
	got := UserFromContext(ctx)
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.ID != 42 {
		t.Errorf("expected user ID 42, got %d", got.ID)
	}
	if got.DisplayName != "Test User" {
		t.Errorf("expected display name 'Test User', got %q", got.DisplayName)
	}
}

func TestUserFromContext_Missing(t *testing.T) {
	got := UserFromContext(context.Background())
	if got != nil {
		t.Errorf("expected nil user from empty context, got %+v", got)
	}
}

func TestDevAuth(t *testing.T) {
	var capturedUser *model.User

	handler := DevAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if capturedUser == nil {
		t.Fatal("expected user from DevAuth, got nil")
	}
	if capturedUser.ID != 1 {
		t.Errorf("expected dev user ID 1, got %d", capturedUser.ID)
	}
	if capturedUser.TailscaleID != "dev-user" {
		t.Errorf("expected tailscale_id 'dev-user', got %q", capturedUser.TailscaleID)
	}
}
