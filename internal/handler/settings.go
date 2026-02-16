package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/model"
)

type SettingsHandler struct {
	db *sqlx.DB
}

func NewSettingsHandler(db *sqlx.DB) *SettingsHandler {
	return &SettingsHandler{db: db}
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var settings model.UserSettings
	err := h.db.Get(&settings, "SELECT * FROM user_settings WHERE user_id = ?", user.ID)
	if err != nil {
		// Return defaults if no settings exist
		settings = model.UserSettings{
			UserID:        user.ID,
			Theme:         "system",
			Units:         "imperial",
			SpeciesFilter: "all",
		}
	}

	jsonResponse(w, http.StatusOK, settings)
}

type updateSettingsRequest struct {
	Theme         string `json:"theme"`
	Units         string `json:"units"`
	SpeciesFilter string `json:"species_filter"`
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Validate theme
	switch req.Theme {
	case "light", "dark", "system":
	default:
		req.Theme = "system"
	}

	// Validate units
	switch req.Units {
	case "imperial", "metric":
	default:
		req.Units = "imperial"
	}

	// Validate species filter
	switch req.SpeciesFilter {
	case "all", "freshwater", "saltwater":
	default:
		req.SpeciesFilter = "all"
	}

	_, err := h.db.Exec(`INSERT INTO user_settings (user_id, theme, units, species_filter, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET theme=?, units=?, species_filter=?, updated_at=CURRENT_TIMESTAMP`,
		user.ID, req.Theme, req.Units, req.SpeciesFilter, req.Theme, req.Units, req.SpeciesFilter)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to update settings")
		return
	}

	var settings model.UserSettings
	h.db.Get(&settings, "SELECT * FROM user_settings WHERE user_id = ?", user.ID)

	jsonResponse(w, http.StatusOK, settings)
}
