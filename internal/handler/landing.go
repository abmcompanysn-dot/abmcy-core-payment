package handler

import (
	"html/template"
	"net/http"
)

// landingTmpl — page d'accueil publique d'ABMCY Core Payment.
// RÈGLE : ne cite JAMAIS un agrégateur ni un partenaire technique. ABMCY
// Core se présente comme l'orchestrateur de paiement, point.
var landingTmpl = template.Must(template.New("landing").Parse(`<!doctype html>
<html lang="fr"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>ABMCY Core Payment — la passerelle de paiement mobile money pour l'Afrique</title>
<meta name="description" content="Encaissez par mobile money dans 12 pays africains avec une seule intégration. Page de paiement hébergée, widget, API. Reversements et remboursements inclus.">
<style>
:root{color-scheme:light}
*{box-sizing:border-box;margin:0;padding:0}
body{font:16px/1.65 -apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#0f1720;background:#fff}
a{color:inherit}
.wrap{max-width:980px;margin:0 auto;padding:0 22px}
header{border-bottom:1px solid #eceff2}
header .wrap{display:flex;align-items:center;justify-content:space-between;height:64px}
.brand{font-weight:800;font-size:18px;letter-spacing:-.01em}
.brand span{color:#2f6bff}
.nav a{margin-left:20px;font-size:14px;color:#48525e;text-decoration:none}
.nav a.cta{background:#2f6bff;color:#fff;padding:9px 16px;border-radius:8px;font-weight:600}
.hero{padding:76px 0 64px;text-align:center;background:linear-gradient(180deg,#f7f9fc,#fff)}
.hero h1{font-size:40px;line-height:1.15;letter-spacing:-.02em;max-width:760px;margin:0 auto 18px}
.hero p{font-size:19px;color:#48525e;max-width:620px;margin:0 auto 30px}
.btn{display:inline-block;background:#2f6bff;color:#fff;text-decoration:none;padding:14px 26px;border-radius:10px;font-weight:600;font-size:16px}
.btn.ghost{background:#fff;color:#0f1720;border:1px solid #d5dbe2;margin-left:10px}
section{padding:56px 0;border-top:1px solid #eceff2}
section h2{font-size:26px;letter-spacing:-.01em;margin-bottom:8px}
section .sub{color:#48525e;margin-bottom:30px}
.grid{display:grid;grid-template-columns:repeat(3,1fr);gap:22px}
.card{border:1px solid #eceff2;border-radius:12px;padding:22px}
.card h3{font-size:16px;margin-bottom:6px}
.card p{font-size:14px;color:#48525e}
.steps{counter-reset:s;display:grid;gap:16px;max-width:720px}
.steps li{list-style:none;display:flex;gap:14px;align-items:flex-start}
.steps li::before{counter-increment:s;content:counter(s);flex:0 0 28px;height:28px;border-radius:50%;background:#2f6bff;color:#fff;font-weight:700;font-size:14px;display:flex;align-items:center;justify-content:center}
.countries{display:flex;flex-wrap:wrap;gap:10px}
.chip{border:1px solid #dfe4ea;border-radius:999px;padding:6px 14px;font-size:13px;color:#3a444f}
.price{display:grid;grid-template-columns:repeat(3,1fr);gap:22px}
.price .card b{font-size:22px;display:block;margin-bottom:4px}
.price .card .small{font-size:12px;color:#6b7480}
table.fees{width:100%;border-collapse:collapse;font-size:14px;margin-top:8px}
table.fees td{padding:8px 6px;border-bottom:1px solid #eceff2}
table.fees td:last-child{text-align:right;font-weight:600}
footer{padding:40px 0;color:#6b7480;font-size:13px;text-align:center;border-top:1px solid #eceff2}
form{max-width:560px;display:grid;gap:12px}
form .row2{display:grid;grid-template-columns:1fr 1fr;gap:12px}
label{font-size:13px;color:#48525e;display:block;margin-bottom:4px}
input,textarea,select{width:100%;border:1px solid #d5dbe2;border-radius:8px;padding:10px 12px;font:inherit}
textarea{min-height:84px;resize:vertical}
button.submit{background:#2f6bff;color:#fff;border:none;border-radius:9px;padding:13px;font-weight:600;font-size:15px;cursor:pointer}
button.submit:disabled{opacity:.55}
.note{font-size:13px;padding:12px 14px;border-radius:8px}
.note.ok{background:#e9f8ef;color:#1c7a43}
.note.err{background:#fdecec;color:#b42318}
@media(max-width:760px){.grid,.price{grid-template-columns:1fr}.hero h1{font-size:30px}.form .row2{grid-template-columns:1fr}}
</style></head><body>
<header><div class="wrap">
  <div class="brand">ABMCY <span>Core</span> Payment</div>
  <nav class="nav">
    <a href="#comment">Comment ça marche</a>
    <a href="#tarifs">Tarifs</a>
    <a href="/console/login">Console</a>
    <a class="cta" href="#inscription">Créer ma passerelle</a>
  </nav>
</div></header>

<div class="hero"><div class="wrap">
  <h1>Encaissez par mobile money dans 12 pays africains, avec une seule intégration.</h1>
  <p>ABMCY Core Payment est l'orchestrateur qui gère la page de paiement, les
     reversements aux marchands et les remboursements. Vous branchez une fois,
     vous encaissez partout.</p>
  <a class="btn" href="#inscription">Créer ma passerelle</a>
  <a class="btn ghost" href="/console/login">Accéder à la console</a>
</div></div>

<section id="comment"><div class="wrap">
  <h2>Comment ça marche</h2>
  <p class="sub">Trois façons d'encaisser, selon votre besoin.</p>
  <div class="grid">
    <div class="card"><h3>Page de paiement hébergée</h3><p>Redirigez votre client
      vers une page prête à l'emploi à votre nom. Zéro code de paiement à écrire.</p></div>
    <div class="card"><h3>Widget JavaScript</h3><p>Un bouton « Payer » sur votre
      site ouvre le paiement en fenêtre. Le paiement se règle, votre page se met à jour.</p></div>
    <div class="card"><h3>API</h3><p>Créez un paiement depuis votre backend,
      recevez le résultat par webhook signé. Contrôle total.</p></div>
  </div>
  <ol class="steps" style="margin-top:34px">
    <li><div><b>Créez votre passerelle</b><br>Remplissez le formulaire ci-dessous. Après validation, vous recevez vos identifiants par email.</div></li>
    <li><div><b>Intégrez</b><br>Page hébergée, widget ou API — la documentation vous guide.</div></li>
    <li><div><b>Encaissez</b><br>Vos clients paient par mobile money. Vous êtes reversé, commission déduite.</div></li>
  </ol>
</div></section>

<section><div class="wrap">
  <h2>Pays &amp; opérateurs couverts</h2>
  <p class="sub">Dépôts et remboursements opérationnels partout ci-dessous.</p>
  <div class="countries">
    <span class="chip">Bénin — Moov, MTN</span>
    <span class="chip">Sénégal — Orange, Free, Wave</span>
    <span class="chip">Côte d'Ivoire — MTN, Orange</span>
    <span class="chip">Cameroun — MTN</span>
    <span class="chip">RD Congo — Airtel, Orange, M-Pesa</span>
    <span class="chip">Congo-Brazzaville — Airtel, MTN</span>
    <span class="chip">Gabon — Airtel</span>
    <span class="chip">Kenya — M-Pesa</span>
    <span class="chip">Rwanda — Airtel, MTN</span>
    <span class="chip">Sierra Leone — Orange</span>
    <span class="chip">Ouganda — Airtel, MTN</span>
    <span class="chip">Zambie — Airtel, MTN, Zamtel</span>
  </div>
</div></section>

<section id="tarifs"><div class="wrap">
  <h2>Tarifs &amp; limites</h2>
  <p class="sub">Simple : une commission par transaction, déduite du montant reversé.</p>
  <div class="price">
    <div class="card">
      <b>Sans vérification</b>
      <span class="small">Démarrage immédiat après validation</span>
      <p style="margin-top:10px">Jusqu'à <b style="font-size:16px">200 000 FCFA</b> par transaction.</p>
    </div>
    <div class="card">
      <b>Compte vérifié</b>
      <span class="small">Après envoi des justificatifs</span>
      <p style="margin-top:10px">Jusqu'à <b style="font-size:16px">1 000 000 FCFA</b> par transaction.</p>
    </div>
    <div class="card">
      <b>Volume &amp; intégration directe</b>
      <span class="small">Sur mesure</span>
      <p style="margin-top:10px">Au-delà, nous accompagnons votre entreprise à mettre en place sa propre intégration.</p>
    </div>
  </div>
  <table class="fees">
    <tr><td>Commission ABMCY Core</td><td>50 FCFA par dollar de transaction</td></tr>
    <tr><td>Frais opérateur mobile money</td><td>en sus, selon l'opérateur</td></tr>
  </table>
  <p style="font-size:13px;color:#6b7480;margin-top:10px">La commission est retenue automatiquement sur le montant reversé au marchand. Aucun frais fixe, aucun abonnement.</p>
</div></section>

<section id="inscription"><div class="wrap">
  <h2>Créer ma passerelle de paiement</h2>
  <p class="sub">Dites-nous en deux minutes ce que vous voulez encaisser. Nous revenons vers vous avec vos accès.</p>
  <form id="signup">
    <div class="row2">
      <div><label>Nom de l'entreprise / projet *</label><input name="business_name" required></div>
      <div><label>Votre nom *</label><input name="contact_name" required></div>
    </div>
    <div class="row2">
      <div><label>Email *</label><input name="email" type="email" required></div>
      <div><label>Téléphone</label><input name="phone"></div>
    </div>
    <div class="row2">
      <div><label>Site web</label><input name="website" type="url" placeholder="https://"></div>
      <div><label>Pays principal</label><input name="country" placeholder="Sénégal"></div>
    </div>
    <div><label>Que voulez-vous encaisser ?</label><textarea name="description" placeholder="Ex. abonnements, boutique en ligne, réservations…"></textarea></div>
    <div><label>Volume mensuel estimé</label><input name="expected_volume" placeholder="Ex. ~500 000 FCFA / mois"></div>
    <div id="msg"></div>
    <button class="submit" type="submit">Envoyer la demande</button>
  </form>
</div></section>

<footer>ABMCY Core Payment · <a href="/console/login">Console</a></footer>

<script>
var f = document.getElementById('signup'), msg = document.getElementById('msg');
f.addEventListener('submit', function (e) {
  e.preventDefault();
  var btn = f.querySelector('button'); btn.disabled = true; msg.innerHTML = '';
  var data = {};
  new FormData(f).forEach(function (v, k) { data[k] = v; });
  fetch('/public/signup', {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data)
  }).then(function (r) { return r.json().then(function (b) { return { ok: r.ok, b: b }; }); })
    .then(function (res) {
      if (res.ok) {
        msg.className = 'note ok';
        msg.textContent = res.b.message || 'Demande reçue. Nous revenons vers vous par email.';
        f.reset();
      } else {
        msg.className = 'note err';
        msg.textContent = res.b.error || 'Une erreur est survenue. Réessayez.';
      }
    }).catch(function () {
      msg.className = 'note err'; msg.textContent = 'Erreur réseau. Réessayez.';
    }).finally(function () { btn.disabled = false; });
});
</script>
</body></html>`))

// Landing — GET / : page d'accueil publique.
func (h *PaymentHandler) Landing(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_ = landingTmpl.Execute(w, nil)
}
