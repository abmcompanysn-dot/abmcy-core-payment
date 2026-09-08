// Package middleware — authentification des applications ABMCY clientes du
// widget/SDK de paiement. Même principe à deux couches que
// DIARRA backend/internal/middleware/gateway.go (X-Gateway-Key + signature
// HMAC), reproduit ici pour les apps qui parlent à ABMCY Core — chaque niveau
// de l'orchestration (app -> ABMCY Core -> DIARRA -> agrégateur) a sa propre
// paire clé/secret, jamais partagée entre niveaux.
package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"time"
)

type contextKey string

const AppIDKey contextKey = "app_id"

type AppLookup struct {
	ID             string
	HMACSecretHash string
}

const signatureMaxAge = 5 * time.Minute

// RequireApp authentifie une app cliente via X-App-Key (identité) +
// X-Abmcy-Signature/X-Abmcy-Timestamp (HMAC-SHA256(hash(secret), timestamp +
// "." + corps), comparaison en temps constant, fenêtre anti-rejeu 5 min).
func RequireApp(lookup func(ctx context.Context, apiKeyHash string) (AppLookup, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-App-Key")
			if key == "" {
				http.Error(w, `{"error":"missing_app_key"}`, http.StatusUnauthorized)
				return
			}
			sum := sha256.Sum256([]byte(key))
			keyHash := hex.EncodeToString(sum[:])
			app, err := lookup(r.Context(), keyHash)
			if err != nil || app.ID == "" {
				http.Error(w, `{"error":"invalid_app_key"}`, http.StatusUnauthorized)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, `{"error":"read_failed"}`, http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			if !verifySignature(app.HMACSecretHash, r, body) {
				http.Error(w, `{"error":"invalid_signature"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), AppIDKey, app.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func verifySignature(hmacSecretHash string, r *http.Request, body []byte) bool {
	if hmacSecretHash == "" {
		return false
	}
	sigHex := r.Header.Get("X-Abmcy-Signature")
	tsHeader := r.Header.Get("X-Abmcy-Timestamp")
	if sigHex == "" || tsHeader == "" {
		return false
	}
	ts, err := strconv.ParseInt(tsHeader, 10, 64)
	if err != nil {
		return false
	}
	age := time.Since(time.Unix(ts, 0))
	if age < 0 {
		age = -age
	}
	if age > signatureMaxAge {
		return false
	}
	mac := hmac.New(sha256.New, []byte(hmacSecretHash))
	mac.Write([]byte(tsHeader + "." + string(body)))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sigHex))
}

func GetAppID(ctx context.Context) string {
	if v, ok := ctx.Value(AppIDKey).(string); ok {
		return v
	}
	return ""
}
