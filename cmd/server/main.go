// ABMCY Core — orchestrateur de paiement pour les applications ABMCY, qui
// passe PAR la passerelle DIARRA (POST /api/gateway/v1/*) au lieu
// d'intégrer PawaPay/KPay/PayPal directement. DIARRA reste la seule
// intégration agrégateur ; ABMCY Core n'en connaît jamais les détails.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"os"

	"github.com/abmcy/core/internal/gateway"
	"github.com/abmcy/core/internal/handler"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func main() {
	diarraBaseURL := os.Getenv("DIARRA_GATEWAY_URL")
	if diarraBaseURL == "" {
		diarraBaseURL = "http://localhost:8080" // backend DIARRA en local par défaut
	}
	apiKey := os.Getenv("DIARRA_GATEWAY_API_KEY")
	hmacSecret := os.Getenv("DIARRA_GATEWAY_HMAC_SECRET")
	if apiKey == "" || hmacSecret == "" {
		log.Fatal("DIARRA_GATEWAY_API_KEY et DIARRA_GATEWAY_HMAC_SECRET sont requis (voir README pour les obtenir via /api/admin/gateway/clients côté DIARRA)")
	}
	// DIARRA vérifie contre le HASH du secret, jamais le secret brut (voir
	// backend/internal/middleware/gateway.go, verifyGatewaySignature) — on
	// calcule donc ce hash une fois au démarrage plutôt qu'à chaque requête.
	sum := sha256.Sum256([]byte(hmacSecret))
	hmacSecretHash := hex.EncodeToString(sum[:])

	diarra := gateway.New(gateway.Config{
		BaseURL:        diarraBaseURL,
		APIKey:         apiKey,
		HMACSecretHash: hmacSecretHash,
	})
	paymentHandler := handler.NewPaymentHandler(diarra, hmacSecretHash)

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Post("/pay", paymentHandler.Pay)
	r.Get("/payments/{client_ref}", paymentHandler.PaymentStatus)
	r.Post("/webhooks/diarra", paymentHandler.DiarraCallback)

	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}
	log.Printf("ABMCY Core démarré sur le port %s (passerelle DIARRA: %s)", port, diarraBaseURL)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
