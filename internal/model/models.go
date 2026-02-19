package model

import "time"

type User struct {
	ID          int64     `db:"id" json:"id"`
	TailscaleID string    `db:"tailscale_id" json:"tailscale_id"`
	DisplayName string    `db:"display_name" json:"display_name"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// TailscaleInfo holds transient identity data from the Tailscale WhoIs response.
// Not persisted — fresh on every request.
type TailscaleInfo struct {
	LoginName     string `json:"login_name"`
	DisplayName   string `json:"display_name"`
	TailscaleID   string `json:"tailscale_id"`
	NodeName      string `json:"node_name"`
	ProfilePicURL string `json:"profile_pic_url,omitempty"`
}

type Trip struct {
	ID        int64      `db:"id" json:"id"`
	UserID    int64      `db:"user_id" json:"user_id"`
	Name      string     `db:"name" json:"name"`
	StartedAt *time.Time `db:"started_at" json:"started_at"`
	EndedAt   *time.Time `db:"ended_at" json:"ended_at"`
	Notes     string     `db:"notes" json:"notes"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	Catches   []Catch    `db:"-" json:"catches,omitempty"`
}

type Catch struct {
	ID           int64     `db:"id" json:"id"`
	UserID       int64     `db:"user_id" json:"user_id"`
	TripID       *int64    `db:"trip_id" json:"trip_id"`
	SpeciesName  string    `db:"species_name" json:"species_name"`
	CaughtAt     time.Time `db:"caught_at" json:"caught_at"`
	Latitude     *float64  `db:"latitude" json:"latitude"`
	Longitude    *float64  `db:"longitude" json:"longitude"`
	LocationName string    `db:"location_name" json:"location_name"`
	LengthIn     *float64  `db:"length_in" json:"length_in"`
	WeightLb     *float64  `db:"weight_lb" json:"weight_lb"`
	Kept         bool      `db:"kept" json:"kept"`
	BaitOrLure   string    `db:"bait_or_lure" json:"bait_or_lure"`
	RodSetup     string    `db:"rod_setup" json:"rod_setup"`
	LineInfo     string    `db:"line_info" json:"line_info"`
	HookSize     string    `db:"hook_size" json:"hook_size"`
	AirTempF     *float64  `db:"air_temp_f" json:"air_temp_f"`
	WindMph      *float64  `db:"wind_mph" json:"wind_mph"`
	WindDir      string    `db:"wind_dir" json:"wind_dir"`
	Conditions   string    `db:"conditions" json:"conditions"`
	PressureMb   *float64  `db:"pressure_mb" json:"pressure_mb"`
	HumidityPct  *float64  `db:"humidity_pct" json:"humidity_pct"`
	WaterTempF   *float64  `db:"water_temp_f" json:"water_temp_f"`
	WaterClarity string    `db:"water_clarity" json:"water_clarity"`
	Notes        string    `db:"notes" json:"notes"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
	Photos       []Photo   `db:"-" json:"photos,omitempty"`
}

type Photo struct {
	ID        int64     `db:"id" json:"id"`
	CatchID   int64     `db:"catch_id" json:"catch_id"`
	Filename  string    `db:"filename" json:"filename"`
	Thumbnail string    `db:"thumbnail" json:"thumbnail"`
	SortOrder int       `db:"sort_order" json:"sort_order"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	URL       string    `db:"-" json:"url"`
}

type UserSettings struct {
	ID        int64     `db:"id" json:"id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	Theme     string    `db:"theme" json:"theme"`
	Units     string    `db:"units" json:"units"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type CreateCatchRequest struct {
	TripID       *int64   `json:"trip_id"`
	SpeciesName  string   `json:"species_name"`
	CaughtAt     string   `json:"caught_at"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	LocationName string   `json:"location_name"`
	LengthIn     *float64 `json:"length_in"`
	WeightLb     *float64 `json:"weight_lb"`
	Kept         bool     `json:"kept"`
	BaitOrLure   string   `json:"bait_or_lure"`
	RodSetup     string   `json:"rod_setup"`
	LineInfo     string   `json:"line_info"`
	HookSize     string   `json:"hook_size"`
	AirTempF     *float64 `json:"air_temp_f"`
	WindMph      *float64 `json:"wind_mph"`
	WindDir      string   `json:"wind_dir"`
	Conditions   string   `json:"conditions"`
	PressureMb   *float64 `json:"pressure_mb"`
	HumidityPct  *float64 `json:"humidity_pct"`
	WaterTempF   *float64 `json:"water_temp_f"`
	WaterClarity string   `json:"water_clarity"`
	Notes        string   `json:"notes"`
}

type StatsResponse struct {
	TotalCatches  int            `json:"total_catches"`
	TotalSpecies  int            `json:"total_species"`
	TotalTrips    int            `json:"total_trips"`
	SpeciesCounts []SpeciesCount `json:"species_counts"`
	PersonalBests []PersonalBest `json:"personal_bests"`
	BaitCounts    []BaitCount    `json:"bait_counts"`
	MonthlyCounts []MonthCount   `json:"monthly_counts"`
}

type SpeciesCount struct {
	SpeciesName string `db:"species_name" json:"species_name"`
	Count       int    `db:"count" json:"count"`
}

type PersonalBest struct {
	SpeciesName string  `db:"species_name" json:"species_name"`
	MaxWeightLb float64 `db:"max_weight_lb" json:"max_weight_lb"`
	MaxLengthIn float64 `db:"max_length_in" json:"max_length_in"`
}

type BaitCount struct {
	BaitOrLure string `db:"bait_or_lure" json:"bait_or_lure"`
	Count      int    `db:"count" json:"count"`
}

type MonthCount struct {
	Month string `db:"month" json:"month"`
	Count int    `db:"count" json:"count"`
}
