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

**Root Cause:**
The custom dropdown uses `onclick` handlers on `<button>` elements inside an absolutely-positioned container. On iOS Safari:
1. `onclick` on dynamically-shown elements inside absolute-positioned containers is unreliable — iOS doesn't always fire click events on non-anchor/non-input elements
2. The `$effect` that filters species re-runs when `speciesQuery` changes, which can re-trigger `showSpeciesDropdown = true` immediately after `selectSpecies` sets it to `false`, causing a race condition
3. No way to dismiss the dropdown by tapping outside it (no backdrop), so once stuck the user is trapped

**Fix (in `LogCatch.svelte`):**

### Step 1: Add a backdrop overlay to dismiss the dropdown
When `showSpeciesDropdown` is true, render an invisible full-screen `<div>` behind the dropdown that closes it on touch/click. This gives users a way out and prevents the trapped state.

```svelte
{#if showSpeciesDropdown}
  <!-- Invisible backdrop to dismiss dropdown on outside tap -->
  <div class="dropdown-backdrop" onclick={() => { showSpeciesDropdown = false; }}></div>
  <div class="dropdown">
    ...
  </div>
{/if}
```

```css
.dropdown-backdrop {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  z-index: 9; /* below dropdown z-index of 10 */
}
```

### Step 2: Use onpointerup instead of onclick for dropdown items
iOS Safari reliably fires pointer events on all elements. Replace `onclick` with `onpointerup` on each dropdown item.

```svelte
<button class="dropdown-item" onpointerup={() => selectSpecies(s)}>
```

### Step 3: Break the $effect race condition
The species filter `$effect` currently sets `showSpeciesDropdown` based on `speciesQuery.length > 0`. When `selectSpecies` sets `speciesQuery = s.name`, the effect re-runs and reopens the dropdown because `speciesQuery.length > 0`.

Fix: track whether a selection was just made. Add a `justSelected` flag that the effect checks:

```typescript
let justSelected = $state(false);

function selectSpecies(s: any) {
  justSelected = true;
  form.species_id = s.id;
  form.species_name = s.name;
  speciesQuery = s.name;
  showSpeciesDropdown = false;
}

// Filter species on query change
$effect(() => {
  if (justSelected) {
    justSelected = false;
    return;
  }
  if (speciesQuery.length > 0) {
    filteredSpecies = speciesList.filter(s =>
      s.name.toLowerCase().includes(speciesQuery.toLowerCase())
    ).slice(0, 8);
    showSpeciesDropdown = filteredSpecies.length > 0;
  } else {
    showSpeciesDropdown = false;
    form.species_id = null;
    form.species_name = '';
  }
});
```

### Step 4: Blur the input on selection
After selecting a species, blur the input to dismiss the iOS keyboard and prevent further focus-related issues:

```typescript
function selectSpecies(s: any) {
  justSelected = true;
  form.species_id = s.id;
  form.species_name = s.name;
  speciesQuery = s.name;
  showSpeciesDropdown = false;
  // Blur the input to dismiss iOS keyboard
  if (document.activeElement instanceof HTMLElement) {
    document.activeElement.blur();
  }
}
```

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
