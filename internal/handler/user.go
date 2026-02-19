package handler

import (
	"net/http"

	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/model"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

type getMeResponse struct {
	ID            int64                `json:"id"`
	TailscaleID   string               `json:"tailscale_id"`
	DisplayName   string               `json:"display_name"`
	CreatedAt     string               `json:"created_at"`
	TailscaleInfo *model.TailscaleInfo `json:"tailscale_info,omitempty"`
}

// GetMe returns the authenticated user's information
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp := getMeResponse{
		ID:            user.ID,
		TailscaleID:   user.TailscaleID,
		DisplayName:   user.DisplayName,
		CreatedAt:     user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		TailscaleInfo: middleware.TailscaleInfoFromContext(r.Context()),
	}

	jsonResponse(w, http.StatusOK, resp)
}
