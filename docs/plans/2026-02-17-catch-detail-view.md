# Catch Detail View - Design Document

**Date:** 2026-02-17
**Status:** Approved

## Problem Statement

Currently, the catch log shows a list of catch cards with basic information (species, date, location, bait, weather). The cards are clickable but don't do anything - the `onEdit` callback just navigates back to the same log page. Users cannot view full catch details or edit existing catches.

## Goals

1. **View full catch details** - Tap a catch to see all fields, photos, and complete information
2. **Edit existing catches** - Allow users to update any field in a caught fish record
3. **Simple navigation** - Clear flow between list → detail → edit with back buttons
4. **Photo viewing** - Display catch photos in a gallery format

## Approach: Separate Detail Component

Create a dedicated read-only detail view component that shows all catch information, then reuse the existing LogCatch form for editing.

### Why This Works

- **Clean separation** - Detail view optimized for reading, form optimized for editing
- **Component reuse** - LogCatch already works well for creating catches, extend it for editing
- **Better UX** - Read-only view prevents accidental edits, clear "Edit" button for intentional changes
- **Flexible layout** - Can organize detail view differently than form layout

## Component Architecture

### New Component: `CatchDetail.svelte`

**Purpose:** Display full catch details in read-only format

**Props:**
```typescript
{
  catch: Catch;           // Full catch object with photos
  onBack: () => void;     // Return to catch list
  onEdit: () => void;     // Switch to edit mode
  onDelete: (id: number) => void;  // Delete this catch
}
```

**Layout Structure:**
```
CatchDetail.svelte
├── Photo Gallery (horizontal thumbnails, tap to view full-screen)
├── Header (Species name, date/time, kept/released badge)
├── Location Card (name, coordinates, "View on Map" link)
├── Size Card (length, weight) - if data exists
├── Gear Card (bait, rod, line, hook) - if data exists
├── Weather Card (conditions, temp, wind, pressure, humidity) - if data exists
├── Water Card (water temp, clarity) - if data exists
├── Notes Card - if notes exist
└── Action Buttons (Back, Edit, Delete)
```

### Modified Component: `CatchLog.svelte`

**New State:**
```typescript
let view = $state<'list' | 'detail' | 'edit'>('list');
let selectedCatchId = $state<number | null>(null);
let selectedCatch = $state<Catch | null>(null);
```

**Responsibilities:**
- Manage view state (list/detail/edit)
- Fetch full catch data when viewing details
- Pass catch data to CatchDetail or LogCatch components
- Handle navigation between views

### Modified Component: `LogCatch.svelte`

**New Props:**
```typescript
{
  catchId?: number;               // If provided, edit this catch
  mode: 'create' | 'edit';        // Create new or edit existing
  onDone: () => void;             // Callback when done
}
```

**Changes:**
- If `catchId` provided, fetch catch data and pre-fill form
- Update page title: "Log Catch" (create) vs "Edit Catch" (edit)
- Use `POST /api/v1/catches` for create, `PUT /api/v1/catches/:id` for edit
- "Cancel" button behavior depends on mode (different onDone handling)

## Data Flow & Navigation

### Flow 1: View Catch Details
```
1. User taps catch card in list
2. CatchLog.onEdit(catchId) called
3. Set view='detail', selectedCatchId=catchId
4. Fetch GET /api/v1/catches/:id (includes photos)
5. Set selectedCatch with full data
6. Render CatchDetail component
```

### Flow 2: Edit Catch
```
1. User taps "Edit" button in detail view
2. CatchDetail.onEdit() called
3. Set view='edit'
4. Render LogCatch with catchId={selectedCatchId} mode='edit'
5. LogCatch fetches catch data and pre-fills form
```

### Flow 3: Save Edits
```
1. User updates fields and taps "Save Catch"
2. LogCatch calls PUT /api/v1/catches/:id with updated data
3. On success, call onDone()
4. CatchLog re-fetches full catch data
5. Set view='detail' to show updated catch
```

