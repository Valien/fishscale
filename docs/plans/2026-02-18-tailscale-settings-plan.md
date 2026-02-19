# Tailscale Identity on Settings Page — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Surface Tailscale identity and tailnet info on the Settings page (GH#2).

**Architecture:** Expand the WhoIs data extracted in middleware, pass it through request context, enrich the `/api/v1/me` response, and render a read-only "Account" card at the top of Settings.svelte.

**Tech Stack:** Go (chi, sqlx), Svelte 5, TypeScript

---

### Task 1: Add `TailscaleInfo` model

**Files:**
- Modify: `internal/model/models.go`

**Step 1: Add the struct**

Add after the `User` struct (around line 10):

```go
// TailscaleInfo holds transient identity data from the Tailscale WhoIs response.
// Not persisted — fresh on every request.
type TailscaleInfo struct {
	LoginName     string `json:"login_name"`
	DisplayName   string `json:"display_name"`
	TailscaleID   string `json:"tailscale_id"`
	NodeName      string `json:"node_name"`
	ProfilePicURL string `json:"profile_pic_url,omitempty"`
}
```

**Step 2: Verify it compiles**

Run: `GOWORK=off go build ./internal/model/`
Expected: success, no errors

**Step 3: Commit**

```bash
git add internal/model/models.go
git commit -m "feat: add TailscaleInfo model struct"
```

---

### Task 2: Add context helpers for TailscaleInfo

**Files:**
- Modify: `internal/middleware/auth.go`

**Step 1: Add context key and helpers**

Add below the existing `userContextKey` and helper functions:

```go
const tsInfoContextKey contextKey = "tailscale_info"

// TailscaleInfoFromContext retrieves the Tailscale identity info from the request context.
// Returns nil if not present (e.g., tests without TailscaleInfo setup).
func TailscaleInfoFromContext(ctx context.Context) *model.TailscaleInfo {
	info, _ := ctx.Value(tsInfoContextKey).(*model.TailscaleInfo)
	return info
}

// WithTailscaleInfo returns a new context with the given TailscaleInfo attached.
func WithTailscaleInfo(ctx context.Context, info *model.TailscaleInfo) context.Context {
	return context.WithValue(ctx, tsInfoContextKey, info)
}
```

**Step 2: Update DevAuth to include placeholder TailscaleInfo**

Replace the existing `DevAuth` function body so it also sets TailscaleInfo:

```go
func DevAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		devUser := &model.User{
			ID:          1,
			TailscaleID: "dev-user",
			DisplayName: "Dev User",
			CreatedAt:   time.Now(),
		}
		devTSInfo := &model.TailscaleInfo{
			LoginName:   "dev@localhost",
			DisplayName: "Dev User",
			TailscaleID: "dev-user",
			NodeName:    "fishscale.dev.ts.net",
		}
		ctx := WithUser(r.Context(), devUser)
		ctx = WithTailscaleInfo(ctx, devTSInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

**Step 3: Verify it compiles**

Run: `GOWORK=off go build ./internal/middleware/`
Expected: success, no errors

**Step 4: Commit**

```bash
git add internal/middleware/auth.go
git commit -m "feat: add TailscaleInfo context helpers and dev defaults"
```

---

### Task 3: Extract TailscaleInfo in Tailscale middleware

**Files:**
- Modify: `internal/middleware/tailscale.go`

**Step 1: Extract additional fields from WhoIs and store in context**

After the existing user upsert+fetch block (around line 42, before `ctx := WithUser(...)`), add TailscaleInfo construction. Also extract `Node.Name` and strip the trailing dot:

```go
nodeName := ""
if whois.Node != nil {
	nodeName = strings.TrimSuffix(whois.Node.Name, ".")
}
tsInfo := &model.TailscaleInfo{
	LoginName:     whois.UserProfile.LoginName,
	DisplayName:   whois.UserProfile.DisplayName,
	TailscaleID:   tailscaleID,
	NodeName:      nodeName,
	ProfilePicURL: whois.UserProfile.ProfilePicURL,
}

