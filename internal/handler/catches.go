package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/model"
	"github.com/allen/fishscale/internal/storage"
)

type CatchHandler struct {
	db    *sqlx.DB
	store storage.Store
}

func NewCatchHandler(db *sqlx.DB, store storage.Store) *CatchHandler {
	return &CatchHandler{db: db, store: store}
}

func (h *CatchHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	query := `SELECT c.*, COALESCE(s.name, '') as species_name
		FROM catches c
		LEFT JOIN species s ON c.species_id = s.id
		WHERE c.user_id = ?
		ORDER BY c.caught_at DESC`

	var catches []model.Catch
	if err := h.db.SelectContext(r.Context(), &catches, query, user.ID); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query catches")
		return
	}

	if catches == nil {
		catches = []model.Catch{}
	}

	jsonResponse(w, http.StatusOK, catches)
}

func (h *CatchHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var catch model.Catch
	query := `SELECT c.*, COALESCE(s.name, '') as species_name
		FROM catches c
		LEFT JOIN species s ON c.species_id = s.id
		WHERE c.id = ? AND c.user_id = ?`
	if err := h.db.GetContext(r.Context(), &catch, query, id, user.ID); err != nil {
		jsonError(w, http.StatusNotFound, "catch not found")
		return
	}

	var photos []model.Photo
	if err := h.db.SelectContext(r.Context(), &photos, "SELECT * FROM photos WHERE catch_id = ? ORDER BY sort_order", id); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query photos")
		return
	}
	catch.Photos = photos

	jsonResponse(w, http.StatusOK, catch)
}

func (h *CatchHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req model.CreateCatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	caughtAt, err := time.Parse(time.RFC3339, req.CaughtAt)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid caught_at format, use RFC3339")
		return
	}

	if err := validateCatchRequest(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.db.ExecContext(r.Context(), `INSERT INTO catches (
		user_id, trip_id, species_id, caught_at, latitude, longitude, location_name,
		length_in, weight_lb, kept, bait_or_lure, rod_setup, line_info, hook_size,
		air_temp_f, wind_mph, wind_dir, conditions, pressure_mb, humidity_pct,
		water_temp_f, water_clarity, notes
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, req.TripID, req.SpeciesID, caughtAt, req.Latitude, req.Longitude, req.LocationName,
		req.LengthIn, req.WeightLb, req.Kept, req.BaitOrLure, req.RodSetup, req.LineInfo, req.HookSize,
		req.AirTempF, req.WindMph, req.WindDir, req.Conditions, req.PressureMb, req.HumidityPct,
		req.WaterTempF, req.WaterClarity, req.Notes,
	)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to create catch")
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to get created ID")
		return
	}

	var catch model.Catch
	if err := h.db.GetContext(r.Context(), &catch, `SELECT c.*, COALESCE(s.name, '') as species_name
		FROM catches c
		LEFT JOIN species s ON c.species_id = s.id
		WHERE c.id = ?`, id); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to fetch created catch")
		return
	}

	jsonResponse(w, http.StatusCreated, catch)
}

func (h *CatchHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var exists int
	if err := h.db.GetContext(r.Context(), &exists, "SELECT 1 FROM catches WHERE id = ? AND user_id = ?", id, user.ID); err != nil {
		jsonError(w, http.StatusNotFound, "catch not found")
		return
	}

	var req model.CreateCatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	caughtAt, err := time.Parse(time.RFC3339, req.CaughtAt)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid caught_at format")
		return
	}

	if err := validateCatchRequest(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	_, err = h.db.ExecContext(r.Context(), `UPDATE catches SET
		trip_id=?, species_id=?, caught_at=?, latitude=?, longitude=?, location_name=?,
		length_in=?, weight_lb=?, kept=?, bait_or_lure=?, rod_setup=?, line_info=?, hook_size=?,
		air_temp_f=?, wind_mph=?, wind_dir=?, conditions=?, pressure_mb=?, humidity_pct=?,
		water_temp_f=?, water_clarity=?, notes=?, updated_at=CURRENT_TIMESTAMP
		WHERE id=? AND user_id=?`,
		req.TripID, req.SpeciesID, caughtAt, req.Latitude, req.Longitude, req.LocationName,
		req.LengthIn, req.WeightLb, req.Kept, req.BaitOrLure, req.RodSetup, req.LineInfo, req.HookSize,
		req.AirTempF, req.WindMph, req.WindDir, req.Conditions, req.PressureMb, req.HumidityPct,
		req.WaterTempF, req.WaterClarity, req.Notes,
		id, user.ID,
	)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to update catch")
		return
	}

	var catch model.Catch
	if err := h.db.GetContext(r.Context(), &catch, `SELECT c.*, COALESCE(s.name, '') as species_name
		FROM catches c
		LEFT JOIN species s ON c.species_id = s.id
		WHERE c.id = ?`, id); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to fetch updated catch")
		return
	}

	jsonResponse(w, http.StatusOK, catch)
}

func (h *CatchHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var photos []model.Photo
	if err := h.db.SelectContext(r.Context(), &photos, "SELECT * FROM photos WHERE catch_id = ?", id); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query photos")
		return
	}

	result, err := h.db.ExecContext(r.Context(), "DELETE FROM catches WHERE id = ? AND user_id = ?", id, user.ID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to delete")
		return
	}

	rows, err := result.RowsAffected()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to confirm deletion")
		return
	}
	if rows == 0 {
		jsonError(w, http.StatusNotFound, "catch not found")
		return
	}

	for _, p := range photos {
		h.store.Delete(p.Filename)
	}

	w.WriteHeader(http.StatusNoContent)
}
