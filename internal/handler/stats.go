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

	// Total catches
	h.db.GetContext(r.Context(), &stats.TotalCatches, "SELECT COUNT(*) FROM catches WHERE user_id = ?", user.ID)

	// Total species caught
	h.db.GetContext(r.Context(), &stats.TotalSpecies, "SELECT COUNT(DISTINCT species_id) FROM catches WHERE user_id = ? AND species_id IS NOT NULL", user.ID)

	// Total trips
	h.db.GetContext(r.Context(), &stats.TotalTrips, "SELECT COUNT(*) FROM trips WHERE user_id = ?", user.ID)

	// Top species by count
	h.db.SelectContext(r.Context(), &stats.SpeciesCounts, `SELECT s.id as species_id, s.name as species_name, COUNT(*) as count
		FROM catches c JOIN species s ON c.species_id = s.id
		WHERE c.user_id = ?
		GROUP BY s.id ORDER BY count DESC LIMIT 5`, user.ID)
	if stats.SpeciesCounts == nil {
		stats.SpeciesCounts = []model.SpeciesCount{}
	}

	// Personal bests (heaviest per species)
	h.db.SelectContext(r.Context(), &stats.PersonalBests, `SELECT s.id as species_id, s.name as species_name,
		MAX(COALESCE(c.weight_lb, 0)) as max_weight_lb,
		MAX(COALESCE(c.length_in, 0)) as max_length_in
		FROM catches c JOIN species s ON c.species_id = s.id
		WHERE c.user_id = ? AND (c.weight_lb IS NOT NULL OR c.length_in IS NOT NULL)
		GROUP BY s.id ORDER BY max_weight_lb DESC LIMIT 10`, user.ID)
	if stats.PersonalBests == nil {
		stats.PersonalBests = []model.PersonalBest{}
	}

	// Top baits
	h.db.SelectContext(r.Context(), &stats.BaitCounts, `SELECT bait_or_lure, COUNT(*) as count
		FROM catches WHERE user_id = ? AND bait_or_lure != ''
		GROUP BY bait_or_lure ORDER BY count DESC LIMIT 5`, user.ID)
	if stats.BaitCounts == nil {
		stats.BaitCounts = []model.BaitCount{}
	}

	// Catches by month (last 12 months)
	h.db.SelectContext(r.Context(), &stats.MonthlyCounts, `SELECT strftime('%Y-%m', caught_at) as month, COUNT(*) as count
		FROM catches WHERE user_id = ? AND caught_at >= date('now', '-12 months')
		GROUP BY month ORDER BY month`, user.ID)
	if stats.MonthlyCounts == nil {
		stats.MonthlyCounts = []model.MonthCount{}
	}

	jsonResponse(w, http.StatusOK, stats)
}
