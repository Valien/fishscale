# Map Location Picker Design

**Date:** 2026-02-18
**GitHub Issue:** [#1](https://github.com/Valien/fishscale/issues/1)
**Status:** Implemented

## Problem

Users can only set catch location via GPS auto-detection on the LogCatch page. This fails when logging catches after the fact (e.g., from home), since GPS captures the current position, not where the fish was caught. Users need a way to manually pick a location on a map.

## Design

### Approach: Standalone LocationPicker component

A new `LocationPicker.svelte` component renders a full-screen MapLibre map overlay. LogCatch opens it via a "Pick on Map" button next to the Location field. The picker has its own map instance, a confirm button, and passes coordinates back via a callback prop.

This was chosen over sharing a map instance with MapView or embedding an inline mini-map. A standalone component keeps things decoupled — the picker's needs (single pin, tap-to-place) are unrelated to MapView's marker sync logic.

### Component: LocationPicker.svelte

- Full-screen overlay positioned above the form, below bottom nav
- MapLibre map with OSM raster tiles (same tile config as MapView)
- On open: if coordinates already exist in the form, center the map there with a pre-placed marker; otherwise center on a sensible default (US center or last known location)
- Tap anywhere on the map to place or move a single pin marker
- Top bar displays "Cancel" button (left) and current pin coordinates (right)
- Bottom "Confirm Location" button, disabled until a pin is placed
- On confirm: calls `onSelect({latitude: number, longitude: number})` callback, overlay closes
- On cancel: closes with no changes

### Integration in LogCatch.svelte

- Add a "Pick on Map" button below the Location text input, next to the GPS coords display
- When coordinates come back from the picker, set `form.latitude` and `form.longitude`
- GPS auto-location still runs on mount as the default; the map picker overrides it
- No weather fetch on map-picked coordinates (weather auto-fetch only applies to GPS)
- Works in both create and edit modes

### What doesn't change

- No backend changes. The lat/lng fields already exist in the data model.
- No changes to MapView.svelte or the catches store.
- No changes to the API client.

## Files

| File | Action |
|------|--------|
| `frontend/src/lib/components/LocationPicker.svelte` | Create |
| `frontend/src/lib/pages/LogCatch.svelte` | Modify (add button + picker integration) |
