package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/jmoiron/sqlx"
	"tailscale.com/client/local"

	"github.com/allen/fishscale/internal/model"
)

func TailscaleAuth(lc *local.Client, db *sqlx.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			whois, err := lc.WhoIs(r.Context(), r.RemoteAddr)
			if err != nil {
				slog.Error("tailscale WhoIs failed", "error", err, "remote_addr", r.RemoteAddr)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			tailscaleID := whois.UserProfile.ID.String()
			displayName := whois.UserProfile.DisplayName

			// Upsert user
			_, err = db.ExecContext(r.Context(), `INSERT INTO users (tailscale_id, display_name) VALUES (?, ?)
				ON CONFLICT(tailscale_id) DO UPDATE SET display_name = ?`,
				tailscaleID, displayName, displayName)
			if err != nil {
				slog.Error("upsert user failed", "error", err, "tailscale_id", tailscaleID)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			var user model.User
			err = db.GetContext(r.Context(), &user, "SELECT * FROM users WHERE tailscale_id = ?", tailscaleID)
			if err != nil {
				slog.Error("get user failed", "error", err, "tailscale_id", tailscaleID)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

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
		})
	}
}
