# Intégrer ABMCY Core Payment

ABMCY Core est un **orchestrateur de paiement**. Votre application ne parle
jamais directement à un opérateur mobile money : elle appelle ABMCY Core,
qui gère la page de paiement, le reversement et les remboursements.

```
votre app  ──(clé + signature)──▶  ABMCY Core  ──▶  opérateur mobile money
    ▲                                   │
    └──────────(webhook signé)──────────┘
```

Base : `https://core.diarra.app`

---

## 1. Créer votre passerelle

Onglet **Applications** de la console → « Nouvelle application ». Vous
recevez, **une seule fois** :

| Valeur | Rôle |
|---|---|
| `api_key` | identifie votre app (en-tête `X-App-Key`) |
| `hmac_secret` | signe vos requêtes — **ne jamais l'exposer côté navigateur** |

Gardez-les côté serveur uniquement.

---

## 2. Signer une requête

Trois en-têtes sur chaque appel `/v1/*` :

| En-tête | Valeur |
|---|---|
| `X-App-Key` | votre `api_key` |
| `X-Abmcy-Timestamp` | horodatage Unix en secondes — fenêtre de validité : 5 min |
| `X-Abmcy-Signature` | `HMAC_SHA256( SHA256(hmac_secret) , "{timestamp}.{corps_json}" )` en hexadécimal |

> La clé HMAC est le **hash SHA-256** du secret, pas le secret brut.
> Le message signé est `timestamp + "." + corps`.

### Node.js

```js
import crypto from 'node:crypto';

const API_KEY = process.env.ABMCY_API_KEY;
const SECRET  = process.env.ABMCY_HMAC_SECRET;
const BASE    = 'https://core.diarra.app';

function signedFetch(path, bodyObj) {
  const body = JSON.stringify(bodyObj);
  const ts  = Math.floor(Date.now() / 1000).toString();
  const key = crypto.createHash('sha256').update(SECRET).digest('hex');
  const sig = crypto.createHmac('sha256', key).update(ts + '.' + body).digest('hex');
  return fetch(BASE + path, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-App-Key': API_KEY,
      'X-Abmcy-Timestamp': ts,
      'X-Abmcy-Signature': sig,
    },
    body,
  });
}
```

### PHP

```php
$body = json_encode($payload);
$ts   = (string) time();
$key  = hash('sha256', $secret);              // hash du secret
$sig  = hash_hmac('sha256', $ts . '.' . $body, $key);

$ch = curl_init("$base/v1/pay");
curl_setopt_array($ch, [
  CURLOPT_POST => true,
  CURLOPT_POSTFIELDS => $body,
  CURLOPT_RETURNTRANSFER => true,
  CURLOPT_HTTPHEADER => [
    'Content-Type: application/json',
    "X-App-Key: $apiKey",
    "X-Abmcy-Timestamp: $ts",
    "X-Abmcy-Signature: $sig",
  ],
]);
$res = curl_exec($ch);
```

### curl (test rapide)

```sh
BODY='{"app_ref":"cmd-1","amount_cfa":5000,"country":"SEN"}'
TS=$(date +%s)
KEY=$(printf '%s' "$SECRET" | sha256sum | cut -d' ' -f1)
SIG=$(printf '%s' "$TS.$BODY" | openssl dgst -sha256 -hmac "$KEY" | cut -d' ' -f2)

curl -X POST https://core.diarra.app/v1/pay \
  -H "X-App-Key: $API_KEY" \
  -H "X-Abmcy-Timestamp: $TS" \
  -H "X-Abmcy-Signature: $SIG" \
  -H "Content-Type: application/json" \
  -d "$BODY"
```

---

## 3. Créer un paiement — `POST /v1/pay`

```json
{
  "app_ref":      "commande-1234",       // VOTRE identifiant (unique par app)
  "amount_cfa":   5000,
  "country":      "SEN",
  "description":  "Abonnement 1 mois",   // optionnel
  "callback_url": "https://votre-app.com/webhooks/abmcy",  // optionnel (notif serveur)
  "return_url":   "https://votre-app.com/merci"            // optionnel (retour navigateur)
}
```

Réponse :

```json
{
  "payment": {
    "app_ref": "commande-1234",
    "status": "pending",
    "amount_cfa": 5000,
    "fee_cfa": 450,
    "net_cfa": 4550,
    "redirect_url": "https://paywith.…/…",
    "return_url": "https://votre-app.com/merci"
  },
  "hosted_pay_url": "https://core.diarra.app/pay/<ref>"
}
```

**Idempotence stricte** : réutiliser le même `app_ref` renvoie **toujours la
même transaction** — jamais de doublon, jamais un lien différent. Si le lien
de paiement a expiré et que le paiement est encore en attente, il est
régénéré automatiquement sans changer la transaction.

---

## 4. Encaisser — trois options

### a) Page hébergée *(le plus simple)*

