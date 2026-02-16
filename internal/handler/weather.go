package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/allen/fishscale/internal/middleware"
)

type WeatherHandler struct {
	client *http.Client
}

func NewWeatherHandler() *WeatherHandler {
	return &WeatherHandler{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type weatherResponse struct {
	AirTempF    float64 `json:"air_temp_f"`
	WindMph     float64 `json:"wind_mph"`
	WindDir     string  `json:"wind_dir"`
	Conditions  string  `json:"conditions"`
	PressureMb  float64 `json:"pressure_mb"`
	HumidityPct float64 `json:"humidity_pct"`
}

func (h *WeatherHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	lat, err := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid lat parameter")
		return
	}
	lon, err := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid lon parameter")
		return
	}

	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,wind_speed_10m,wind_direction_10m,relative_humidity_2m,surface_pressure,weather_code&temperature_unit=fahrenheit&wind_speed_unit=mph",
		lat, lon,
	)

	resp, err := h.client.Get(url)
	if err != nil {
		jsonError(w, http.StatusBadGateway, "failed to fetch weather")
		return
	}
	defer resp.Body.Close()

	var raw struct {
		Current struct {
			Temperature2m    float64 `json:"temperature_2m"`
			WindSpeed10m     float64 `json:"wind_speed_10m"`
			WindDirection10m float64 `json:"wind_direction_10m"`
			Humidity2m       float64 `json:"relative_humidity_2m"`
			SurfacePressure  float64 `json:"surface_pressure"`
			WeatherCode      int     `json:"weather_code"`
		} `json:"current"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		jsonError(w, http.StatusBadGateway, "failed to parse weather response")
		return
	}

	result := weatherResponse{
		AirTempF:    raw.Current.Temperature2m,
		WindMph:     raw.Current.WindSpeed10m,
		WindDir:     degreesToCardinal(raw.Current.WindDirection10m),
		Conditions:  wmoCodeToCondition(raw.Current.WeatherCode),
		PressureMb:  raw.Current.SurfacePressure,
		HumidityPct: raw.Current.Humidity2m,
	}

	jsonResponse(w, http.StatusOK, result)
}

func degreesToCardinal(deg float64) string {
	dirs := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	idx := int((deg + 11.25) / 22.5) % 16
	return dirs[idx]
}

func wmoCodeToCondition(code int) string {
	switch {
	case code == 0:
		return "Clear"
	case code == 1:
		return "Mostly Clear"
	case code == 2:
		return "Partly Cloudy"
	case code == 3:
		return "Overcast"
	case code == 45 || code == 48:
		return "Fog"
	case code >= 51 && code <= 57:
		return "Drizzle"
	case code >= 61 && code <= 67:
		return "Rain"
	case code >= 71 && code <= 77:
		return "Snow"
	case code >= 80 && code <= 82:
		return "Rain Showers"
	case code >= 85 && code <= 86:
		return "Snow Showers"
	case code >= 95 && code <= 99:
		return "Thunderstorm"
	default:
		return "Unknown"
	}
}
