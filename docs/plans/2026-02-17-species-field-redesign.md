# Species Field Redesign - Design Document

**Date:** 2026-02-17
**Status:** Approved

## Problem Statement

The current species dropdown has persistent UI issues:
- Custom dropdown locks up the page on iOS Safari
- Complex touch event handling (`ontouchend`, `onmousedown`, `onblur` with timers) still unreliable
- Users cannot dismiss the dropdown when it freezes
- Sometimes saves catch data to backend while UI appears unresponsive
- Adds unnecessary complexity with pre-seeded species table, categories, and species filter setting

The core issue: we're fighting browser behavior with custom dropdown code when the app's goal is speed and simplicity.

## Goals

1. **Reliability** - Species input must work flawlessly on all devices, especially iOS Safari
2. **Simplicity** - Remove complex custom dropdown code and species management
3. **Speed** - Make logging a catch as fast as possible ("back to fishing, not playing with an app")
4. **Data Preservation** - Keep all existing catch data during migration

## Approach: Freeform Text with Native Autocomplete

**Selected:** Approach 1 - Clean Migration with Data Preservation

Replace the pre-populated species dropdown with a freeform text field using HTML5 `<datalist>` for autocomplete. Autocomplete suggestions come from the user's own catch history, sorted by frequency (most-caught species appear first).

### Why This Works

- **Native datalist** - Browser-native, works reliably across all platforms including iOS Safari
- **No custom dropdown code** - Eliminates ~80 lines of complex event handling
- **User-specific** - Autocomplete learns from your catches, keeps the list small and relevant
- **Frequency-sorted** - If you catch bass 90% of the time, bass appears first
- **No categories needed** - User types what they caught, no freshwater/saltwater distinction required

## Database Schema Changes

### Migration Steps

1. Add `species_name TEXT NOT NULL DEFAULT ''` to catches table
2. Backfill data: `UPDATE catches SET species_name = (SELECT name FROM species WHERE species.id = catches.species_id)`
3. Drop foreign key constraint on species_id (requires table recreation in SQLite)
4. Drop species_id column from catches table
5. Drop the entire species table
6. Remove species_filter column from user_settings table (requires table recreation in SQLite)

### Migration Safety

- Idempotent - checks if species_name column exists before running
- Data preserved - backfill completes before any drops
- SQLite column removal requires table recreation (standard practice)

### Final Schema

**catches table (modified):**
```sql
CREATE TABLE catches (
  id INTEGER PRIMARY KEY,
  user_id INTEGER NOT NULL,
  species_name TEXT NOT NULL,  -- NEW: replaces species_id
  -- (all other columns unchanged)
);
```

**user_settings table (modified):**
```sql
CREATE TABLE user_settings (
  id INTEGER PRIMARY KEY,
  user_id INTEGER UNIQUE NOT NULL,
  theme TEXT NOT NULL DEFAULT 'system',
  units TEXT NOT NULL DEFAULT 'imperial',
  -- species_filter removed
);
```

**Tables dropped:**
- `species` table - completely removed

## API Changes

### New Endpoint

**GET /api/v1/autocomplete/species**

Returns frequency-sorted unique species names from authenticated user's catch history.

**Response:**
```json
["Largemouth Bass", "Bluegill", "Crappie", "Smallmouth Bass"]
```

**Query:**
```sql
SELECT species_name, COUNT(*) as catch_count
FROM catches
WHERE user_id = ? AND species_name != ''
GROUP BY species_name
ORDER BY catch_count DESC
LIMIT 50
```

**Handler location:** `internal/handler/autocomplete.go` (new file)

**Behavior:**
- Returns empty array for new users (no catches yet)
- Limit 50 unique species (reasonable for autocomplete)
- Scoped to authenticated user only

### Modified Endpoints

**POST /api/v1/catches**
- Change: `species_id` (integer) → `species_name` (string)
- Validation: species_name required, max 100 characters

**PUT /api/v1/catches/:id**
- Change: `species_id` (integer) → `species_name` (string)
- Validation: species_name required, max 100 characters

**GET/PUT /api/v1/settings**
- Remove: `species_filter` field from request/response body

### Removed Endpoints

- **DELETE:** `GET /api/v1/species` - no longer needed
- **DELETE:** `POST /api/v1/species` - no longer needed

## Frontend Changes

### LogCatch.svelte