ctx := WithUser(r.Context(), &user)
ctx = WithTailscaleInfo(ctx, tsInfo)
next.ServeHTTP(w, r.WithContext(ctx))
```

This replaces the existing last 2 lines:
```go
ctx := WithUser(r.Context(), &user)
next.ServeHTTP(w, r.WithContext(ctx))
```

Also add `"strings"` to the import block.

**Step 2: Verify it compiles**

Run: `GOWORK=off go build ./internal/middleware/`
Expected: success, no errors

**Step 3: Commit**

```bash
git add internal/middleware/tailscale.go
git commit -m "feat: extract TailscaleInfo from WhoIs in auth middleware"
```

---

### Task 4: Write failing test for GetMe with TailscaleInfo

**Files:**
- Modify: `internal/handler/handlers_test.go`

**Step 1: Add `/me` route to `setupFullRouter`**

In `setupFullRouter()`, add the user handler and route. After `export := NewExportHandler(db)` (line 39), add:

```go
user := NewUserHandler()
```

Inside the `r.Route("/api/v1", ...)` block, after the `/export` line (line 64), add:

```go
r.Get("/me", user.GetMe)
```

**Step 2: Write the test**

Add at the end of `handlers_test.go`:

```go
func TestGetMe(t *testing.T) {
	router := setupFullRouter(t)

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("get me: got %d, want 200. body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	// Verify basic user fields
	if resp["display_name"] != "Dev User" {
		t.Errorf("expected display_name 'Dev User', got %q", resp["display_name"])
	}
	if resp["tailscale_id"] != "dev-user" {
		t.Errorf("expected tailscale_id 'dev-user', got %q", resp["tailscale_id"])
	}

	// Verify tailscale_info is present
	tsInfo, ok := resp["tailscale_info"].(map[string]any)
	if !ok {
		t.Fatal("expected tailscale_info object in response")
	}
	if tsInfo["login_name"] != "dev@localhost" {
		t.Errorf("expected login_name 'dev@localhost', got %q", tsInfo["login_name"])
	}
	if tsInfo["display_name"] != "Dev User" {
		t.Errorf("expected ts display_name 'Dev User', got %q", tsInfo["display_name"])
	}
	if tsInfo["node_name"] != "fishscale.dev.ts.net" {
		t.Errorf("expected node_name 'fishscale.dev.ts.net', got %q", tsInfo["node_name"])
	}
}
```

**Step 3: Run test to verify it fails**

Run: `GOWORK=off go test ./internal/handler/ -run TestGetMe -v`
Expected: FAIL — `tailscale_info` not present in response (GetMe currently returns just the User struct)

**Step 4: Commit**

```bash
git add internal/handler/handlers_test.go
git commit -m "test: add failing test for GetMe with TailscaleInfo"
```

---

### Task 5: Update GetMe handler to include TailscaleInfo

**Files:**
- Modify: `internal/handler/user.go`

**Step 1: Add response struct and update GetMe**

Replace the contents of `user.go` with:

```go
package handler

import (
	"net/http"

	"github.com/allen/fishscale/internal/middleware"
	"github.com/allen/fishscale/internal/model"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

type getMeResponse struct {
	ID            int64                `json:"id"`
	TailscaleID   string               `json:"tailscale_id"`
	DisplayName   string               `json:"display_name"`
	CreatedAt     string               `json:"created_at"`
	TailscaleInfo *model.TailscaleInfo `json:"tailscale_info,omitempty"`
}

// GetMe returns the authenticated user's information
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp := getMeResponse{
		ID:            user.ID,
		TailscaleID:   user.TailscaleID,
		DisplayName:   user.DisplayName,
		CreatedAt:     user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		TailscaleInfo: middleware.TailscaleInfoFromContext(r.Context()),
	}

	jsonResponse(w, http.StatusOK, resp)
}
```

**Step 2: Run the test to verify it passes**

Run: `GOWORK=off go test ./internal/handler/ -run TestGetMe -v`
Expected: PASS

**Step 3: Run all tests to verify nothing broke**

Run: `GOWORK=off go test ./internal/handler/ -v`
Expected: all tests PASS

**Step 4: Commit**

```bash
git add internal/handler/user.go
git commit -m "feat: include TailscaleInfo in GetMe response"
```

---

### Task 6: Update Settings.svelte to show Account section

**Files:**
- Modify: `frontend/src/lib/pages/Settings.svelte`

**Step 1: Add account state and fetch**

In the `<script>` block, add after the existing imports:

```typescript
interface TailscaleInfo {
  login_name: string;
  display_name: string;
  tailscale_id: string;
  node_name: string;
  profile_pic_url?: string;
}

interface MeResponse {
  id: number;
  tailscale_id: string;
  display_name: string;
  created_at: string;
  tailscale_info?: TailscaleInfo;
}

let accountInfo: MeResponse | null = $state(null);

$effect(() => {
  api.me.get().then((data: MeResponse) => {
    accountInfo = data;
  });
});
```

**Step 2: Add Account card markup**

Add after `<h1 class="page-title">Settings</h1>` and before the Theme card:

```svelte
{#if accountInfo?.tailscale_info}
  <div class="card">
    <h2 class="section-title">Account</h2>
    <div class="info-grid">
      <span class="info-label">Display Name</span>
      <span class="info-value">{accountInfo.tailscale_info.display_name}</span>

      <span class="info-label">Login</span>
      <span class="info-value">{accountInfo.tailscale_info.login_name}</span>

      <span class="info-label">Device</span>
      <span class="info-value">{accountInfo.tailscale_info.node_name.split('.')[0]}</span>

      <span class="info-label">Tailnet URL</span>
      <span class="info-value">{accountInfo.tailscale_info.node_name}</span>
    </div>
  </div>
{/if}
```

**Step 3: Add styles**

Add inside the `<style>` block:

```css
.info-grid {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 8px 16px;
  font-size: 0.9rem;
}

.info-label {
  font-weight: 600;
  color: var(--text-secondary);
}

.info-value {
  word-break: break-all;
}
```

**Step 4: Verify frontend builds**

Run: `cd frontend && npm run check && npm run build`
Expected: no type errors, build succeeds

**Step 5: Commit**

```bash
git add frontend/src/lib/pages/Settings.svelte
git commit -m "feat: add Tailscale identity section to Settings page"
```

---

### Task 7: Run full CI and manual verification

**Step 1: Run make ci**

Run: `make ci`
Expected: all checks pass (Go tests, frontend lint, format check, type check, build). Note: Go lint step may fail because `golangci-lint` is not installed — this is expected and does not block.

**Step 2: Manual test**

Run: `FISHSCALE_DEV_MODE=true FISHSCALE_DB_PATH=./fish.db FISHSCALE_PHOTO_DIR=./photos GOWORK=off go run ./cmd/fishscale`

Open http://localhost:8080, go to Settings tab, verify the Account section appears at the top with:
- Display Name: Dev User
- Login: dev@localhost
- Device: fishscale
- Tailnet URL: fishscale.dev.ts.net

**Step 3: Update todo.md**

Add GH#2 to the Completed section in `docs/plans/todo.md`.

**Step 4: Final commit**

```bash
git add docs/plans/todo.md
git commit -m "docs: mark GH#2 Tailscale settings as implemented"
```
