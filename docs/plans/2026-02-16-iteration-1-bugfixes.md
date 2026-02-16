# Fishscale Iteration 1: Bugfixes and Photo Upload

**Date:** 2026-02-16
**Status:** Pending

## Overview

Fixes three related iOS Safari bugs with the species dropdown, a map re-initialization issue, and adds photo upload/camera capture to the Log Catch form.

---

## Bug 1-3: Species Dropdown Freezes iOS Safari

**Symptoms:**
- Tapping a species name in the dropdown does not fill the form field
- The dropdown stays open and freezes the page on iOS Safari
- Cancel button and rest of UI become unresponsive
- Only recovery is a full page refresh
- Backend does receive the catch (save works), but the UI is stuck

**Root Causes (discovered across multiple iterations):**
1. `onclick` on dynamically-shown elements inside absolute-positioned containers is unreliable on iOS Safari
2. The `$effect` that filters species re-runs when `speciesQuery` changes, which can re-trigger `showSpeciesDropdown = true` immediately after `selectSpecies` sets it to `false` (race condition)
3. First attempted fix (backdrop overlay with `onpointerup`) failed: a `position: fixed` backdrop inside a `position: relative` container creates broken z-index stacking on Safari — the backdrop blocks the entire viewport (including Save/Cancel buttons) while the dropdown items' `onpointerup` events don't fire reliably through the stacking context boundary

**Fix applied (in `LogCatch.svelte`):**

The working solution uses **`onmousedown`** on dropdown items and **`onblur`** with a delay on the input. No backdrop overlay.

### Why `onmousedown` + `onblur`:
- `onmousedown` fires **before** `blur` on the input, so the selection registers before the dropdown is dismissed
- On iOS Safari, `mousedown` fires via the touch-to-mouse compatibility layer on `<button>` elements
- `onblur` on the input closes the dropdown when the user taps anywhere else (input loses focus)
- The 150ms delay in `dismissDropdown()` ensures `onmousedown` on a dropdown item fires before the blur hides the dropdown

### What was removed:
- **Backdrop div** — caused z-index stacking issues on Safari, blocked all UI behind it
- **`onpointerup`** — unreliable on Safari through stacking context boundaries
- **`document.activeElement.blur()` in `selectSpecies`** — unnecessary, the input blur happens naturally

### What was kept:
- **`justSelected` flag** — prevents the `$effect` race condition that reopens the dropdown after selection
- **`onfocus` handler** — reopens dropdown if input is re-focused with text in it

```typescript
function selectSpecies(s: any) {
  justSelected = true;
  form.species_id = s.id;
  form.species_name = s.name;
  speciesQuery = s.name;
  showSpeciesDropdown = false;
}

function dismissDropdown() {
  // Delay so mousedown on a dropdown item fires before we hide it
  setTimeout(() => { showSpeciesDropdown = false; }, 150);
}
```

```svelte
<input ... onblur={dismissDropdown} />
{#if showSpeciesDropdown}
  <div class="dropdown">
    {#each filteredSpecies as s}
      <button class="dropdown-item" onmousedown={() => selectSpecies(s)}>
```

**Approaches that failed:**
1. `onclick` — doesn't fire on iOS Safari for elements in absolute-positioned containers
2. `onpointerup` + backdrop — backdrop's `position: fixed` inside `position: relative` parent breaks z-index stacking on Safari, blocking entire UI
3. `onpointerup` without backdrop — event doesn't fire reliably through stacking contexts on Safari

**Files changed:** `frontend/src/lib/pages/LogCatch.svelte`

---

## Bug 4: Map Resets / Infinite Scroll on Safari

**Symptom (original):**
Every time the user taps the Map tab, the map briefly shows the hardcoded Texas center (32.7767, -96.7970) at zoom 5, then jumps to fit catch bounds.

**Symptom (after initial fix):**
Map starts zooming to GPS location then enters infinite upward scroll on both desktop and mobile Safari.

**Root Causes:**
1. `MapView` was destroyed/recreated on every tab switch (`{#if}` unmounts components)
2. Manual `getCurrentPosition` + `flyTo` and catches `fitBounds` fired as competing animations, causing MapLibre's renderer to fight between two targets on Safari
3. Markers removed via `document.querySelectorAll('.catch-marker').remove()` instead of MapLibre's `marker.remove()` API — bypasses MapLibre's internal state tracking
4. `loadCatches()` called before map `load` event, causing positioning before tiles ready

**Fix applied:**

### App.svelte: CSS visibility instead of conditional rendering
Always render MapView, hide with `display: none` via `.hidden` class. Pass `visible` prop so map can resize when shown.