Redirigez l'utilisateur vers `hosted_pay_url`. ABMCY Core affiche un
récapitulatif à votre nom, un bouton « Payer », puis renvoie l'utilisateur
vers votre `return_url` (avec `?status=completed&ref=…`).

### b) Redirection directe

Redirigez vous-même vers `redirect_url` (la page de l'opérateur).

### c) Widget JavaScript

```html
<script src="https://core.diarra.app/widget/abmcy-pay.js"></script>
<button id="pay">Payer 5 000 FCFA</button>
<script>
  AbmcyPay.mount('#pay', {
    // votre backend crée le paiement (étape 3) et renvoie hosted_pay_url :
    createPayment: async () => (await fetch('/api/creer-paiement', { method: 'POST' })).json(),
    onSuccess: (ref) => { window.location = '/merci?ref=' + ref; },
    onCancel:  ()    => { console.log('annulé'); },
  });
</script>
```

Le widget ouvre `hosted_pay_url` dans une fenêtre et écoute le résultat.
Le secret HMAC reste sur votre serveur.

---

## 5. Recevoir le résultat — webhook serveur

Si vous avez fourni `callback_url`, ABMCY Core y envoie un `POST` à chaque
changement de statut :

```
POST https://votre-app.com/webhooks/abmcy
X-Abmcy-Signature: <hmac hex du corps brut, clé = SHA256(votre hmac_secret)>

{ "app_ref": "commande-1234", "status": "completed", "amount_cfa": 5000, "failure_reason": null }
```

Vérifiez la signature :
`HMAC_SHA256( SHA256(hmac_secret) , corps_brut )` doit égaler l'en-tête.
Répondez `2xx` — sinon le relais est marqué « failed » et renvoyable depuis
l'onglet **Paiements** de la console.

Filet de sécurité : vous pouvez toujours interroger l'état via
`GET /v1/payments/<app_ref>` (mêmes en-têtes signés, corps vide).

---

## 6. Verser vers un numéro — `POST /v1/payouts`

```json
{
  "app_ref":            "payout-042",
  "amount_cfa":         25000,
  "recipient_phone":    "770000000",
  "recipient_operator": "ORANGE_SEN",
  "country":            "SEN",
  "callback_url":       "https://votre-app.com/webhooks/abmcy"
}
```

Le destinataire reçoit `amount_cfa` moins la commission. Même callback signé
que les paiements. Idempotent sur `app_ref`. La gestion du solde est de
votre ressort.

Codes opérateur : `ORANGE_SEN`, `FREE_SEN`, `WAVE_SEN`, `MOOV_BEN`,
`MTN_MOMO_BEN`, `MTN_MOMO_CIV`, `ORANGE_CIV`, `MTN_MOMO_CMR`, `AIRTEL_COD`,
`ORANGE_COD`, `VODACOM_MPESA_COD`, `AIRTEL_COG`, `MTN_MOMO_COG`,
`AIRTEL_GAB`, `MPESA_KEN`, `AIRTEL_RWA`, `MTN_MOMO_RWA`, `ORANGE_SLE`,
`AIRTEL_UGA`, `MTN_MOMO_UGA`, `AIRTEL_ZMB`, `MTN_MOMO_ZMB`, `ZAMTEL_ZMB`.

---

## 7. Rembourser — `POST /v1/refunds`

```json
{
  "deposit_app_ref": "commande-1234",     // le paiement à rembourser (statut completed)
  "refund_app_ref":  "remb-1234",          // optionnel (défaut: refund-<deposit_app_ref>)
  "callback_url":    "https://votre-app.com/webhooks/abmcy"
}
```

Remboursement **intégral**. La commission du paiement d'origine n'est pas
rendue. Réponse : un `payment` de `type: "refund"` avec son propre statut et
le même callback signé. Idempotent sur `refund_app_ref`.

---

## 8. Statuts

| status | signification |
|---|---|
| `pending` | créé, en attente de paiement |
| `processing` | paiement en cours côté opérateur |
| `completed` | payé — livrez la commande |
| `failed` | échec (voir `failure_reason`) |
| `cancelled` | annulé par l'utilisateur |

---

## 9. Limites & tarifs

| | Plafond par transaction |
|---|---|
| Sans vérification | **200 000 FCFA** |
| Compte vérifié (KYC) | **1 000 000 FCFA** |

Au-delà : `422 limit_exceeded`. La vérification se demande dans la console.

**Commission ABMCY Core** : 50 FCFA par dollar de transaction (retenue sur
le montant reversé). Frais opérateur mobile money en sus.

---

## 10. Erreurs courantes

| Code | Cause |
|---|---|
| `401 missing_app_key` / `invalid_app_key` | en-tête `X-App-Key` absent ou inconnu |
| `401 invalid_signature` | signature HMAC fausse, ou horodatage hors fenêtre 5 min |
| `422 limit_exceeded` | montant au-dessus du plafond KYC |
| `400 app_ref_amount_country_required` | champ obligatoire manquant |
| `502 payment_init_failed` | l'opérateur a refusé l'initialisation — réessayez |