**Removed (~80 lines):**
- Custom dropdown HTML (`.dropdown`, `.dropdown-item` structure)
- Complex event handling (`ontouchend`, `onmousedown`, `onblur`, `dismissDropdown`, `cancelDismiss`)
- `justSelected` flag and $effect re-trigger prevention logic
- `showSpeciesDropdown` state
- `filteredSpecies` filtering logic
- Species filter integration (checking `$settings.species_filter`)
- Custom dropdown CSS (~50 lines)

**Added (~15 lines):**
```svelte
<script>
  let speciesSuggestions = $state<string[]>([]);

  $effect(() => {
    api.autocomplete.species().then(s => {
      speciesSuggestions = s;
    });
  });

  let form = $state({
    species_name: '',  // replaces species_id + species_name tracking
    // ... other fields
  });
</script>

<div class="form-group">
  <label>Species</label>
  <input
    type="text"
    list="species-datalist"
    placeholder="e.g. Largemouth Bass"
    bind:value={form.species_name}
  />
  <datalist id="species-datalist">
    {#each speciesSuggestions as species}
      <option value={species}></option>
    {/each}
  </datalist>
</div>
```

**Net change:** ~65 lines removed, code simplified dramatically

### Settings.svelte

**Removed:**
- Species Filter radio group (All / Freshwater / Saltwater)
- `species_filter` from form state

### API Client (api.ts)

**Removed:**
```typescript
species: {
  list: () => Promise<Species[]>
  create: (name: string, category: string) => Promise<Species>
}
```

**Added:**
```typescript
autocomplete: {
  species: () => Promise<string[]>
}
```

**Modified:**
```typescript
catches: {
  create: (data: { species_name: string, ... }) => Promise<Catch>
  update: (id: number, data: { species_name: string, ... }) => Promise<Catch>
}
```

**Settings interface:**
```typescript
interface Settings {
  theme: string;
  units: string;
  // species_filter removed
}
```

## Backend Changes

### New Files
- `internal/handler/autocomplete.go` - New handler for autocomplete endpoints

### Modified Files
- `internal/database/migrations.go` - Add new migration function
- `internal/handler/catch.go` - Update create/update handlers to use species_name
- `internal/handler/settings.go` - Remove species_filter from settings handling
- `internal/model/models.go` - Update Catch model, remove Species model, update Settings model
- `internal/server/routes.go` - Add autocomplete route, remove species routes

### Deleted Files
- `internal/handler/species.go` - No longer needed

## Testing Considerations

### Manual Testing Checklist

1. **Migration:**
   - Start with existing database containing catches with species_id
   - Run migration, verify all catches have species_name populated
   - Verify species and species_id columns removed
   - Verify species_filter removed from user_settings

2. **Autocomplete:**
   - New user: autocomplete returns empty array
   - User with catches: autocomplete returns frequency-sorted species
   - Most-caught species appears first in suggestions

3. **Catch Creation:**
   - Type freeform species name, save catch
   - Select from autocomplete suggestions, save catch
   - Both methods work on iOS Safari and desktop

4. **UI Responsiveness:**
   - Species field never locks up the page
   - Can always dismiss/navigate away
   - No frozen dropdowns

5. **Settings:**
   - Species filter setting no longer visible
   - Theme and units still work

### Backend Tests

Update existing handler tests:
- `TestCreateCatch` - use species_name instead of species_id
- `TestUpdateCatch` - use species_name instead of species_id
- Add `TestAutocompleteSpecies` - test frequency sorting and user scoping

Remove tests:
- `TestListSpecies`
- `TestCreateSpecies`

## Rollout Plan

1. **Run `make ci`** before deployment to catch any issues
2. **Backup database** before running migration (standard practice)
3. **Deploy** - migration runs automatically on startup
4. **Verify** - Check that existing catches display species names correctly
5. **Monitor** - Watch for any user reports of species field issues (expect none)

## Success Criteria

- ✅ Species input works reliably on iOS Safari with no UI lockups
- ✅ Autocomplete suggestions appear from user's catch history
- ✅ All custom dropdown code removed (~80 lines eliminated)
- ✅ All existing catch data preserved with species names intact
- ✅ Users can log catches faster (no fighting with dropdown)

## Future Considerations

If species categories become important again (e.g., for analytics, regulations, or filtering):
- Can be added back on a feature branch
- Would likely use a hybrid approach: freeform entry + optional category tagging
- Current simplified design makes future iteration easier, not harder
