package handler

import (
	"fmt"

	"github.com/allen/fishscale/internal/model"
)

const (
	maxTextFieldLen  = 2000
	maxShortFieldLen = 200
	maxNotesLen      = 5000
)

func validateStringLen(field, value string, max int) error {
	if len(value) > max {
		return fmt.Errorf("%s exceeds maximum length of %d characters", field, max)
	}
	return nil
}

func validateLatitude(lat *float64) error {
	if lat != nil && (*lat < -90 || *lat > 90) {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	return nil
}

func validateLongitude(lon *float64) error {
	if lon != nil && (*lon < -180 || *lon > 180) {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	return nil
}

func validatePositive(field string, val *float64) error {
	if val != nil && *val < 0 {
		return fmt.Errorf("%s must not be negative", field)
	}
	return nil
}

func validateCatchRequest(req *model.CreateCatchRequest) error {
	for _, check := range []struct {
		name string
		val  string
		max  int
	}{
		{"species_name", req.SpeciesName, maxShortFieldLen},
		{"location_name", req.LocationName, maxTextFieldLen},
		{"bait_or_lure", req.BaitOrLure, maxShortFieldLen},
		{"rod_setup", req.RodSetup, maxShortFieldLen},
		{"line_info", req.LineInfo, maxShortFieldLen},
		{"hook_size", req.HookSize, maxShortFieldLen},
		{"water_clarity", req.WaterClarity, maxShortFieldLen},
		{"wind_dir", req.WindDir, maxShortFieldLen},
		{"conditions", req.Conditions, maxShortFieldLen},
		{"notes", req.Notes, maxNotesLen},
	} {
		if err := validateStringLen(check.name, check.val, check.max); err != nil {
			return err
		}
	}

	if err := validateLatitude(req.Latitude); err != nil {
		return err
	}
	if err := validateLongitude(req.Longitude); err != nil {
		return err
	}

	for _, check := range []struct {
		name string
		val  *float64
	}{
		{"weight_lb", req.WeightLb},
		{"length_in", req.LengthIn},
	} {
		if err := validatePositive(check.name, check.val); err != nil {
			return err
		}
	}

	return nil
}
