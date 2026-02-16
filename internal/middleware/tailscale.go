package middleware

import (
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	"tailscale.com/client/tailscale"

	"github.com/allen/fishscale/internal/model"
)

func TailscaleAuth(lc *tailscale.LocalClient, db *sqlx.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			whois, err := lc.WhoIs(r.Context(), r.RemoteAddr)
			if err != nil {
				log.Printf("WhoIs error: %v", err)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			tailscaleID := whois.UserProfile.ID.String()
			displayName := whois.UserProfile.DisplayName

			// Upsert user
			_, err = db.Exec(`INSERT INTO users (tailscale_id, display_name) VALUES (?, ?)
				ON CONFLICT(tailscale_id) DO UPDATE SET display_name = ?`,
				tailscaleID, displayName, displayName)
			if err != nil {
				log.Printf("upsert user error: %v", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			var user model.User
			err = db.Get(&user, "SELECT * FROM users WHERE tailscale_id = ?", tailscaleID)
			if err != nil {
				log.Printf("get user error: %v", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			ctx := WithUser(r.Context(), &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
