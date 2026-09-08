# ABMCY Core Payment

Orchestrateur de paiement **multi-applications** pour l'écosystème ABMCY.

Plusieurs applications ABMCY (une boutique, une app de réservation…) intègrent
le widget/SDK ABMCY Core pour encaisser des paiements. ABMCY Core route chaque
paiement vers la **passerelle de paiement DIARRA** (`POST /api/gateway/v1/*`),
qui reste la seule intégration agrégateur réelle (PawaPay/KPay/PayPal).

ABMCY Core ne parle jamais directement à un agrégateur.

```
 app cliente A ─┐
 app cliente B ─┼─(X-App-Key + X-Abmcy-Signature)─▶ ABMCY Core ─(X-Gateway-Key + X-Diarra-Signature)─▶ DIARRA ─▶ PawaPay
 app cliente … ─┘                                        │  ▲                                            │  ▲
                                                         │  └───────────(callback signé)────────────────┘  │
                                                         └──────────(callback signé vers l'app)──────────────
```

Trois niveaux, **chacun avec sa propre paire clé API / secret HMAC**, jamais
partagée entre niveaux. Les callbacks remontent le chemin inverse, resignés à
chaque étage avec le secret propre au destinataire.

## Architecture

| Table | Rôle |
|---|---|
| `apps` | Une application cliente d'ABMCY Core (≈ `gateway_clients` côté DIARRA, un niveau au-dessus). `api_key_hash` / `hmac_secret_hash` jamais en clair. |
| `payments` | Un paiement initié par une app. Relie `app_ref` (identifiant chez l'app) à `diarra_client_ref` (généré ici, unique, utilisé pour parler à DIARRA). Tenu à jour par les callbacks DIARRA. |

## Configuration

```powershell
$env:DATABASE_URL              = "postgres://…@…:5432/abmcy_core?sslmode=disable"
$env:DIARRA_GATEWAY_URL        = "https://api.diarra.app"     # ou http://localhost:8080 en dev DIARRA local
$env:DIARRA_GATEWAY_API_KEY    = "<clé remise par l'admin DIARRA>"
$env:DIARRA_GATEWAY_HMAC_SECRET= "<secret HMAC remis par l'admin DIARRA>"
$env:ABMCY_CORE_CALLBACK_URL   = "https://core.diarra.app/webhooks/diarra"  # URL publique que DIARRA doit rappeler
$env:ABMCY_ADMIN_TOKEN         = "<jeton admin ABMCY Core>"   # pour /admin/* ; vide = /admin/* désactivé
$env:PORT                      = "9090"                       # optionnel

go run ./cmd/migrate    # applique migrations/*.sql (table schema_migrations)
go run ./cmd/server
```

> `ABMCY_CORE_CALLBACK_URL` est **obligatoire** : DIARRA ne relaie un callback
> que si la transaction porte un `callback_url` explicite (pas de repli sur le
> `default_callback_url` du client), donc ABMCY Core le passe sur chaque
> `CreateDeposit`.

## Obtenir la clé DIARRA

Côté admin DIARRA (scope finance) :

```
POST /api/admin/gateway/clients
{"name": "ABMCY Core", "default_callback_url": "https://core.diarra.app/webhooks/diarra"}
```

La réponse contient `api_key` et `hmac_secret` **une seule fois**. DIARRA ne
stocke que leurs hash SHA-256 — régénérer un nouveau client si perdus.

## Endpoints

### Pour les apps clientes (`/v1/*`, signés par l'app)

Auth : `X-App-Key` + `X-Abmcy-Timestamp` + `X-Abmcy-Signature` =
HMAC-SHA256(`hash(secret_app)`, `timestamp + "." + corps`), fenêtre anti-rejeu
5 min. Voir `internal/middleware/app_auth.go`.

- `POST /v1/pay` — `{"app_ref", "amount_cfa", "country", "description?", "callback_url?"}`
  → crée un `payments`, appelle DIARRA, renvoie `{"payment": {…, "redirect_url"}}`
  (page PawaPay hébergée à ouvrir pour l'utilisateur final). Idempotent par
  `(app, app_ref)`.
- `GET /v1/payments/{app_ref}` — état local du paiement (déjà à jour via les
  callbacks DIARRA, pas d'appel réseau).

### Callback entrant

- `POST /webhooks/diarra` — DIARRA relaie ici le statut final. Vérifie
  `X-Diarra-Gateway-Signature` (HMAC-SHA256 du corps brut, clé =
  `hash(secret_gateway_DIARRA)`), met à jour le `payments`, puis relaie un
  callback signé vers le `callback_url` de l'app d'origine.

### Admin ABMCY Core (`/admin/*`)

Auth : `Authorization: Bearer $ABMCY_ADMIN_TOKEN`.

- `GET /admin/apps` — liste les apps.
- `POST /admin/apps` — `{"name", "default_callback_url?"}` → crée une app,
  renvoie `api_key` + `hmac_secret` **une seule fois**.
- `PUT /admin/apps/{id}/active` — `{"active": true|false}`.

## Statut

Validé de bout en bout le **2026-09-08** contre la production :

- app créée via `/admin/apps` → requête `/v1/pay` signée → `payments` créé →
  dépôt réel via la passerelle DIARRA prod → `redirect_url` PawaPay renvoyée ;
- idempotence, rejet signature/clé invalides (401) ;
- callback `POST /webhooks/diarra` signé → `payments` passe `completed` →
  relais déclenché vers le `callback_url` de l'app.

Base `abmcy_core` provisionnée sur le Postgres de diarra-vps, migration `001`
appliquée. Reste : déploiement k3s + `core.diarra.app`, clé DIARRA de
production (le test utilise une clé provisoire à révoquer).
