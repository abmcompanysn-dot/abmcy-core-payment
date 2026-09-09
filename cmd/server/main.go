// ABMCY Core — orchestrateur de paiement MULTI-APPLICATIONS pour les
// applications ABMCY. Plusieurs apps clientes (une boutique, une app de
// réservation...) intègrent son widget/SDK ; ABMCY Core route chaque
// paiement PAR la passerelle DIARRA (POST /api/gateway/v1/*) au lieu
// d'intégrer PawaPay/KPay/PayPal directement. DIARRA reste la seule
// intégration agrégateur ; ABMCY Core n'en connaît jamais les détails.
//
// Trois niveaux, chacun avec sa propre paire clé/secret (jamais partagée) :
//
//	app cliente ──X-App-Key + X-Abmcy-Signature──▶ ABMCY Core
//	ABMCY Core  ──X-Gateway-Key + X-Diarra-Signature──▶ passerelle DIARRA
//	DIARRA      ──▶ agrégateur (PawaPay/KPay/PayPal)
//
// et les callbacks remontent le chemin inverse, resignés à chaque étage.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"os"

	"github.com/abmcy/core/internal/db"
	"github.com/abmcy/core/internal/gateway"
	"github.com/abmcy/core/internal/handler"
	"github.com/abmcy/core/internal/middleware"
	"github.com/abmcy/core/internal/repository"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func main() {
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL est requis (base abmcy_core sur le Postgres de diarra-vps)")
	}
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connexion Postgres: %v", err)
	}
	defer pool.Close()

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
	// Ce même hash sert de clé HMAC pour signer nos requêtes SORTANTES vers
	// DIARRA et pour vérifier les callbacks ENTRANTS de DIARRA.
	sum := sha256.Sum256([]byte(hmacSecret))
	diarraHMACSecretHash := hex.EncodeToString(sum[:])

	diarra := gateway.New(gateway.Config{
		BaseURL:        diarraBaseURL,
		APIKey:         apiKey,
		HMACSecretHash: diarraHMACSecretHash,
	})

	// URL publique de CE service, telle que DIARRA doit la rappeler. DIARRA
	// ne relaie un callback QUE si la transaction porte un callback_url
	// explicite (pas de fallback sur default_callback_url) — donc requis.
	selfCallbackURL := os.Getenv("ABMCY_CORE_CALLBACK_URL")
	if selfCallbackURL == "" {
		log.Fatal("ABMCY_CORE_CALLBACK_URL est requis (ex. https://core.diarra.app/webhooks/diarra) — DIARRA rappelle cette URL")
	}

	appRepo := repository.NewAppRepo(pool)
	paymentHandler := handler.NewPaymentHandler(appRepo, diarra, diarraHMACSecretHash, selfCallbackURL)
	adminHandler := handler.NewAdminHandler(appRepo, paymentHandler)

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	// API pour les apps clientes — chaque requête authentifiée par
	// X-App-Key + signature HMAC de l'app (voir middleware.RequireApp).
	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.RequireApp(paymentHandler.LookupApp))
		r.Post("/pay", paymentHandler.Pay)
		r.Get("/payments/{app_ref}", paymentHandler.PaymentStatus)
	})

	// Callback signé de la passerelle DIARRA (vérification HMAC dans le
	// handler, pas de middleware app ici — l'appelant est DIARRA, pas une app).
	r.Post("/webhooks/diarra", paymentHandler.DiarraCallback)

	// Page de paiement hébergée, ouverte par le navigateur de l'utilisateur
	// final (aucune auth). L'app ouvre hosted_pay_url (renvoyée par /v1/pay).
	r.Get("/pay/{ref}", paymentHandler.HostedPay)

	// SDK navigateur (statique, cacheable).
	r.Get("/widget/abmcy-pay.js", paymentHandler.WidgetJS)

	// Administration d'ABMCY Core : création/désactivation des apps clientes.
	// Jeton unique porté par ABMCY_ADMIN_TOKEN (voir middleware.RequireAdmin).
	adminToken := os.Getenv("ABMCY_ADMIN_TOKEN")
	r.Route("/admin", func(r chi.Router) {
		r.Use(middleware.RequireAdmin(adminToken))
		r.Get("/apps", adminHandler.ListApps)
		r.Post("/apps", adminHandler.CreateApp)
		r.Put("/apps/{id}/active", adminHandler.SetAppActive)
		r.Get("/payments", adminHandler.ListPayments)
		r.Get("/payments/{id}", adminHandler.GetPayment)
		r.Post("/payments/{id}/relay", adminHandler.RelayPayment)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}
	log.Printf("ABMCY Core démarré sur le port %s (passerelle DIARRA: %s)", port, diarraBaseURL)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
