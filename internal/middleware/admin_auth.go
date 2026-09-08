// Package middleware — authentification admin d'ABMCY Core lui-même
// (gestion des apps clientes). ABMCY Core est un petit service interne, pas
// un produit multi-utilisateurs comme DIARRA : un unique jeton admin porté
// par variable d'environnement (ABMCY_ADMIN_TOKEN) suffit, comparé en temps
// constant — pas besoin de la table users/RBAC complète de DIARRA ici.
package middleware

import (
	"crypto/subtle"
	"net/http"
)

// RequireAdmin vérifie l'en-tête Authorization: Bearer <token> contre le
// jeton admin configuré. token vide désactive toute requête admin (plutôt
// que d'accepter n'importe quel jeton) — fail-closed par défaut.
func RequireAdmin(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				http.Error(w, `{"error":"admin_disabled"}`, http.StatusServiceUnavailable)
				return
			}
			const prefix = "Bearer "
			auth := r.Header.Get("Authorization")
			if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix {
				http.Error(w, `{"error":"missing_admin_token"}`, http.StatusUnauthorized)
				return
			}
			supplied := auth[len(prefix):]
			if subtle.ConstantTimeCompare([]byte(supplied), []byte(token)) != 1 {
				http.Error(w, `{"error":"invalid_admin_token"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
