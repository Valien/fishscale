# Map Location Picker Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow users to pick a catch location by tapping on a full-screen map overlay, instead of relying solely on GPS auto-detection.

**Architecture:** New standalone `LocationPicker.svelte` component renders a full-screen MapLibre overlay triggered from the LogCatch form. Tap-to-place pin, confirm button returns coordinates via callback. No backend changes needed.

**Tech Stack:** Svelte 5, MapLibre GL JS, TypeScript

**Design Doc:** `docs/plans/2026-02-18-map-location-picker-design.md`

---

### Task 1: Create LocationPicker component

**Files:**
- Create: `frontend/src/lib/components/LocationPicker.svelte`

**Step 1: Create the component file**

Create `frontend/src/lib/components/LocationPicker.svelte` with the full implementation:

```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import maplibregl from 'maplibre-gl';
  import 'maplibre-gl/dist/maplibre-gl.css';

  let {
    initialLat = null,
    initialLng = null,
    onSelect,
    onCancel,
  }: {
    initialLat?: number | null;
    initialLng?: number | null;
    onSelect: (coords: { latitude: number; longitude: number }) => void;
    onCancel: () => void;
  } = $props();

  let mapContainer: HTMLDivElement;
  let map: maplibregl.Map | null = null;
  let marker: maplibregl.Marker | null = null;
  let pinLat = $state<number | null>(null);
  let pinLng = $state<number | null>(null);

  onMount(() => {
    const center: [number, number] =
      initialLat && initialLng ? [initialLng, initialLat] : [-98.5, 39.8];
    const zoom = initialLat && initialLng ? 12 : 3;

    map = new maplibregl.Map({
      container: mapContainer,
      style: {
        version: 8,
        sources: {
          osm: {
            type: 'raster',
            tiles: ['https://tile.openstreetmap.org/{z}/{x}/{y}.png'],
            tileSize: 256,
            attribution: '&copy; OpenStreetMap contributors',
          },
        },
        layers: [{ id: 'osm', type: 'raster', source: 'osm' }],
      },
      center,
      zoom,
    });

    map.addControl(new maplibregl.NavigationControl(), 'top-right');

    // Place initial marker if coordinates were provided
    if (initialLat && initialLng) {
      marker = new maplibregl.Marker({ color: '#dc3545' })
        .setLngLat([initialLng, initialLat])
        .addTo(map);
      pinLat = initialLat;
      pinLng = initialLng;
    }

    // Tap to place/move pin
    map.on('click', (e) => {
      const { lng, lat } = e.lngLat;
      pinLat = lat;
      pinLng = lng;

      if (marker) {
        marker.setLngLat([lng, lat]);
      } else if (map) {
        marker = new maplibregl.Marker({ color: '#dc3545' })
          .setLngLat([lng, lat])
          .addTo(map);
      }
    });

    return () => {
      marker?.remove();
      map?.remove();
      map = null;
    };
  });

  function confirm() {
    if (pinLat !== null && pinLng !== null) {
      onSelect({ latitude: pinLat, longitude: pinLng });
    }
  }
</script>

<div class="picker-overlay">
  <div class="picker-header">
    <button class="picker-btn" type="button" onclick={onCancel}>Cancel</button>
    <span class="picker-coords">
      {#if pinLat !== null && pinLng !== null}
        {pinLat.toFixed(4)}, {pinLng.toFixed(4)}
      {:else}
        Tap to place pin
      {/if}
    </span>
  </div>

  <div class="picker-map" bind:this={mapContainer}></div>

  <div class="picker-footer">
    <button
      class="btn btn-primary btn-block"
      type="button"
      onclick={confirm}
      disabled={pinLat === null}
    >
      Confirm Location
    </button>
  </div>
</div>

<style>
  .picker-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 100;
    display: flex;
    flex-direction: column;
    background: var(--bg);
  }

  .picker-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    background: var(--card-bg);
    border-bottom: 1px solid var(--card-border);
    z-index: 1;
  }

  .picker-btn {
    background: none;
    border: none;
    color: var(--primary);
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    padding: 4px 0;
  }

  .picker-coords {
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  .picker-map {
    flex: 1;
    width: 100%;
  }

  .picker-footer {
    padding: 12px 16px;
    background: var(--card-bg);
    border-top: 1px solid var(--card-border);
  }
</style>
```

**Step 2: Verify it compiles**

Run: `cd frontend && npm run build`
Expected: Build succeeds with no new errors.

**Step 3: Commit**

