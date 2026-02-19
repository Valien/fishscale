package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/allen/fishscale/internal/model"
)

type contextKey string

const userContextKey contextKey = "user"
const tsInfoContextKey contextKey = "tailscale_info"

// UserFromContext retrieves the authenticated user from the request context.
// Returns nil if no user is present.
func UserFromContext(ctx context.Context) *model.User {
	u, _ := ctx.Value(userContextKey).(*model.User)
	return u
}

// WithUser returns a new context with the given user attached.
func WithUser(ctx context.Context, user *model.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

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

// DevAuth is middleware that injects a fake development user into the request
// context. This is intended for local development only.
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
