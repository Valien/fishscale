# Catch Detail View Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add clickable catch log entries that show full catch details in read-only view with edit capability

**Architecture:** Create CatchDetail.svelte for read-only display, extend LogCatch.svelte for editing, manage view state in CatchLog.svelte with list/detail/edit modes

**Tech Stack:** Svelte 5, TypeScript, existing API endpoints

---

## Task 1: Create CatchDetail Component

**Files:**
- Create: `frontend/src/lib/pages/CatchDetail.svelte`

**Step 1: Create CatchDetail component structure**

```svelte
<script lang="ts">
  interface Catch {
    id: number;
    species_name: string;
    caught_at: string;
    location_name: string;
    latitude: number | null;
    longitude: number | null;
    length_in: number | null;
    weight_lb: number | null;
    kept: boolean;
    bait_or_lure: string;
    rod_setup: string;
    line_info: string;
    hook_size: string;
    air_temp_f: number | null;
    wind_mph: number | null;
    wind_dir: string;
    conditions: string;
    pressure_mb: number | null;
    humidity_pct: number | null;
    water_temp_f: number | null;
    water_clarity: string;
    notes: string;
    photos: Array<{ id: number; url: string; filename: string }>;
  }

  let {
    catch: catchData,
    onBack,
    onEdit,
    onDelete,
  }: {
    catch: Catch;
    onBack: () => void;
    onEdit: () => void;
    onDelete: (id: number) => void;
  } = $props();

  let showPhotoModal = $state(false);
  let selectedPhotoUrl = $state('');

  function viewPhoto(url: string) {
    selectedPhotoUrl = url;
    showPhotoModal = true;
  }

  function closePhotoModal() {
    showPhotoModal = false;
    selectedPhotoUrl = '';
  }

  function handleDelete() {
    if (confirm('Delete this catch? This cannot be undone.')) {
      onDelete(catchData.id);
    }
  }
</script>

<div class="page">
  <h1 class="page-title">Catch Details</h1>

  <!-- Photo Gallery -->
  {#if catchData.photos && catchData.photos.length > 0}
    <div class="photo-gallery">
      {#each catchData.photos as photo}
        <button class="photo-thumb" onclick={() => viewPhoto(photo.url)} type="button">
          <img src={photo.url} alt="Catch photo" />
        </button>
      {/each}
      {#if catchData.photos.length > 1}
        <p class="photo-count">({catchData.photos.length} photos)</p>
      {/if}
    </div>
  {/if}

  <!-- Header -->
  <div class="card header-card">
    <h2 class="species-name">{catchData.species_name || 'Unknown Species'}</h2>
    <p class="catch-date">{new Date(catchData.caught_at).toLocaleString()}</p>
    {#if catchData.kept}
      <span class="chip chip-success">Kept</span>
    {:else}
      <span class="chip">Released</span>
    {/if}
  </div>

  <!-- Location Card -->
  {#if catchData.location_name || catchData.latitude}
    <div class="card">
      <h3 class="card-label">📍 Location</h3>
      {#if catchData.location_name}
        <p>{catchData.location_name}</p>
      {/if}
      {#if catchData.latitude && catchData.longitude}
        <p class="coords">{catchData.latitude.toFixed(4)}, {catchData.longitude.toFixed(4)}</p>
      {/if}
    </div>
  {/if}

  <!-- Size Card -->
  {#if catchData.length_in || catchData.weight_lb}
    <div class="card">
      <h3 class="card-label">📏 Size</h3>
      {#if catchData.length_in}
        <p>Length: {catchData.length_in} inches</p>
      {/if}
      {#if catchData.weight_lb}
        <p>Weight: {catchData.weight_lb} lb</p>
      {/if}
    </div>
  {/if}

  <!-- Gear Card -->
  {#if catchData.bait_or_lure || catchData.rod_setup || catchData.line_info || catchData.hook_size}
    <div class="card">
      <h3 class="card-label">🎣 Gear</h3>
      {#if catchData.bait_or_lure}
        <p><strong>Bait/Lure:</strong> {catchData.bait_or_lure}</p>
      {/if}
      {#if catchData.rod_setup}
        <p><strong>Rod:</strong> {catchData.rod_setup}</p>
      {/if}
      {#if catchData.line_info}
        <p><strong>Line:</strong> {catchData.line_info}</p>
      {/if}
      {#if catchData.hook_size}
        <p><strong>Hook:</strong> {catchData.hook_size}</p>
      {/if}
    </div>
  {/if}

  <!-- Weather Card -->
  {#if catchData.conditions || catchData.air_temp_f}
    <div class="card">
      <h3 class="card-label">☁️ Weather</h3>
      {#if catchData.conditions}
        <p>{catchData.conditions}{#if catchData.air_temp_f}, {catchData.air_temp_f.toFixed(0)}°F{/if}</p>
      {/if}
      {#if catchData.wind_mph}
        <p>Wind {catchData.wind_mph.toFixed(0)} mph {catchData.wind_dir || ''}</p>
      {/if}
      {#if catchData.pressure_mb || catchData.humidity_pct}
        <p>
          {#if catchData.pressure_mb}Pressure {catchData.pressure_mb.toFixed(0)} mb{/if}
          {#if catchData.pressure_mb && catchData.humidity_pct}, {/if}
          {#if catchData.humidity_pct}Humidity {catchData.humidity_pct.toFixed(0)}%{/if}
        </p>
      {/if}
    </div>
  {/if}

  <!-- Water Card -->
  {#if catchData.water_temp_f || catchData.water_clarity}
    <div class="card">
      <h3 class="card-label">💧 Water Conditions</h3>
      {#if catchData.water_temp_f}
        <p>Temperature: {catchData.water_temp_f.toFixed(0)}°F</p>
      {/if}
      {#if catchData.water_clarity}
        <p>Clarity: {catchData.water_clarity}</p>
      {/if}
    </div>
  {/if}

  <!-- Notes Card -->
  {#if catchData.notes}
    <div class="card">
      <h3 class="card-label">📝 Notes</h3>
      <p class="notes-text">{catchData.notes}</p>
    </div>
  {/if}

  <!-- Action Buttons -->
  <div class="action-buttons">
    <button class="btn btn-outline" onclick={onBack}>Back</button>
    <button class="btn btn-primary" onclick={onEdit}>Edit</button>
  </div>
  <button class="delete-link" onclick={handleDelete}>Delete</button>
</div>

<!-- Photo Modal -->
{#if showPhotoModal}
  <div class="photo-modal" onclick={closePhotoModal}>
    <div class="photo-modal-content" onclick={(e) => e.stopPropagation()}>
      <button class="photo-modal-close" onclick={closePhotoModal}>×</button>
      <img src={selectedPhotoUrl} alt="Full size catch photo" />
    </div>
  </div>
{/if}

<style>
  .photo-gallery {
    display: flex;
    gap: 8px;
    overflow-x: auto;
    padding: 8px 0;
    margin-bottom: 12px;
  }

  .photo-thumb {
    flex-shrink: 0;
    width: 80px;
    height: 80px;
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid var(--card-border);
    background: none;
    padding: 0;
    cursor: pointer;
  }

  .photo-thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .photo-count {
    font-size: 0.8rem;
    color: var(--text-secondary);
    align-self: center;
  }

  .header-card {
    text-align: center;
  }

  .species-name {
    font-size: 1.5rem;
    font-weight: 700;
    margin-bottom: 6px;
  }

  .catch-date {
    font-size: 0.9rem;
    color: var(--text-secondary);
    margin-bottom: 8px;
  }

  .card-label {
    font-size: 0.8rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-secondary);
    margin-bottom: 8px;
  }

  .card p {
    margin-bottom: 4px;
    font-size: 0.9rem;
  }

  .coords {
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  .notes-text {
    white-space: pre-wrap;
    line-height: 1.6;
  }

  .chip-success {
    background: rgba(25, 135, 84, 0.1);
    color: var(--success);
  }

  .action-buttons {
    display: flex;
    gap: 12px;
    margin-top: 16px;
  }

  .action-buttons button {
    flex: 1;
  }

  .delete-link {
    display: block;
    margin: 12px auto 0;
    background: none;
    border: none;
    color: var(--danger);
    cursor: pointer;
    text-align: center;
    font-size: 0.9rem;
    text-decoration: underline;
  }

  .photo-modal {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.9);
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .photo-modal-content {
    position: relative;
    max-width: 90vw;
    max-height: 90vh;
  }

  .photo-modal-content img {
    width: 100%;
    height: 100%;
    object-fit: contain;
  }

  .photo-modal-close {
    position: absolute;
    top: -40px;
    right: 0;
    background: none;
    border: none;
    color: white;
    font-size: 2rem;
    cursor: pointer;
    width: 40px;
    height: 40px;
  }

  @media (max-width: 360px) {
    .action-buttons {
      flex-direction: column;
    }
  }
</style>
```

