# User Greeting Feature Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add personalized fishing-themed greetings with user's first name to Log Catch page

**Architecture:** Create GET /api/v1/me endpoint that returns authenticated user, add greeting array to LogCatch.svelte that picks random greeting and substitutes first name on component mount

**Tech Stack:** Go 1.25, chi router, Svelte 5, TypeScript

---

## Task 1: Create User Handler

**Files:**
- Create: `internal/handler/user.go`

**Step 1: Create UserHandler with GetMe method**

Create `internal/handler/user.go`:

```go
package handler

import (
	"net/http"

	"github.com/allen/fishscale/internal/middleware"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// GetMe returns the authenticated user's information
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	jsonResponse(w, http.StatusOK, user)
}
```

**Step 2: Commit**

```bash
git add internal/handler/user.go
git commit -m "feat: add user handler with GetMe endpoint

Returns authenticated user information from context.
No database query needed - user already injected by auth middleware.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: Register User API Route

**Files:**
- Modify: `internal/server/server.go`

**Step 1: Initialize user handler**

After line 48 (after stats handler initialization), add:

```go
user := handler.NewUserHandler()
```

**Step 2: Add /me route**

Inside the `/api/v1` route group (after line 73, after /export route), add:

```go
r.Get("/me", user.GetMe)
```

**Step 3: Test the endpoint**

Run dev server:
```bash
FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=./fish.db FISHSCALE_PHOTO_DIR=./photos GOWORK=off go run ./cmd/fishscale
```

Test endpoint:
```bash
curl http://localhost:8080/api/v1/me
```

Expected response:
```json
{
  "id": 1,
  "tailscale_id": "dev-user",
  "display_name": "Dev User",
  "created_at": "2026-02-17T12:00:00Z"
}
```

**Step 4: Commit**

```bash
git add internal/server/server.go
git commit -m "feat: register GET /api/v1/me endpoint

Returns authenticated user information.
Works with both DevAuth and Tailscale auth.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: Add API Client Method

**Files:**
- Modify: `frontend/src/lib/api.ts`

**Step 1: Add me endpoint to API client**

Add after the `stats` object (around line 41), before the `export` object:

```typescript
me: {
  get: () => request<any>('/me'),
},
```

**Step 2: Commit**

```bash
git add frontend/src/lib/api.ts
git commit -m "feat: add me.get() to API client

Fetches authenticated user information from /api/v1/me.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: Update LogCatch Component with Greetings

**Files:**
- Modify: `frontend/src/lib/pages/LogCatch.svelte`

**Step 1: Add greeting array at top of script section**

After the imports (around line 4), add:

```typescript
const FISHING_GREETINGS = [
  "Hey {name}! What did you catch today?",
  "Welcome back, {name}! Let's log that monster!",
  "Nice to see you, {name}! Ready to brag?",
  "What's biting, {name}?",
  "Reel 'em in, {name}!",
  "Another one, {name}? You're on fire!",
  "Tell us about it, {name}!",
  "What'd you hook, {name}?",
  "Time to record the catch, {name}!",
  "Let's hear it, {name}! What's the story?",
  "Back for more, {name}?",
  "What treasure did you land, {name}?",
];
```

**Step 2: Add greeting state variable**

After the existing state variables (around line 11), add:

```typescript
let greeting = $state('Log Catch'); // fallback
```

**Step 3: Add user fetch effect**

After the existing `$effect` hooks (around line 85, after the catch loading effect), add:

```typescript
// Fetch user and generate greeting
$effect(() => {
  api.me
    .get()
    .then((user) => {
      if (user.display_name) {
        const firstName = user.display_name.split(' ')[0];
        const randomGreeting =
          FISHING_GREETINGS[Math.floor(Math.random() * FISHING_GREETINGS.length)];
        greeting = randomGreeting.replace('{name}', firstName);
      }
    })
    .catch(() => {
      greeting = 'Log Catch'; // fallback on error
    });
});
```

**Step 4: Update page title**

Find the line with `<h1 class="page-title">` (around line 210) and change it from:

```svelte
<h1 class="page-title">{mode === 'edit' ? 'Edit Catch' : 'Log Catch'}</h1>
```

To:

```svelte
<h1 class="page-title">{mode === 'edit' ? 'Edit Catch' : greeting}</h1>
```

**Step 5: Commit**

```bash
git add frontend/src/lib/pages/LogCatch.svelte
git commit -m "feat: add personalized fishing greetings to Log Catch

Add 12 fishing-themed greeting templates with {name} placeholder.
Fetch user on mount and extract first name.
Pick random greeting and replace placeholder.
Falls back to 'Log Catch' on error.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 5: Build and Test

**Files:**
- None (testing step)

**Step 1: Build frontend**

```bash
cd frontend && npm run build
```

Expected: Build completes successfully

**Step 2: Copy dist to embed directory**

```bash
cd .. && rm -rf internal/frontend/dist && cp -r frontend/dist internal/frontend/dist
```

Expected: Files copied successfully

**Step 3: Restart dev server**

Kill existing server, then:

```bash
FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=./fish.db FISHSCALE_PHOTO_DIR=./photos GOWORK=off go run ./cmd/fishscale
```

Expected: Server starts on :8080

**Step 4: Manual testing checklist**

Test in browser at http://localhost:8080:

1. **Initial greeting:**
   - Click the + button (bottom nav)
   - Verify title shows personalized greeting instead of "Log Catch"
   - Should see something like "Hey Dev! What did you catch today?"

2. **Greeting variety:**
   - Navigate away (tap Log tab)
   - Click + button again
   - Verify greeting changes (may be same due to randomness, but should vary over multiple tries)

3. **Edit mode:**
   - Log a catch
   - Tap the catch to view details
   - Tap "Edit"
   - Verify title shows "Edit Catch" (not personalized greeting in edit mode)

4. **Fallback behavior:**
   - Stop the dev server
   - Click + button
   - Should show "Log Catch" (fallback)
   - Should still be able to use the form

**Step 5: Commit built frontend**

```bash
git add internal/frontend/dist
git commit -m "build: update frontend dist with user greeting feature

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 6: Update Documentation

**Files:**
- Modify: `docs/plans/todo.md`

**Step 1: Mark feature as completed**

In the "Ideas & Enhancements" section, change:

```markdown
- [ ] Bring in the tailscale users identity (name) and display it on the app
```

To:

```markdown
- [x] ~~Bring in the tailscale users identity (name) and display it on the app~~ (Completed 2026-02-17: added personalized fishing greetings to Log Catch page, see user-greeting-implementation.md)
```

And move it to the "Completed" section at the bottom.

**Step 2: Commit**

```bash
git add docs/plans/todo.md
git commit -m "docs: mark user greeting feature as completed

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Summary

**Total Tasks:** 6 tasks covering backend endpoint, API client, frontend greeting logic, testing, and documentation

**Key Changes:**
1. New UserHandler with GetMe endpoint (no DB query, uses auth context)
2. GET /api/v1/me route registered in server
3. API client method me.get() added
4. LogCatch.svelte has 12 greeting templates
5. Random greeting with first name extracted on component mount
6. Graceful fallback to "Log Catch" on error

**Testing:**
- Manual testing of greeting display
- Verify greeting variety over multiple navigations
- Confirm edit mode still shows "Edit Catch"
- Test fallback behavior when API unavailable

**No breaking changes** - feature is additive and gracefully degrades if user data unavailable.
