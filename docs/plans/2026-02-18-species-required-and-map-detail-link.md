# Species Required + Map Detail Link — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make species a required field on the catch form, and add a "View Details" link in map pin popups that navigates to the catch detail view.

**Architecture:** Two independent frontend-only changes. Feature 1 adds validation to the save function. Feature 2 threads a callback from MapView through App.svelte to CatchLog to open a catch detail by ID.

**Tech Stack:** Svelte 5, TypeScript, MapLibre GL JS

---

### Task 1: Add species required validation to LogCatch

**Files:**
- Modify: `frontend/src/lib/pages/LogCatch.svelte:166-169` (save function)

**Step 1: Add validation at top of save()**

In `LogCatch.svelte`, add a species check as the first thing in the `save()` function (line 166), before setting `saving = true`:

```typescript
  async function save() {
    if (!form.species_name.trim()) {
      error = 'Species is required';
      return;
    }
    saving = true;
    error = '';
```

This replaces lines 166-168. The existing error banner at line 255 already displays `error` state, so no template changes needed.

**Step 2: Verify manually**

Run the dev server:
```bash
cd /Users/allen/Documents/GitHub/fishscale
FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=./fish.db FISHSCALE_PHOTO_DIR=./photos GOWORK=off go run ./cmd/fishscale
```

Test: Open Log Catch, leave species blank, tap Save. Should see "Species is required" error. Fill in species, tap Save. Should save normally. Tap Cancel with blank species — should navigate away without error.

**Step 3: Commit**

```bash
git add frontend/src/lib/pages/LogCatch.svelte
git commit -m "feat: make species field required on catch form"
```

---

### Task 2: Add onViewCatch callback prop to MapView

**Files:**
- Modify: `frontend/src/lib/pages/MapView.svelte:7` (props) and `44-72` (popup builder)

**Step 1: Add the callback prop**

In `MapView.svelte` line 7, add `onViewCatch` to the props:

```typescript
  let { visible = true, onViewCatch }: { visible?: boolean; onViewCatch?: (id: number) => void } = $props();
```

**Step 2: Add "View Details" link to the popup DOM**

In the `syncMarkers` function, after the bait section (after line 71), add a "View Details" link to the popup element. Insert before the `new maplibregl.Popup(...)` line (line 73):

```typescript
      if (onViewCatch) {
        popupEl.appendChild(document.createElement('br'));
        const link = document.createElement('a');
        link.textContent = 'View Details →';
        link.href = '#';
        link.style.cssText = 'color:var(--primary, #0d6efd);text-decoration:none;font-size:0.85rem;font-weight:600;display:inline-block;margin-top:4px;';
        link.addEventListener('click', (e) => {
          e.preventDefault();
          onViewCatch(c.id);
        });
        popupEl.appendChild(link);
      }
```

**Step 3: Commit**

```bash
git add frontend/src/lib/pages/MapView.svelte
git commit -m "feat: add View Details link to map pin popups"
```

---

### Task 3: Wire up App.svelte to pass catch ID from map to log

**Files:**
- Modify: `frontend/src/App.svelte:11` (state), `25-27` (handler), `32` (MapView), `35` (CatchLog)

**Step 1: Add viewCatchId state and handler**

In `App.svelte`, add a `viewCatchId` state variable after line 11:

```typescript
  let viewCatchId = $state<number | null>(null);
```

Add a new handler after `handleEditCatch` (after line 27):

```typescript
  function handleViewCatch(id: number) {
    viewCatchId = id;
    activePage = 'log';
  }
```

**Step 2: Pass onViewCatch to MapView and viewCatchId to CatchLog**

Update the MapView line (line 32) to pass the callback:

```svelte
    <MapView visible={activePage === 'map'} onViewCatch={handleViewCatch} />
```

Update the CatchLog line (line 35) to pass the catch ID:

```svelte
    <CatchLog onEdit={handleEditCatch} {viewCatchId} />
```

**Step 3: Commit**

```bash
git add frontend/src/App.svelte
git commit -m "feat: wire map View Details to CatchLog via App state"
```

---

### Task 4: Accept viewCatchId in CatchLog and auto-open detail

**Files:**
- Modify: `frontend/src/lib/pages/CatchLog.svelte:1-6` (props) and add an effect

**Step 1: Add viewCatchId prop**

Add the prop and onEdit to a proper props declaration. Replace lines 1-5:

```typescript
<script lang="ts">
  import { catches, loading, loadCatches, deleteCatch } from '../stores/catches';
  import CatchDetail from './CatchDetail.svelte';
  import LogCatch from './LogCatch.svelte';
  import { api } from '../api';

  let {
    onEdit,
    viewCatchId = null,
  }: {
    onEdit: (id: number) => void;
    viewCatchId?: number | null;
  } = $props();
```

**Step 2: Add effect to react to viewCatchId**

After the existing `$effect` for `loadCatches()` (after line 11 in original), add:

```typescript
  // Auto-open catch detail when navigated from map
  $effect(() => {
    if (viewCatchId) {
      handleViewDetail(viewCatchId);
    }
  });
```

**Step 3: Verify manually**

With the dev server running, create a catch with a location. Go to the map, click a pin, click "View Details →". Should switch to the Log tab and show the catch detail view. Click Back — should return to the catch list.

**Step 4: Commit**

```bash
git add frontend/src/lib/pages/CatchLog.svelte
git commit -m "feat: auto-open catch detail when navigated from map popup"
```

---

### Task 5: Build frontend, run CI, update todo.md

**Step 1: Build frontend**

```bash
cd /Users/allen/Documents/GitHub/fishscale/frontend && npm run build && cd ..
rm -rf internal/frontend/dist && cp -r frontend/dist internal/frontend/dist
```

**Step 2: Run CI checks**

```bash
make ci
```

Fix any issues that arise.

**Step 3: Update todo.md**

Move items #4 and #5 from Ideas to Completed in `docs/plans/todo.md`.

**Step 4: Commit**

```bash
git add -A
git commit -m "build: update frontend dist with species required + map detail link features

feat: make species required on catch form
feat: add View Details link in map pin popups"
```
