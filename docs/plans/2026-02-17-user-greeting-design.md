# User Greeting Feature - Design Document

**Date:** 2026-02-17
**Status:** Approved

## Problem Statement

Currently, the app shows no personalization or user identity information. The Log Catch page has a generic "Log Catch" title that doesn't acknowledge who's using the app. Users want to see their Tailscale identity displayed with fun, fishing-related greetings to make the logging experience more engaging and personal.

## Goals

1. Display user's first name from Tailscale identity in the Log Catch page
2. Show randomized fishing-related greetings that include the user's name
3. Make the experience fun and personalized without adding complexity
4. Maintain graceful fallback behavior if user data isn't available

## Approach: Frontend Greeting Array with User API

**Why this approach:**
- Simplest solution that meets all requirements
- No server-side greeting logic needed
- Easy to add/modify/remove greetings
- Fast performance - just string replacement
- Graceful degradation if API fails

### Key Design Decisions

1. **Placement:** Log Catch page title - replaces "Log Catch" header
2. **Name extraction:** Split DisplayName on space, take first word
3. **Randomization:** New greeting every time user navigates to Log Catch page
4. **Fallback:** Shows "Log Catch" if user fetch fails or name unavailable

## Architecture

### Backend Changes

**New Handler: UserHandler**
- Single endpoint: `GET /api/v1/me`
- Returns authenticated user from request context
- No database query needed (user already in context from auth middleware)

**Endpoint Response:**
```json
{
  "id": 1,
  "display_name": "John Doe",
  "tailscale_id": "user@example.com",
  "created_at": "2026-02-17T12:00:00Z"
}
```

### Frontend Changes

**API Client Addition:**
```typescript
me: {
  get: () => request<User>('/me')
}
```

**LogCatch.svelte Updates:**

1. **Greeting Templates:**
   - Array of 12 fishing-related greeting strings
   - Use `{name}` placeholder for substitution
   - Examples: "Hey {name}! What did you catch today?", "Welcome back, {name}! Let's log that monster!"

2. **User Fetch Logic:**
   - Fetch user on component mount via `$effect`
   - Extract first name: `displayName.split(' ')[0]`
   - Select random greeting: `Math.floor(Math.random() * GREETINGS.length)`
   - Replace `{name}` with first name
   - Set greeting state variable

3. **Title Rendering:**
   - Replace static "Log Catch" with dynamic `{greeting}` variable
   - Defaults to "Log Catch" until user loads

## Data Flow

```
1. User taps + button → Navigate to LogCatch page
2. LogCatch mounts → $effect runs
3. Fetch GET /api/v1/me
4. Extract first name from display_name
5. Pick random greeting from array
6. Replace {name} placeholder
7. Update greeting state
8. Render personalized title
```

## Error Handling

| Scenario | Behavior |
|----------|----------|
| API failure | Falls back to "Log Catch" |
| Empty DisplayName | Falls back to "Log Catch" |
| Single-word name | Uses full name (split returns one element) |
| Network timeout | Falls back to "Log Catch" |
| No user in context | API returns 401, falls back to "Log Catch" |

**Graceful degradation:** User can always log catches even if greeting fails.

## Greeting Examples

Full list of 12 greetings:
- "Hey {name}! What did you catch today?"
- "Welcome back, {name}! Let's log that monster!"
- "Nice to see you, {name}! Ready to brag?"
- "What's biting, {name}?"
- "Reel 'em in, {name}!"
- "Another one, {name}? You're on fire!"
- "Tell us about it, {name}!"
- "What'd you hook, {name}?"
- "Time to record the catch, {name}!"
- "Let's hear it, {name}! What's the story?"
- "Back for more, {name}?"
- "What treasure did you land, {name}?"

## Dev Mode Behavior

DevAuth middleware provides:
```go
DisplayName: "Dev User"
```

Result: "Hey Dev! What did you catch today?"

## Success Criteria

- ✅ User's first name appears in Log Catch page title
- ✅ Greeting is fishing-themed and varies each visit
- ✅ Works with Tailscale authentication
- ✅ Works in development mode
- ✅ Graceful fallback if user data unavailable
- ✅ No breaking changes to existing functionality

## Out of Scope

These are intentionally excluded from this iteration:

- Showing user name in other pages (Map, Stats, Settings)
- User profile/account page
- Editable "preferred name" field
- Time-of-day greetings ("Good morning, John!")
- Activity-based greetings (based on recent catches)
- User avatar/photo display

## Future Enhancements

Potential additions if this feature is well-received:

- **Settings page header:** Show "Signed in as John Doe" in Settings
- **Preferred name field:** Let users override first name extraction
- **Contextual greetings:** "Good morning, John!" based on time of day
- **Achievement-based greetings:** "20 catches this month, John! Legend!"

## Technical Notes

**No schema changes:**
- User model already has DisplayName field
- No new database tables or migrations needed

**No breaking changes:**
- New endpoint is additive
- Frontend gracefully handles missing user data
- Existing functionality unchanged

**Performance:**
- Single lightweight API call on LogCatch mount
- No caching needed (greetings are cheap to generate)
- Fast response (<50ms) since user is in auth context
