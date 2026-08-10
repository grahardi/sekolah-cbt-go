package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/grahardi/sekolah-cbt-go/internal/auth"
	"github.com/grahardi/sekolah-cbt-go/internal/httpx"
)

type ctxKey int

const (
	pesertaClaimsKey ctxKey = iota
	adminClaimsKey
)

// RequirePeserta validates the Bearer JWT issued by /peserta/login and
// injects its claims into the request context for downstream handlers.
func (h *Handlers) RequirePeserta(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || tokenStr == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		claims, err := auth.ParsePesertaToken(h.JWTSecret, tokenStr)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), pesertaClaimsKey, claims)
		next(w, r.WithContext(ctx))
	}
}

func pesertaClaimsFrom(r *http.Request) *auth.PesertaClaims {
	claims, _ := r.Context().Value(pesertaClaimsKey).(*auth.PesertaClaims)
	return claims
}

// RequireAdmin validates a Bearer JWT issued by Laravel (typ=admin,
// sekolah_id must match this instance) and injects its claims into the
// request context. This never issues tokens itself — only verifies.
func (h *Handlers) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || tokenStr == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		claims, err := auth.ParseAdminToken(h.JWTSecret, tokenStr)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		if claims.SekolahID != h.SekolahID {
			httpx.WriteError(w, http.StatusForbidden, "token bukan untuk sekolah ini")
			return
		}

		ctx := context.WithValue(r.Context(), adminClaimsKey, claims)
		next(w, r.WithContext(ctx))
	}
}

func adminClaimsFrom(r *http.Request) *auth.AdminClaims {
	claims, _ := r.Context().Value(adminClaimsKey).(*auth.AdminClaims)
	return claims
}
