package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/model"
	"github.com/allen/fishscale/internal/storage"
)

type PhotoHandler struct {
	db    *sqlx.DB
	store storage.Store
}

func NewPhotoHandler(db *sqlx.DB, store storage.Store) *PhotoHandler {
	return &PhotoHandler{db: db, store: store}
}

func (h *PhotoHandler) Add(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	catchID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid catch id")
		return
	}

	// Verify catch ownership
	var exists int
	if err := h.db.GetContext(r.Context(), &exists, "SELECT 1 FROM catches WHERE id = ? AND user_id = ?", catchID, user.ID); err != nil {
		jsonError(w, http.StatusNotFound, "catch not found")
		return
	}

	// Parse multipart form (32MB max)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, http.StatusBadRequest, "failed to parse form")
		return
	}

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		jsonError(w, http.StatusBadRequest, "no photos provided")
		return
	}

	var photos []model.Photo
	for i, fh := range files {
		f, err := fh.Open()
		if err != nil {
			continue
		}

		path, err := h.store.Save(fh.Filename, f)
		f.Close()
		if err != nil {
			continue
		}

		result, err := h.db.ExecContext(r.Context(),
			"INSERT INTO photos (catch_id, filename, sort_order) VALUES (?, ?, ?)",
			catchID, path, i,
		)
		if err != nil {
			h.store.Delete(path)
			continue
		}

		id, _ := result.LastInsertId()
		photos = append(photos, model.Photo{
			ID:        id,
			CatchID:   catchID,
			Filename:  path,
			SortOrder: i,
		})
	}

	if len(photos) == 0 {
		jsonError(w, http.StatusInternalServerError, "failed to save any photos")
		return
	}

	jsonResponse(w, http.StatusCreated, photos)
}

func (h *PhotoHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	// Get photo and verify ownership through catch
	var photo model.Photo
	err = h.db.GetContext(r.Context(), &photo, `SELECT p.* FROM photos p
		JOIN catches c ON p.catch_id = c.id
		WHERE p.id = ? AND c.user_id = ?`, id, user.ID)
	if err != nil {
		jsonError(w, http.StatusNotFound, "photo not found")
		return
	}

	h.db.ExecContext(r.Context(), "DELETE FROM photos WHERE id = ?", id)
	h.store.Delete(photo.Filename)

	w.WriteHeader(http.StatusNoContent)
}
