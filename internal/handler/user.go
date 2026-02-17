package handler

import (
	"net/http"

	"github.com/allen/fishscale/internal/middleware"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// GetMe returns the authenticated user's information
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	jsonResponse(w, http.StatusOK, user)
}
