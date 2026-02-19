# Tailscale Identity on Settings Page

**GitHub Issue:** [#2](https://github.com/Valien/fishscale/issues/2)
**Date:** 2026-02-18

## Goal

Surface Tailscale identity and tailnet information on the Settings page so users can see who they are and what device/tailnet they're connected through.

## Approach

Expand the existing `GET /api/v1/me` endpoint to include Tailscale metadata extracted from the WhoIs response. No database migration required — the extra fields are transient (fresh from WhoIs on every request).

## Backend

### New model: `TailscaleInfo`

Added to `internal/model/models.go`:

```go
type TailscaleInfo struct {
    LoginName     string `json:"login_name"`
    DisplayName   string `json:"display_name"`
    TailscaleID   string `json:"tailscale_id"`
    NodeName      string `json:"node_name"`
    ProfilePicURL string `json:"profile_pic_url,omitempty"`
}
```

### Middleware changes

`internal/middleware/tailscale.go`: Extract `LoginName`, `ProfilePicURL`, and `Node.Name` from the WhoIs response. Store as `*model.TailscaleInfo` in the request context.

`internal/middleware/auth.go`: Add `WithTailscaleInfo` / `TailscaleInfoFromContext` context helpers. `DevAuth` sets placeholder TailscaleInfo.

### Handler changes

`internal/handler/user.go`: `GetMe` returns a combined response struct that includes the existing `User` fields plus `tailscale_info` from context.

### API response

`GET /api/v1/me`:

```json
{
  "id": 42,
  "tailscale_id": "u1234abc",
  "display_name": "Allen Smith",
  "created_at": "2026-02-17T...",
  "tailscale_info": {
    "login_name": "allen@example.com",
    "display_name": "Allen Smith",
    "tailscale_id": "u1234abc",
    "node_name": "fishscale.tail1234.ts.net",
    "profile_pic_url": "https://..."
  }
}
```

In dev mode, `tailscale_info` returns placeholder values (`dev@localhost`, `fishscale.dev.ts.net`, etc.).

## Frontend

New read-only "Account" section at the top of `Settings.svelte`, above the existing Theme section. Fields displayed:

- **Display Name** — from `tailscale_info.display_name`
- **Login** — from `tailscale_info.login_name`
- **Device** — first label of `tailscale_info.node_name` (before first dot)
- **Tailnet URL** — full `tailscale_info.node_name` with trailing dot stripped

Fetches from `/api/v1/me` on component mount. No edit controls — this is informational only.

## Testing

- Go test for `GetMe` verifying the response includes `tailscale_info` with dev mode values.
- Existing handler tests continue to pass (additive change only).

## Dev mode behavior

`DevAuth` middleware populates `TailscaleInfo` with sensible defaults so the Settings page renders correctly during local development.
