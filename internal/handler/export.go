package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/model"
)

type ExportHandler struct {
	db *sqlx.DB
}

func NewExportHandler(db *sqlx.DB) *ExportHandler {
	return &ExportHandler{db: db}
}

func (h *ExportHandler) Export(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	var catches []model.Catch
	err := h.db.SelectContext(r.Context(), &catches, `SELECT * FROM catches WHERE user_id = ? ORDER BY caught_at DESC`, user.ID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to query catches")
		return
	}

	if catches == nil {
		catches = []model.Catch{}
	}

	timestamp := time.Now().Format("2006-01-02")

	switch format {
	case "csv":
		h.exportCSV(w, catches, timestamp)
	default:
		h.exportJSON(w, catches, timestamp)
	}
}

func (h *ExportHandler) exportJSON(w http.ResponseWriter, catches []model.Catch, timestamp string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=fishscale-export-%s.json", timestamp))
	_ = json.NewEncoder(w).Encode(catches)
}

func (h *ExportHandler) exportCSV(w http.ResponseWriter, catches []model.Catch, timestamp string) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=fishscale-export-%s.csv", timestamp))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	headers := []string{
		"ID", "Date", "Species", "Location", "Latitude", "Longitude",
		"Length (in)", "Weight (lb)", "Kept", "Bait/Lure",
		"Rod Setup", "Line Info", "Hook Size",
		"Air Temp (F)", "Wind (mph)", "Wind Dir", "Conditions",
		"Pressure (mb)", "Humidity (%)", "Water Temp (F)", "Water Clarity",
		"Notes",
	}
	_ = writer.Write(headers)

	for _, c := range catches {
		row := []string{
			fmt.Sprintf("%d", c.ID),
			c.CaughtAt.Format(time.RFC3339),
			c.SpeciesName,
			c.LocationName,
			optFloat(c.Latitude),
			optFloat(c.Longitude),
			optFloat(c.LengthIn),
			optFloat(c.WeightLb),
			fmt.Sprintf("%v", c.Kept),
			c.BaitOrLure,
			c.RodSetup,
			c.LineInfo,
			c.HookSize,
			optFloat(c.AirTempF),
			optFloat(c.WindMph),
			c.WindDir,
			c.Conditions,
			optFloat(c.PressureMb),
			optFloat(c.HumidityPct),
			optFloat(c.WaterTempF),
			c.WaterClarity,
			c.Notes,
		}
		_ = writer.Write(row)
	}
}

func optFloat(f *float64) string {
	if f == nil {
		return ""
	}
	return fmt.Sprintf("%.2f", *f)
}
