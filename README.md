# ABMCY Core

Orchestrateur de paiement pour les applications ABMCY. Ne parle jamais
directement à un agrégateur (PawaPay/KPay/PayPal) : tout passe par la
**passerelle de paiement DIARRA** (`POST /api/gateway/v1/*`), qui reste la
seule intégration agrégateur réelle.

```
ABMCY Core  --(clé API + signature HMAC)-->  DIARRA (passerelle)  -->  PawaPay
    ^                                              |
    +---------------(callback signé)----------------+
```

## Démarrer en local

```powershell
$env:DIARRA_GATEWAY_URL = "https://api.diarra.app"       # ou http://localhost:8080 en dev DIARRA local
$env:DIARRA_GATEWAY_API_KEY = "<clé remise par l'admin DIARRA>"
$env:DIARRA_GATEWAY_HMAC_SECRET = "<secret HMAC remis par l'admin DIARRA>"
$env:PORT = "9090"  # optionnel, 9090 par défaut

go run ./cmd/server
```

## Obtenir une clé API + secret HMAC

Côté admin DIARRA (scope finance) :

```
POST /api/admin/gateway/clients
{"name": "ABMCY Core", "default_callback_url": "https://core.diarra.app/webhooks/diarra"}
```

La réponse contient `api_key` et `hmac_secret` **une seule fois** — à copier
immédiatement dans les variables d'environnement ci-dessus. DIARRA ne stocke
que leurs hash SHA-256, il n'existe aucun moyen de les récupérer après coup
(régénérer un nouveau client si perdus).

## Endpoints

- `POST /pay` — `{"client_ref", "amount_cfa", "country", "description"}` →
  crée un dépôt via DIARRA, renvoie `redirect_url` (page PawaPay hébergée) à
  ouvrir pour l'utilisateur final.
- `GET /payments/{client_ref}` — relit le statut auprès de DIARRA.
- `POST /webhooks/diarra` — reçoit le relais de callback DIARRA (statut final
  d'une transaction), vérifie `X-Diarra-Gateway-Signature`.

## Sécurité

Chaque requête sortante vers DIARRA est signée : `X-Gateway-Key` (identifie
le client) + `X-Diarra-Signature`/`X-Diarra-Timestamp` (HMAC-SHA256 du corps,
fenêtre anti-rejeu de 5 min) — voir `internal/gateway/diarra_client.go`.

## Statut

Squelette validé de bout en bout le 2026-09-08 : création de dépôt réelle
contre l'API PawaPay via la passerelle DIARRA en production (`client_ref:
test-order-002`, `depositId` PawaPay confirmé). `DiarraCallback` trace
seulement l'événement pour l'instant (`TODO` dans
`internal/handler/payment_handler.go`) — la mise à jour d'un état métier réel
(commandes, soldes...) reste à construire selon le produit ABMCY visé.
