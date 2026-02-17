package handler

import (
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/allen/fishscale/internal/middleware"
)

type AutocompleteHandler struct {
	db *sqlx.DB
}

func NewAutocompleteHandler(db *sqlx.DB) *AutocompleteHandler {
	return &AutocompleteHandler{db: db}
}

// Species returns a frequency-sorted list of unique species names from the user's catch history.
func (h *AutocompleteHandler) Species(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var species []string
	err := h.db.SelectContext(r.Context(), &species, `
		SELECT species_name
		FROM catches
		WHERE user_id = ? AND species_name != ''
		GROUP BY species_name
		ORDER BY COUNT(*) DESC
		LIMIT 50`, user.ID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query species")
		return
	}

	if species == nil {
		species = []string{}
	}

	jsonResponse(w, http.StatusOK, species)
}