### MapView.svelte: Complete rewrite
- **Removed manual GPS positioning entirely.** The `GeolocateControl` button (crosshair icon) handles user-initiated location centering. No more competing `getCurrentPosition` + `flyTo` animations.
- **Proper marker lifecycle.** Track markers in an `activeMarkers: maplibregl.Marker[]` array. Clear by calling `marker.remove()` through MapLibre's API, not raw DOM queries.
- **Load catches after map ready.** Use `map.on('load', () => loadCatches())` so positioning only happens after tiles are loaded.
- **One-time `fitBounds` with `animate: false`.** Guard with `hasFittedBounds` flag. Instant positioning, no animation that can race.
- **Visibility resize uses `requestAnimationFrame`** and only fires on actual `false → true` transition (tracked via `prevVisible`), not on every reactive update.
- **Default center is continental US** (`[-98.5, 39.8]`, zoom 3) — reasonable starting view.

**Files changed:** `frontend/src/App.svelte`, `frontend/src/lib/pages/MapView.svelte`

---

## Feature 5: Photo Upload / Camera Capture in LogCatch

**Design decision:** Photo is an optional field within the Log Catch form (not photo-first flow). Placed between the Bait/Lure field and the Kept toggle for natural flow.

**Implementation:**

### Step 1: Add photo state and file input to LogCatch.svelte

Add state for selected photos and a hidden file input:

```typescript
let photoFiles = $state<File[]>([]);
let photoInput: HTMLInputElement;
```

Add a photo section in the form between Bait/Lure and Kept:

```svelte
<div class="form-group">
  <label>Photo</label>
  <input
    type="file"
    accept="image/*"
    capture="environment"
    multiple
    bind:this={photoInput}
    onchange={handlePhotoSelect}
    style="display:none"
  />
  <button class="btn btn-outline btn-block" type="button" onclick={() => photoInput.click()}>
    {photoFiles.length > 0 ? `${photoFiles.length} photo(s) selected` : 'Add Photo'}
  </button>
  {#if photoFiles.length > 0}
    <div class="photo-previews">
      {#each photoFiles as file, i}
        <div class="photo-thumb">
          <img src={URL.createObjectURL(file)} alt="Preview" />
          <button class="photo-remove" onpointerup={() => removePhoto(i)}>x</button>
        </div>
      {/each}
    </div>
  {/if}
</div>
```

Key attributes:
- `accept="image/*"` limits to images
- `capture="environment"` prompts the rear camera on mobile (iOS Safari and Android Chrome)
- `multiple` allows multiple photos

### Step 2: Handle file selection and removal

```typescript
function handlePhotoSelect(e: Event) {
  const input = e.target as HTMLInputElement;
  if (input.files) {
    photoFiles = [...photoFiles, ...Array.from(input.files)];
  }
  input.value = ''; // reset so same file can be re-selected
}

function removePhoto(index: number) {
  photoFiles = photoFiles.filter((_, i) => i !== index);
}
```

### Step 3: Upload photos after catch creation

In the `save()` function, after the catch is created successfully, upload photos using the existing `api.catches.addPhotos` endpoint:

```typescript
async function save() {
  saving = true;
  error = '';
  try {
    const created = await api.catches.create({ ... });

    // Upload photos if any were selected
    if (photoFiles.length > 0 && created?.id) {
      const formData = new FormData();
      for (const file of photoFiles) {
        formData.append('photos', file);
      }
      await api.catches.addPhotos(created.id, formData);
    }

    await loadCatches();
    onDone();
  } catch (e: any) {
    error = e.message || 'Failed to save catch';
  } finally {
    saving = false;
  }
}
```

### Step 4: Add photo preview styles

```css
.photo-previews {
  display: flex;
  gap: 8px;
  margin-top: 8px;
  overflow-x: auto;
}

.photo-thumb {
  position: relative;
  flex-shrink: 0;
}

.photo-thumb img {
  width: 64px;
  height: 64px;
  object-fit: cover;
  border-radius: 8px;
}

.photo-remove {
  position: absolute;
  top: -6px;
  right: -6px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--danger);
  color: white;
  border: none;
  font-size: 0.7rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}
```

**Note:** The backend `POST /api/v1/catches/:id/photos` endpoint and `api.catches.addPhotos` client method already exist. No backend changes needed.

**Files changed:** `frontend/src/lib/pages/LogCatch.svelte`

---

## Testing Plan

1. Build frontend: `cd frontend && npm run build`
2. Copy to embed dir: `cp -r frontend/dist internal/frontend/dist`
3. Run in dev mode: `FISHSCALE_DEV_MODE=true go run ./cmd/fishscale`
4. Test on iOS Safari (or Safari responsive design mode):
   - Species dropdown opens, tapping a species fills the field and closes dropdown
   - Tapping outside the dropdown dismisses it
   - Cancel button works at all times
   - Map preserves position across tab switches
   - Photo button opens camera on mobile, file picker on desktop
   - Photo preview shows after selection, can be removed
   - Saving a catch with photo uploads the photo
5. Run Go test suite: `go test ./... -v`