### Flow 4: Delete Catch
```
1. User taps "Delete" in detail view
2. Confirm dialog appears
3. If confirmed, call DELETE /api/v1/catches/:id
4. On success, reload catch list
5. Set view='list' to return to list
```

### Flow 5: Navigation
```
Detail View → Back button → view='list' (show catch list)
Edit View → Cancel button → view='detail' (back to read-only)
Edit View → Save button → view='detail' (show updated catch)
```

## API Changes

**No backend changes required.** All necessary endpoints already exist:

- `GET /api/v1/catches` - List catches (already used)
- `GET /api/v1/catches/:id` - Fetch single catch with photos (already exists)
- `PUT /api/v1/catches/:id` - Update catch (already exists)
- `DELETE /api/v1/catches/:id` - Delete catch (already exists)

## UI/UX Specification

### CatchDetail Layout

**Header Section:**
- Species name: 1.5rem, bold, primary text color
- Date/time: 0.9rem, secondary text color, below species
- Kept/Released badge: chip style, green for kept, secondary for released

**Photo Gallery:**
- Horizontal scrollable row of thumbnails
- Each thumbnail: 80x80px, rounded corners (8px), object-fit cover
- Tap thumbnail to view full-screen (simple modal with close button)
- Show "(X photos)" text if multiple photos exist

**Info Cards:**
Each card uses `.card` style from theme, groups related information.

**Card 1: Location** (always shown if location data exists)
```
📍 LOCATION
Lake Fork, boat ramp cove
34.9690, -82.2665
[View on Map →]
```

**Card 2: Size** (shown if length_in OR weight_lb exists)
```
📏 SIZE
Length: 18.5 inches
Weight: 3.2 lb
```

**Card 3: Gear** (shown if any gear field exists)
```
🎣 GEAR
Bait/Lure: Texas Rig
Rod: 7' MH spinning
Line: 15lb braid + 12lb fluoro
Hook: 3/0 EWG
```

**Card 4: Weather** (shown if weather data exists)
```
☁️ WEATHER
Partly Cloudy, 72°F
Wind 11 mph SW
Pressure 1013 mb, Humidity 65%
```

**Card 5: Water** (shown if water data exists)
```
💧 WATER CONDITIONS
Temperature: 68°F
Clarity: Stained
```

**Card 6: Notes** (shown if notes exist)
```
📝 NOTES
Hit on third cast near submerged structure...
```

**Action Buttons:**
```
[Back]                    [Edit]
         Delete
```
- Back: `.btn .btn-outline`, left-aligned
- Edit: `.btn .btn-primary`, right-aligned
- Delete: Text link, danger color, centered below buttons

**Styling Details:**
- Use existing theme CSS variables
- Card labels: 0.8rem, uppercase, bold, secondary color
- Card content: 0.9rem, primary text color
- Consistent 16px padding, 12px margins
- Responsive on mobile (100% width, stacks vertically)

## Mobile Considerations

- All cards stack vertically on mobile
- Photo thumbnails scroll horizontally (no wrapping)
- Buttons stack if screen too narrow (< 360px)
- Back navigation uses browser back button on mobile
- Tap targets minimum 44x44px for accessibility

## Success Criteria

- ✅ Users can tap any catch card to view full details
- ✅ All catch fields are visible in detail view (nothing hidden)
- ✅ Photos display in a thumbnail gallery
- ✅ Users can tap "Edit" to modify any field
- ✅ Edit form pre-fills with existing catch data
- ✅ Save updates the catch and returns to detail view
- ✅ Back/Cancel navigation works intuitively
- ✅ Delete confirmation prevents accidental deletion
- ✅ No UI lockups or performance issues

## Future Enhancements

Not in scope for this iteration, but architectural hooks left for:

- **Photo management in edit mode** - Add/remove photos from existing catch
- **Map integration** - "View on Map" opens MapView with catch highlighted
- **Share catch** - Export single catch as image or link
- **Duplicate catch** - Create new catch pre-filled with this catch's data
