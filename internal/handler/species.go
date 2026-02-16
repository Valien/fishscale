package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/model"
)

type SpeciesHandler struct {
	db *sqlx.DB
}

func NewSpeciesHandler(db *sqlx.DB) *SpeciesHandler {
	return &SpeciesHandler{db: db}
}

func (h *SpeciesHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	q := r.URL.Query().Get("q")

	var species []model.Species
	var err error
	if q != "" {
		err = h.db.Select(&species, "SELECT * FROM species WHERE name LIKE ? ORDER BY name", "%"+q+"%")
	} else {
		err = h.db.Select(&species, "SELECT * FROM species ORDER BY name")
	}

	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query species")
		return
	}

	if species == nil {
		species = []model.Species{}
	}

	jsonResponse(w, http.StatusOK, species)
}

type createSpeciesRequest struct {
	Name     string `json:"name"`
	Category string `json:"category"`
}

func (h *SpeciesHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createSpeciesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Name == "" {
		jsonError(w, http.StatusBadRequest, "name is required")
		return
	}

	result, err := h.db.Exec("INSERT INTO species (name, category) VALUES (?, ?)", req.Name, req.Category)
	if err != nil {
		jsonError(w, http.StatusConflict, "species already exists")
		return
	}

	id, _ := result.LastInsertId()
	var species model.Species
	h.db.Get(&species, "SELECT * FROM species WHERE id = ?", id)

	jsonResponse(w, http.StatusCreated, species)
}