**Step 2: Commit**

```bash
git add frontend/src/lib/pages/CatchDetail.svelte
git commit -m "feat: create CatchDetail component for viewing catch details

Display all catch fields in read-only cards with photo gallery.
Includes Back, Edit, Delete actions.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: Update API Client

**Files:**
- Modify: `frontend/src/lib/api.ts`

**Step 1: Add getCatch method to API client**

Find the `catches` object and add a `get` method:

```typescript
catches: {
  list: async (): Promise<Catch[]> => {
    // ... existing code
  },
  get: async (id: number): Promise<Catch> => {
    const res = await fetch(`/api/v1/catches/${id}`);
    if (!res.ok) throw new Error('Failed to fetch catch');
    return res.json();
  },
  create: async (data: any): Promise<Catch> => {
    // ... existing code
  },
  update: async (id: number, data: any): Promise<Catch> => {
    const res = await fetch(`/api/v1/catches/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error('Failed to update catch');
    return res.json();
  },
  // ... rest of catches methods
},
```

**Step 2: Commit**

```bash
git add frontend/src/lib/api.ts
git commit -m "feat: add get and update methods to catches API client

Add catches.get() for fetching single catch with photos.
Add catches.update() for updating existing catch.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: Update CatchLog Component

**Files:**
- Modify: `frontend/src/lib/pages/CatchLog.svelte`

**Step 1: Import CatchDetail component**

Add at top of script section:

```typescript
import CatchDetail from './CatchDetail.svelte';
import LogCatch from './LogCatch.svelte';
import { api } from '../api';
```

**Step 2: Add view state management**

Replace the existing props and add new state:

```typescript
let { onEdit }: { onEdit: (id: number) => void } = $props();
let search = $state('');

// NEW: View state management
let view = $state<'list' | 'detail' | 'edit'>('list');
let selectedCatchId = $state<number | null>(null);
let selectedCatch = $state<any | null>(null);
let loading CatchDetail = $state(false);
```

**Step 3: Add fetch catch function**

```typescript
async function fetchCatch(id: number) {
  loadingDetail = true;
  try {
    const data = await api.catches.get(id);
    selectedCatch = data;
    view = 'detail';
  } catch (err) {
    alert('Failed to load catch details');
    view = 'list';
  } finally {
    loadingDetail = false;
  }
}

function handleViewDetail(id: number) {
  selectedCatchId = id;
  fetchCatch(id);
}

function handleBackToList() {
  view = 'list';
  selectedCatchId = null;
  selectedCatch = null;
  loadCatches(); // Refresh list
}

function handleEditCatch() {
  view = 'edit';
}

function handleEditDone() {
  if (selectedCatchId) {
    fetchCatch(selectedCatchId); // Reload catch to show updated data
  }
}

async function handleDeleteCatch(id: number) {
  await deleteCatch(id);
  handleBackToList();
}
```

**Step 4: Update template to show different views**

Replace the entire template section:

```svelte
<div class="page">
  {#if view === 'list'}
    <h1 class="page-title">Catch Log</h1>

    <div class="form-group">
      <input type="text" placeholder="Search catches..." bind:value={search} />
    </div>

    {#if $loading}
      <div class="empty-state"><p>Loading...</p></div>
    {:else if filtered.length === 0}
      <div class="empty-state">
        <p>No catches yet</p>
        <p>Hit the + button to log your first catch!</p>
      </div>
    {:else}
      {#each filtered as c (c.id)}
        <div class="card catch-card" onclick={() => handleViewDetail(c.id)}>
          <!-- ... existing catch card content unchanged ... -->
          <div class="catch-header">
            <span class="catch-species">{c.species_name || 'Unknown Species'}</span>
            <span class="catch-date">{new Date(c.caught_at).toLocaleDateString()}</span>
          </div>
          <div class="catch-details">
            {#if c.location_name}
              <span>{c.location_name}</span>
            {/if}
            {#if c.weight_lb}
              <span>{c.weight_lb} lb</span>
            {/if}
            {#if c.length_in}
              <span>{c.length_in}"</span>
            {/if}
            {#if c.bait_or_lure}
              <span class="chip chip-primary">{c.bait_or_lure}</span>
            {/if}
            {#if c.kept}
              <span class="chip">Kept</span>
            {/if}
          </div>
          {#if c.conditions}
            <div class="catch-weather">
              {c.conditions}
              {#if c.air_temp_f}{c.air_temp_f.toFixed(0)}°F{/if}
            </div>
          {/if}
        </div>
      {/each}
    {/if}
  {:else if view === 'detail'}
    {#if loadingDetail}
      <div class="empty-state"><p>Loading...</p></div>
    {:else if selectedCatch}
      <CatchDetail
        catch={selectedCatch}
        onBack={handleBackToList}
        onEdit={handleEditCatch}
        onDelete={handleDeleteCatch}
      />
    {/if}
  {:else if view === 'edit'}
    <LogCatch catchId={selectedCatchId} mode="edit" onDone={handleEditDone} />
  {/if}
</div>
```

**Step 5: Remove delete button from catch cards**

Delete the delete button that was in the catch card (it's now in CatchDetail).

**Step 6: Commit**

```bash
git add frontend/src/lib/pages/CatchLog.svelte
git commit -m "feat: add view state management to CatchLog

Add list/detail/edit views with state management.
Fetch full catch data when viewing details.
Remove delete button from list cards.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: Update LogCatch Component for Edit Mode

**Files:**
- Modify: `frontend/src/lib/pages/LogCatch.svelte`

**Step 1: Update props**

Replace the existing props at the top of the script section:

```typescript
let {
  catchId = undefined,
  mode = 'create',
  onDone,
}: {
  catchId?: number;
  mode?: 'create' | 'edit';
  onDone: () => void;
} = $props();
```

**Step 2: Add loading state for existing catch**

```typescript
let loadingCatch = $state(false);
```

**Step 3: Add effect to load catch data in edit mode**

After the existing $effect hooks, add:

```typescript
// Load existing catch data in edit mode
$effect(() => {
  if (mode === 'edit' && catchId) {
    loadingCatch = true;
    api.catches.get(catchId).then((data) => {
      form.caught_at = new Date(new Date(data.caught_at).getTime() - new Date().getTimezoneOffset() * 60000)
        .toISOString()
        .slice(0, 16);
      form.latitude = data.latitude;
      form.longitude = data.longitude;
      form.location_name = data.location_name || '';
      form.species_name = data.species_name || '';
      form.bait_or_lure = data.bait_or_lure || '';
      form.kept = data.kept || false;
      form.length_in = data.length_in;
      form.weight_lb = data.weight_lb;
      form.rod_setup = data.rod_setup || '';
      form.line_info = data.line_info || '';
      form.hook_size = data.hook_size || '';
      form.water_temp_f = data.water_temp_f;
      form.water_clarity = data.water_clarity || '';
      form.notes = data.notes || '';
      form.air_temp_f = data.air_temp_f;
      form.wind_mph = data.wind_mph;
      form.wind_dir = data.wind_dir || '';
      form.conditions = data.conditions || '';
      form.pressure_mb = data.pressure_mb;
      form.humidity_pct = data.humidity_pct;
      loadingCatch = false;
    }).catch(() => {
      error = 'Failed to load catch data';
      loadingCatch = false;
    });
  }
});
```

**Step 4: Update save function to handle both create and edit**

Replace the existing `save` function:

```typescript
async function save() {
  saving = true;
  error = '';
  try {
    if (mode === 'edit' && catchId) {
      // Update existing catch
      await api.catches.update(catchId, {
        caught_at: new Date(form.caught_at).toISOString(),
        latitude: form.latitude,
        longitude: form.longitude,
        location_name: form.location_name,
        species_name: form.species_name,
        bait_or_lure: form.bait_or_lure,
        kept: form.kept,
        length_in: form.length_in,
        weight_lb: form.weight_lb,
        rod_setup: form.rod_setup,
        line_info: form.line_info,
        hook_size: form.hook_size,
        water_temp_f: form.water_temp_f,
        water_clarity: form.water_clarity,
        notes: form.notes,
        air_temp_f: form.air_temp_f,
        wind_mph: form.wind_mph,
        wind_dir: form.wind_dir,
        conditions: form.conditions,
        pressure_mb: form.pressure_mb,
        humidity_pct: form.humidity_pct,
      });
      await loadCatches();
      onDone();
    } else {
      // Create new catch (existing code)
      const created = await api.catches.create({
        caught_at: new Date(form.caught_at).toISOString(),
        latitude: form.latitude,
        longitude: form.longitude,
        location_name: form.location_name,
        species_name: form.species_name,
        bait_or_lure: form.bait_or_lure,
        kept: form.kept,
        length_in: form.length_in,
        weight_lb: form.weight_lb,
        rod_setup: form.rod_setup,
        line_info: form.line_info,
        hook_size: form.hook_size,
        water_temp_f: form.water_temp_f,
        water_clarity: form.water_clarity,
        notes: form.notes,
        air_temp_f: form.air_temp_f,
        wind_mph: form.wind_mph,
        wind_dir: form.wind_dir,
        conditions: form.conditions,
        pressure_mb: form.pressure_mb,
        humidity_pct: form.humidity_pct,
      });

      if (photoFiles.length > 0 && created?.id) {
        const formData = new FormData();
        for (const file of photoFiles) {
          formData.append('photos', file);
        }
        await api.catches.addPhotos(created.id, formData);
      }

      await loadCatches();
      onDone();
    }
  } catch (e: any) {
    error = e.message || 'Failed to save catch';
  } finally {
    saving = false;
  }
}
```

**Step 5: Update page title**

Update the template to show different title based on mode:

```svelte
<div class="page">
  <h1 class="page-title">{mode === 'edit' ? 'Edit Catch' : 'Log Catch'}</h1>

  {#if loadingCatch}
    <div class="empty-state"><p>Loading catch data...</p></div>
  {:else}
    <!-- ... rest of form ... -->
  {/if}
</div>
```

**Step 6: Commit**

```bash
git add frontend/src/lib/pages/LogCatch.svelte
git commit -m "feat: extend LogCatch to support editing existing catches

Add catchId and mode props for edit mode.
Load existing catch data and pre-fill form.
Use PUT endpoint for updates, POST for creates.
Update page title based on mode.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 5: Build and Test

**Files:**
- None (testing step)

**Step 1: Build frontend**

Run: `cd frontend && npm run build`

Expected: Build completes successfully

**Step 2: Copy dist to embed directory**

Run: `cd .. && rm -rf internal/frontend/dist && cp -r frontend/dist internal/frontend/dist`

Expected: Files copied successfully

**Step 3: Start dev server**

Run: `FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=./fish.db FISHSCALE_PHOTO_DIR=./photos GOWORK=off go run ./cmd/fishscale`

Expected: Server starts on :8080

**Step 4: Manual testing checklist**

Test in browser at http://localhost:8080:

1. **View catch details:**
   - Go to Log tab
   - Tap a catch card
   - Verify all fields display correctly
   - Verify photos appear (if catch has photos)
   - Tap photo thumbnail to view full-screen
   - Close photo modal

2. **Edit catch:**
   - From detail view, tap "Edit"
   - Verify form pre-fills with catch data
   - Modify a field (e.g., change species name)
   - Tap "Save Catch"
   - Verify returns to detail view with updated data

3. **Navigation:**
   - From detail view, tap "Back"
   - Verify returns to catch list
   - From edit view, tap "Cancel"
   - Verify returns to detail view

4. **Delete catch:**
   - From detail view, tap "Delete"
   - Confirm deletion
   - Verify returns to list
   - Verify catch no longer appears in list

**Step 5: Commit built frontend**

```bash
git add internal/frontend/dist
git commit -m "build: update frontend dist with catch detail view

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 6: Update Design Doc

**Files:**
- Modify: `docs/plans/2026-02-16-fishscale-design.md`

**Step 1: Mark catch log entries as completed**

Find the "Ideas & Enhancements" section and update:

```markdown
- [x] ~~Catch log entries should be clickable/editable~~ (Completed 2026-02-17: added CatchDetail view and edit mode, see catch-detail-view.md)
```

**Step 2: Commit**

```bash
git add docs/plans/2026-02-16-fishscale-design.md
git commit -m "docs: mark catch log entries feature as completed

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Summary

**Total Tasks:** 6 tasks covering frontend components, API updates, and testing

**Key Changes:**
1. New CatchDetail.svelte component (~250 lines)
2. CatchLog.svelte manages list/detail/edit views
3. LogCatch.svelte supports both create and edit modes
4. API client adds get() and update() methods
5. Photo gallery with full-screen viewing
6. Complete navigation flow between views

**Testing:**
- Manual testing of all user flows
- Verify view/edit/delete functionality
- Test navigation between views
- Confirm photo viewing works

**No backend changes required** - all API endpoints already exist.
