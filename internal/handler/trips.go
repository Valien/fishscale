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
)

type TripHandler struct {
	db *sqlx.DB
}

func NewTripHandler(db *sqlx.DB) *TripHandler {
	return &TripHandler{db: db}
}

func (h *TripHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var trips []model.Trip
	if err := h.db.SelectContext(r.Context(), &trips, "SELECT * FROM trips WHERE user_id = ? ORDER BY started_at DESC", user.ID); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query trips")
		return
	}

	if trips == nil {
		trips = []model.Trip{}
	}

	jsonResponse(w, http.StatusOK, trips)
}

func (h *TripHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	var trip model.Trip
	if err := h.db.GetContext(r.Context(), &trip, "SELECT * FROM trips WHERE id = ? AND user_id = ?", id, user.ID); err != nil {
		jsonError(w, http.StatusNotFound, "trip not found")
		return
	}

	var catches []model.Catch
	h.db.SelectContext(r.Context(), &catches, `SELECT c.*, COALESCE(s.name, '') as species_name
		FROM catches c
		LEFT JOIN species s ON c.species_id = s.id
		WHERE c.trip_id = ? ORDER BY c.caught_at DESC`, id)
	trip.Catches = catches

	jsonResponse(w, http.StatusOK, trip)
}

type createTripRequest struct {
	Name      string `json:"name"`
	StartedAt string `json:"started_at"`
	Notes     string `json:"notes"`
}

func (h *TripHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	startedAt := time.Now()
	if req.StartedAt != "" {
		var err error
		startedAt, err = time.Parse(time.RFC3339, req.StartedAt)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid started_at format")
			return
		}
	}

	result, err := h.db.ExecContext(r.Context(), "INSERT INTO trips (user_id, name, started_at, notes) VALUES (?, ?, ?, ?)",
		user.ID, req.Name, startedAt, req.Notes)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to create trip")
		return
	}

	id, _ := result.LastInsertId()
	var trip model.Trip
	h.db.GetContext(r.Context(), &trip, "SELECT * FROM trips WHERE id = ?", id)

	jsonResponse(w, http.StatusCreated, trip)
}

type updateTripRequest struct {
	Name    string  `json:"name"`
	EndedAt *string `json:"ended_at"`
	Notes   string  `json:"notes"`
}

func (h *TripHandler) Update(w http.ResponseWriter, r *http.Request) {
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
	if err := h.db.GetContext(r.Context(), &exists, "SELECT 1 FROM trips WHERE id = ? AND user_id = ?", id, user.ID); err != nil {
		jsonError(w, http.StatusNotFound, "trip not found")
		return
	}

	var req updateTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	var endedAt *time.Time
	if req.EndedAt != nil {
		t, err := time.Parse(time.RFC3339, *req.EndedAt)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid ended_at format")
			return
		}
		endedAt = &t
	}

	_, err = h.db.ExecContext(r.Context(), "UPDATE trips SET name=?, ended_at=?, notes=? WHERE id=? AND user_id=?",
		req.Name, endedAt, req.Notes, id, user.ID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to update trip")
		return
	}

	var trip model.Trip
	h.db.GetContext(r.Context(), &trip, "SELECT * FROM trips WHERE id = ?", id)

	jsonResponse(w, http.StatusOK, trip)
}

func (h *TripHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	// Unlink catches from this trip (don't delete them)
	h.db.ExecContext(r.Context(), "UPDATE catches SET trip_id = NULL WHERE trip_id = ? AND user_id = ?", id, user.ID)

	result, err := h.db.ExecContext(r.Context(), "DELETE FROM trips WHERE id = ? AND user_id = ?", id, user.ID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to delete trip")
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		jsonError(w, http.StatusNotFound, "trip not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
