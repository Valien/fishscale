package handler

import (
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/model"
)

type StatsHandler struct {
	db *sqlx.DB
}

func NewStatsHandler(db *sqlx.DB) *StatsHandler {
	return &StatsHandler{db: db}
}

func (h *StatsHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var stats model.StatsResponse

	if err := h.db.GetContext(r.Context(), &stats.TotalCatches, "SELECT COUNT(*) FROM catches WHERE user_id = ?", user.ID); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query stats")
		return
	}

	if err := h.db.GetContext(r.Context(), &stats.TotalSpecies, "SELECT COUNT(DISTINCT species_name) FROM catches WHERE user_id = ? AND species_name != ''", user.ID); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query stats")
		return
	}

	if err := h.db.GetContext(r.Context(), &stats.TotalTrips, "SELECT COUNT(*) FROM trips WHERE user_id = ?", user.ID); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query stats")
		return
	}

	if err := h.db.SelectContext(r.Context(), &stats.SpeciesCounts, `SELECT species_name, COUNT(*) as count
		FROM catches
		WHERE user_id = ? AND species_name != ''
		GROUP BY species_name ORDER BY count DESC LIMIT 5`, user.ID); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query stats")
		return
	}
	if stats.SpeciesCounts == nil {
		stats.SpeciesCounts = []model.SpeciesCount{}
	}

	if err := h.db.SelectContext(r.Context(), &stats.PersonalBests, `SELECT species_name,
		MAX(COALESCE(weight_lb, 0)) as max_weight_lb,
		MAX(COALESCE(length_in, 0)) as max_length_in
		FROM catches
		WHERE user_id = ? AND species_name != '' AND (weight_lb IS NOT NULL OR length_in IS NOT NULL)
		GROUP BY species_name ORDER BY max_weight_lb DESC LIMIT 10`, user.ID); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query stats")
		return
	}
	if stats.PersonalBests == nil {
		stats.PersonalBests = []model.PersonalBest{}
	}

	if err := h.db.SelectContext(r.Context(), &stats.BaitCounts, `SELECT bait_or_lure, COUNT(*) as count
		FROM catches WHERE user_id = ? AND bait_or_lure != ''
		GROUP BY bait_or_lure ORDER BY count DESC LIMIT 5`, user.ID); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query stats")
		return
	}
	if stats.BaitCounts == nil {
		stats.BaitCounts = []model.BaitCount{}
	}

	if err := h.db.SelectContext(r.Context(), &stats.MonthlyCounts, `SELECT strftime('%Y-%m', caught_at) as month, COUNT(*) as count
		FROM catches WHERE user_id = ? AND caught_at >= date('now', '-12 months')
			AND strftime('%Y-%m', caught_at) IS NOT NULL
		GROUP BY month ORDER BY month`, user.ID); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query stats")
		return
	}
	if stats.MonthlyCounts == nil {
		stats.MonthlyCounts = []model.MonthCount{}
	}

	jsonResponse(w, http.StatusOK, stats)
}