```bash
git add frontend/src/lib/components/LocationPicker.svelte
git commit -m "feat: add LocationPicker map overlay component

Standalone full-screen MapLibre map for picking catch locations.
Tap to place/move pin, confirm to return coordinates.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 2: Integrate LocationPicker into LogCatch

**Files:**
- Modify: `frontend/src/lib/pages/LogCatch.svelte`

**Step 1: Add import and state**

At the top of the `<script>` block in `LogCatch.svelte`, add the import (after the existing imports):

```typescript
import LocationPicker from '../components/LocationPicker.svelte';
```

Add a new state variable (after the existing `$state` declarations around line 37):

```typescript
let showLocationPicker = $state(false);
```

**Step 2: Add the "Pick on Map" button and picker**

Replace the location form-group section (lines 277-289 in `LogCatch.svelte`):

```svelte
      <div class="form-group">
        <label>Location</label>
        <input
          type="text"
          placeholder="e.g. Lake Fork, boat ramp cove"
          bind:value={form.location_name}
        />
        {#if form.latitude}
          <small class="coords">{form.latitude.toFixed(4)}, {form.longitude?.toFixed(4)}</small>
        {:else}
          <small class="coords">Getting GPS location...</small>
        {/if}
      </div>
```

With:

```svelte
      <div class="form-group">
        <label>Location</label>
        <input
          type="text"
          placeholder="e.g. Lake Fork, boat ramp cove"
          bind:value={form.location_name}
        />
        <div class="location-row">
          {#if form.latitude}
            <small class="coords">{form.latitude.toFixed(4)}, {form.longitude?.toFixed(4)}</small>
          {:else}
            <small class="coords">Getting GPS location...</small>
          {/if}
          <button
            class="pick-map-btn"
            type="button"
            onclick={() => (showLocationPicker = true)}
          >
            Pick on Map
          </button>
        </div>
      </div>
```

**Step 3: Add the LocationPicker overlay**

At the end of `LogCatch.svelte`, just before the closing `</div>` of the page div (before `</div>` on the last line of the template, around line 421), add:

```svelte
  {#if showLocationPicker}
    <LocationPicker
      initialLat={form.latitude}
      initialLng={form.longitude}
      onSelect={(coords) => {
        form.latitude = coords.latitude;
        form.longitude = coords.longitude;
        showLocationPicker = false;
      }}
      onCancel={() => (showLocationPicker = false)}
    />
  {/if}
```

**Step 4: Add styles**

Add these styles to the `<style>` block in `LogCatch.svelte`:

```css
  .location-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: 4px;
  }

  .pick-map-btn {
    background: none;
    border: none;
    color: var(--primary);
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    padding: 2px 0;
    white-space: nowrap;
  }
```

**Step 5: Build and verify**

Run: `cd frontend && npm run build`
Expected: Build succeeds.

Run: `cd frontend && npm run check`
Expected: No new errors (pre-existing warnings are OK).

**Step 6: Commit**

```bash
git add frontend/src/lib/pages/LogCatch.svelte
git commit -m "feat: integrate map location picker into LogCatch form

Adds 'Pick on Map' button next to GPS coordinates. Opens full-screen
map overlay to tap and select a location. Works in both create and
edit modes. Closes #1.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 3: Build frontend dist and manual test

**Files:**
- Modify: `internal/frontend/dist/` (rebuild)

**Step 1: Build and copy frontend dist**

```bash
cd frontend && npm run build && cd ..
rm -rf internal/frontend/dist && cp -r frontend/dist internal/frontend/dist
```

**Step 2: Run CI checks**

```bash
make ci
```

Expected: All checks pass (golangci-lint step may fail per CLAUDE.md — that's OK).

**Step 3: Manual test checklist**

Start dev server:
```bash
FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=./fish.db FISHSCALE_PHOTO_DIR=./photos GOWORK=off go run ./cmd/fishscale
```

Test these scenarios:
1. **Create mode — GPS then override:** Open Log Catch, wait for GPS coords to appear, tap "Pick on Map", tap a different location on the map, confirm. Verify the coords update in the form.
2. **Create mode — pick without GPS:** If GPS is slow/denied, tap "Pick on Map" immediately. Map should show US center view. Tap to place pin, confirm. Coords should appear in the form.
3. **Create mode — cancel:** Open picker, tap cancel. Coords should remain unchanged.
4. **Edit mode:** Edit an existing catch. The "Pick on Map" button should be present. Tap it — map should center on the existing catch coordinates with a pre-placed pin. Move the pin, confirm. Verify coords update.
5. **Dark mode:** Toggle dark theme in settings. Verify picker header/footer use theme colors correctly.

**Step 4: Commit dist**

```bash
git add internal/frontend/dist
git commit -m "build: update frontend dist with map location picker

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 4: Update docs

**Files:**
- Modify: `docs/plans/todo.md`
- Modify: `docs/plans/2026-02-18-map-location-picker-design.md`

**Step 1: Mark backlog item complete**

In `docs/plans/todo.md`, move the map location picker item from Ideas & Enhancements to Completed:

Remove from Ideas & Enhancements:
```
- [ ] Map-based location picker for logging catches ([GH#1](https://github.com/Valien/fishscale/issues/1)) — scroll/tap on map to drop a pin instead of relying on GPS auto-location; supports logging catches after the fact from home
```

Add to Completed:
```
- [x] ~~Map-based location picker for logging catches~~ ([GH#1](https://github.com/Valien/fishscale/issues/1), Completed 2026-02-18: full-screen map overlay with tap-to-place pin in LogCatch, see 2026-02-18-map-location-picker-design.md)
```

**Step 2: Update design doc status**

In `docs/plans/2026-02-18-map-location-picker-design.md`, change:
```
**Status:** Approved
```
To:
```
**Status:** Implemented
```

**Step 3: Commit**

```bash
git add docs/plans/todo.md docs/plans/2026-02-18-map-location-picker-design.md
git commit -m "docs: mark map location picker as implemented

Co-Authored-By: Claude <noreply@anthropic.com>"
```
