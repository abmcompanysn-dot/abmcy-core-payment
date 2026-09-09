'use client';

import { Shell } from '../shell';
import { BASE_PATH } from '@/lib/bp';

// Doc d'intégration statique — pas d'appel réseau, juste du contenu.
export default function DocsPage() {
  const origin = typeof window !== 'undefined' ? window.location.origin : 'https://core.diarra.app';

  return (
    <Shell>
      <div className="panel">
        <h2>Intégrer une application à ABMCY Core Payment</h2>
        <p className="muted">
          ABMCY Core est un <strong>orchestrateur</strong> : votre application ne parle jamais
          directement à un agrégateur (PawaPay, etc.). Elle appelle ABMCY Core, qui route le paiement
          via la passerelle DIARRA.
        </p>
        <pre className="mono block">
{`votre app  ──(clé + signature HMAC)──▶  ABMCY Core  ──▶  passerelle DIARRA  ──▶  PawaPay
    ▲                                        │
    └──────────(callback signé)──────────────┘`}
        </pre>
      </div>

      <div className="panel">
        <h2>1. Créer votre application</h2>
        <p>
          Onglet <a href={`${BASE_PATH}/apps`}>Applications</a> → « Nouvelle application ». Vous
          obtenez, <strong>une seule fois</strong> :
        </p>
        <ul>
          <li>
            <code>api_key</code> — identifie votre app (en-tête <code>X-App-Key</code>)
          </li>
          <li>
            <code>hmac_secret</code> — sert à signer vos requêtes. <strong>Ne jamais l’exposer</strong>{' '}
            côté navigateur.
          </li>
        </ul>
      </div>

      <div className="panel">
        <h2>2. Signer une requête</h2>
        <p>
          Deux en-têtes d’authentification sur chaque appel <code>/v1/*</code> :
        </p>
        <ul>
          <li>
            <code>X-App-Key</code> : votre <code>api_key</code>
          </li>
          <li>
            <code>X-Abmcy-Timestamp</code> : horodatage Unix (secondes). Fenêtre : 5 min.
          </li>
          <li>
            <code>X-Abmcy-Signature</code> :{' '}
            <code>HMAC_SHA256( SHA256(hmac_secret) , "{'{'}timestamp{'}'}.{'{'}corps_json{'}'}" )</code> en
            hexadécimal.
          </li>
        </ul>
        <p className="muted">
          La clé HMAC est le <strong>hash SHA-256</strong> du secret, pas le secret brut. Le message
          signé est <code>timestamp + "." + corps</code>.
        </p>

        <h3>Node.js</h3>
        <pre className="mono block">
{`import crypto from 'node:crypto';

const API_KEY = process.env.ABMCY_API_KEY;
const SECRET  = process.env.ABMCY_HMAC_SECRET;
const BASE    = '${origin}';

function signedFetch(path, bodyObj) {
  const body = JSON.stringify(bodyObj);
  const ts = Math.floor(Date.now() / 1000).toString();
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
}`}
        </pre>

        <h3>PHP</h3>
        <pre className="mono block">
{`$body = json_encode($payload);
$ts   = (string) time();
$key  = hash('sha256', $secret);            // hash du secret
$sig  = hash_hmac('sha256', $ts.'.'.$body, $key);

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
$res = curl_exec($ch);`}
        </pre>
      </div>

      <div className="panel">
        <h2>3. Créer un paiement</h2>
        <p>
          <code>POST /v1/pay</code>
        </p>
        <pre className="mono block">
{`{
  "app_ref":      "commande-1234",       // votre identifiant (unique par app)
  "amount_cfa":   5000,
  "country":      "SEN",
  "description":  "Abonnement 1 mois",   // optionnel
  "callback_url": "https://votre-app.com/webhooks/abmcy",  // optionnel (notif serveur)
  "return_url":   "https://votre-app.com/merci"            // optionnel (retour navigateur)
}`}
        </pre>
        <p>Réponse :</p>
        <pre className="mono block">
{`{
  "payment": {
    "app_ref": "commande-1234",
    "status": "pending",
    "amount_cfa": 5000,
    "redirect_url": "https://paywith.pawapay.io/...",   // page PawaPay
    ...
  },
  "hosted_pay_url": "${origin}/pay/&lt;ref&gt;"            // page hébergée par ABMCY Core
}`}
        </pre>
        <p className="muted">
          <code>app_ref</code> est idempotent : réutiliser le même renvoie le paiement existant sans
          en recréer.
        </p>
      </div>

      <div className="panel">
        <h2>4. Encaisser — trois options</h2>

        <h3>a) Page hébergée (le plus simple)</h3>
        <p>
          Redirigez l’utilisateur vers <code>hosted_pay_url</code>. ABMCY Core affiche un récapitulatif
          (montant, marque de votre app) et un bouton « Payer » qui ouvre PawaPay. Après paiement,
          l’utilisateur est renvoyé vers votre <code>return_url</code> (avec{' '}
          <code>?status=completed&amp;ref=…</code>).
        </p>

        <h3>b) Redirection directe</h3>
        <p>
          Redirigez vous-même l’utilisateur vers <code>redirect_url</code> (la page PawaPay).
          <code>return_url</code> n’est alors pas utilisé — c’est PawaPay/DIARRA qui gère le retour.
        </p>

        <h3>c) Widget JavaScript</h3>
        <pre className="mono block">
{`<script src="${origin}/widget/abmcy-pay.js"></script>
<button id="pay">Payer 5 000 FCFA</button>
<script>
  AbmcyPay.mount('#pay', {
    // Votre backend crée le paiement (étape 3) et renvoie hosted_pay_url :
    createPayment: async () => {
      const r = await fetch('/api/creer-paiement', { method: 'POST' });
      return r.json();          // { hosted_pay_url: "..." }
    },
    onSuccess: (ref) => { window.location = '/merci?ref=' + ref; },
    onCancel:  ()   => { console.log('paiement annulé'); },
  });
</script>`}
        </pre>
        <p className="muted">
          Le widget ouvre <code>hosted_pay_url</code> dans une fenêtre et écoute le retour. Le secret
          HMAC reste sur votre serveur — le widget n’en a pas besoin.
        </p>
      </div>

      <div className="panel">
        <h2>5. Recevoir le résultat (callback serveur)</h2>
        <p>
          Si vous avez fourni <code>callback_url</code>, ABMCY Core y envoie un{' '}
          <code>POST</code> à chaque changement de statut :
        </p>
        <pre className="mono block">
{`POST https://votre-app.com/webhooks/abmcy
X-Abmcy-Signature: <hmac hex du corps brut, clé = SHA256(votre hmac_secret)>

{ "app_ref": "commande-1234", "status": "completed", "amount_cfa": 5000, "failure_reason": null }`}
        </pre>
        <p>
          Vérifiez la signature :{' '}
          <code>HMAC_SHA256( SHA256(hmac_secret) , corps_brut )</code> doit égaler l’en-tête. Répondez{' '}
          <code>2xx</code> — sinon ABMCY Core marque le relais « failed » (renvoyable depuis l’onglet{' '}
          <a href={`${BASE_PATH}/payments`}>Paiements</a>).
        </p>
        <p className="muted">
          Filet de sécurité : vous pouvez toujours interroger l’état via{' '}
          <code>GET /v1/payments/&lt;app_ref&gt;</code> (mêmes en-têtes signés, corps vide).
        </p>
      </div>

      <div className="panel">
        <h2>6. Rembourser un paiement</h2>
        <p>
          <code>POST /v1/refunds</code> (mêmes en-têtes signés). Rembourse{' '}
          <strong>intégralement</strong> un paiement au statut <code>completed</code>. La commission
          ABMCY Core n’est pas rendue.
        </p>
        <pre className="mono block">
{`{
  "deposit_app_ref": "commande-1234",     // le paiement à rembourser
  "refund_app_ref":  "remb-1234",          // optionnel (défaut: refund-<deposit_app_ref>)
  "callback_url":    "https://votre-app.com/webhooks/abmcy"  // optionnel
}`}
        </pre>
        <p>
          Réponse : un <code>payment</code> de <code>type: "refund"</code>. Son statut évolue
          (<code>processing</code> → <code>completed</code>) et déclenche le même callback signé que
          les paiements. Idempotent sur <code>refund_app_ref</code>.
        </p>
      </div>

      <div className="panel">
        <h2>Statuts</h2>
        <table>
          <thead>
            <tr>
              <th>status</th>
              <th>signification</th>
            </tr>
          </thead>
          <tbody>
            <tr><td className="mono">pending</td><td>créé, en attente de paiement</td></tr>
            <tr><td className="mono">processing</td><td>paiement en cours côté agrégateur</td></tr>
            <tr><td className="mono">completed</td><td>payé — livrez la commande</td></tr>
            <tr><td className="mono">failed</td><td>échec (voir failure_reason)</td></tr>
            <tr><td className="mono">cancelled</td><td>annulé par l’utilisateur</td></tr>
          </tbody>
        </table>
      </div>
    </Shell>
  );
}
