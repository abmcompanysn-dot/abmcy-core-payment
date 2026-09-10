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
*{box-sizing:border-box;margin:0;padding:0}
:root{
  --ink:#0b1220;--muted:#5a6474;--line:#e7eaf0;--bg:#ffffff;
  --brand:#2f6bff;--brand-2:#00c2a8;--soft:#f6f8fc;
}
html{scroll-behavior:smooth}
body{font:16px/1.65 -apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:var(--ink);background:var(--bg);-webkit-font-smoothing:antialiased}
a{color:inherit;text-decoration:none}
.wrap{max-width:1080px;margin:0 auto;padding:0 24px}
.btn{display:inline-flex;align-items:center;gap:8px;background:var(--brand);color:#fff;padding:13px 22px;border-radius:10px;font-weight:600;font-size:15px;transition:transform .15s,box-shadow .15s;box-shadow:0 6px 20px rgba(47,107,255,.28)}
.btn:hover{transform:translateY(-1px);box-shadow:0 10px 28px rgba(47,107,255,.36)}
.btn.ghost{background:#fff;color:var(--ink);border:1px solid var(--line);box-shadow:none}
.btn.ghost:hover{border-color:#c9d2e3}

/* header */
header{position:sticky;top:0;z-index:50;background:rgba(255,255,255,.82);backdrop-filter:saturate(180%) blur(12px);border-bottom:1px solid var(--line)}
header .wrap{display:flex;align-items:center;justify-content:space-between;height:66px}
.brand{font-weight:800;font-size:18px;letter-spacing:-.02em;display:flex;align-items:center;gap:9px}
.brand .dot{width:10px;height:10px;border-radius:50%;background:var(--brand);box-shadow:0 0 0 4px rgba(47,107,255,.16)}
.brand b{color:var(--brand)}
.nav{display:flex;align-items:center;gap:22px}
.nav a{font-size:14px;color:var(--muted)}
.nav a:hover{color:var(--ink)}
@media(max-width:720px){.nav a:not(.cta){display:none}}

/* hero */
.hero{padding:72px 0 40px;background:radial-gradient(1200px 400px at 50% -80px,#eef3ff,transparent)}
.hero .wrap{display:grid;grid-template-columns:1.05fr .95fr;gap:48px;align-items:center}
.hero h1{font-size:44px;line-height:1.1;letter-spacing:-.025em;margin-bottom:18px}
.hero h1 .grad{background:linear-gradient(90deg,var(--brand),var(--brand-2));-webkit-background-clip:text;background-clip:text;color:transparent}
.hero p.lead{font-size:19px;color:var(--muted);margin-bottom:28px;max-width:520px}
.hero .cta-row{display:flex;gap:12px;flex-wrap:wrap}
.trust{margin-top:26px;font-size:13px;color:var(--muted);display:flex;gap:16px;flex-wrap:wrap}
.trust span{display:flex;align-items:center;gap:6px}
.trust .ok{width:16px;height:16px;border-radius:50%;background:rgba(0,194,168,.16);display:grid;place-items:center;flex:0 0 16px}
.trust .ok svg{width:10px;height:10px;stroke:var(--brand-2);stroke-width:2.4;fill:none;stroke-linecap:round;stroke-linejoin:round}
@media(max-width:860px){.hero .wrap{grid-template-columns:1fr;gap:36px}.hero h1{font-size:34px}}

/* animated diagram */
.diagram{position:relative;background:linear-gradient(180deg,#fff,#fbfcff);border:1px solid var(--line);border-radius:20px;padding:26px;box-shadow:0 24px 60px rgba(11,18,32,.08)}
.diagram .lane{display:flex;align-items:center;justify-content:space-between;gap:14px}
.node{flex:1;border:1px solid var(--line);border-radius:14px;padding:14px 12px;text-align:center;background:#fff;position:relative;z-index:2}
.node .ic{width:38px;height:38px;margin:0 auto 8px;border-radius:10px;display:grid;place-items:center;background:var(--soft)}
.node .t{font-weight:700;font-size:13px}
.node .s{font-size:11px;color:var(--muted);margin-top:2px}
.node.core{border-color:rgba(47,107,255,.4);box-shadow:0 0 0 4px rgba(47,107,255,.1)}
.node.core .ic{background:linear-gradient(135deg,var(--brand),var(--brand-2))}
.wire{height:2px;flex:0 0 34px;background:linear-gradient(90deg,var(--line),var(--line));position:relative;overflow:hidden;border-radius:2px}
.wire::after{content:"";position:absolute;top:0;left:-40%;width:40%;height:100%;background:linear-gradient(90deg,transparent,var(--brand),transparent);animation:flow 2.4s linear infinite}
.wire.w2::after{animation-delay:.8s;background:linear-gradient(90deg,transparent,var(--brand-2),transparent)}
.wire.w3::after{animation-delay:1.6s}
@keyframes flow{to{left:120%}}
.diagram .cap{margin-top:16px;font-size:12px;color:var(--muted);text-align:center}
.pulse{position:absolute;inset:0;border-radius:20px;pointer-events:none}
.pulse::before{content:"";position:absolute;left:50%;top:50%;width:10px;height:10px;border-radius:50%;background:var(--brand);transform:translate(-50%,-50%);box-shadow:0 0 0 0 rgba(47,107,255,.35);animation:ping 2.8s ease-out infinite}
@keyframes ping{0%{box-shadow:0 0 0 0 rgba(47,107,255,.30)}70%{box-shadow:0 0 0 90px rgba(47,107,255,0)}100%{box-shadow:0 0 0 0 rgba(47,107,255,0)}}
@media(prefers-reduced-motion:reduce){.wire::after,.pulse::before{animation:none}}

/* sections */
section{padding:64px 0;border-top:1px solid var(--line)}
.eyebrow{font-size:12px;font-weight:700;letter-spacing:.12em;text-transform:uppercase;color:var(--brand)}
section h2{font-size:28px;letter-spacing:-.02em;margin:8px 0 6px}
section .sub{color:var(--muted);margin-bottom:32px;max-width:640px}
.grid3{display:grid;grid-template-columns:repeat(3,1fr);gap:20px}
.card{border:1px solid var(--line);border-radius:16px;padding:22px;transition:transform .15s,box-shadow .15s}
.card:hover{transform:translateY(-2px);box-shadow:0 16px 40px rgba(11,18,32,.08)}
.card .ic{width:40px;height:40px;border-radius:11px;background:var(--soft);display:grid;place-items:center;margin-bottom:12px}
.ic svg{width:20px;height:20px;stroke:var(--brand);stroke-width:1.6;fill:none;stroke-linecap:round;stroke-linejoin:round}
.node .ic svg{width:19px;height:19px;stroke:var(--brand)}
.node.core .ic svg{stroke:#fff}
.cc{display:inline-flex;align-items:center;justify-content:center;min-width:26px;height:18px;padding:0 5px;margin-right:8px;border-radius:5px;background:var(--soft);font-size:10px;font-weight:800;letter-spacing:.04em;color:var(--muted);vertical-align:1px}
.card h3{font-size:16px;margin-bottom:6px}
.card p{font-size:14px;color:var(--muted)}
.steps{display:grid;gap:18px;max-width:760px}
.steps li{list-style:none;display:flex;gap:16px;align-items:flex-start}
.steps li .n{flex:0 0 30px;height:30px;border-radius:50%;background:linear-gradient(135deg,var(--brand),var(--brand-2));color:#fff;font-weight:800;font-size:14px;display:grid;place-items:center}
.chips{display:flex;flex-wrap:wrap;gap:10px}
.chip{border:1px solid var(--line);border-radius:999px;padding:7px 14px;font-size:13px;color:#3a444f;background:#fff;transition:border-color .15s}
.chip:hover{border-color:var(--brand)}
.price{display:grid;grid-template-columns:repeat(3,1fr);gap:20px}
.price .card b.k{font-size:13px;color:var(--muted);font-weight:700;text-transform:uppercase;letter-spacing:.06em}
.price .card .big{font-size:24px;font-weight:800;margin:8px 0 4px;letter-spacing:-.02em}
.fees{width:100%;border-collapse:collapse;font-size:14px;margin-top:14px}
.fees td{padding:10px 8px;border-bottom:1px solid var(--line)}
.fees td:last-child{text-align:right;font-weight:600}
@media(max-width:820px){.grid3,.price{grid-template-columns:1fr}}

/* form */
.formwrap{background:linear-gradient(180deg,var(--soft),#fff);border:1px solid var(--line);border-radius:20px;padding:30px}
form{display:grid;gap:14px;max-width:620px}
.r2{display:grid;grid-template-columns:1fr 1fr;gap:14px}
label{font-size:13px;color:var(--muted);display:block;margin-bottom:5px;font-weight:600}
input,textarea{width:100%;border:1px solid var(--line);border-radius:10px;padding:11px 13px;font:inherit;background:#fff;transition:border-color .15s,box-shadow .15s}
input:focus,textarea:focus{outline:0;border-color:var(--brand);box-shadow:0 0 0 4px rgba(47,107,255,.12)}
textarea{min-height:88px;resize:vertical}
button.submit{background:var(--brand);color:#fff;border:0;border-radius:11px;padding:14px;font-weight:700;font-size:15px;cursor:pointer;box-shadow:0 8px 22px rgba(47,107,255,.3)}
button.submit:disabled{opacity:.55;box-shadow:none}
.note{font-size:14px;padding:12px 14px;border-radius:10px}
.note.ok{background:rgba(0,194,168,.12);color:#0a7d6c}
.note.err{background:#fdecec;color:#b42318}
@media(max-width:640px){.r2{grid-template-columns:1fr}.hero h1{font-size:30px}}

footer{padding:44px 0;color:var(--muted);font-size:13px;text-align:center;border-top:1px solid var(--line)}
</style></head><body>

<header><div class="wrap">
  <div class="brand"><span class="dot"></span>ABMCY <b>Core</b> Payment</div>
  <nav class="nav">
    <a href="#comment">Comment ça marche</a>
    <a href="#pays">Couverture</a>
    <a href="#tarifs">Tarifs</a>
    <a href="/console/login">Console</a>
    <a class="btn cta" href="#inscription">Créer ma passerelle</a>
  </nav>
</div></header>

<div class="hero"><div class="wrap">
  <div>
    <h1>Encaissez par mobile money partout en Afrique, avec <span class="grad">une seule intégration</span>.</h1>
    <p class="lead">ABMCY Core Payment est l'orchestrateur qui gère la page de paiement,
      le reversement aux marchands et les remboursements. Vous branchez une fois,
      vous encaissez dans 12 pays.</p>
    <div class="cta-row">
      <a class="btn" href="#inscription">Créer ma passerelle
        <svg viewBox="0 0 24 24" style="width:16px;height:16px;stroke:#fff;stroke-width:2;fill:none;stroke-linecap:round;stroke-linejoin:round"><path d="M5 12h14M13 6l6 6-6 6"/></svg></a>
      <a class="btn ghost" href="/console/login">Accéder à la console</a>
    </div>
    <div class="trust">
      <span><span class="ok"><svg viewBox="0 0 24 24"><path d="M4 12l6 6L20 6"/></svg></span> Aucun frais fixe</span>
      <span><span class="ok"><svg viewBox="0 0 24 24"><path d="M4 12l6 6L20 6"/></svg></span> Reversement automatique</span>
      <span><span class="ok"><svg viewBox="0 0 24 24"><path d="M4 12l6 6L20 6"/></svg></span> Remboursements inclus</span>
    </div>
  </div>

  <div class="diagram" aria-hidden="true">
    <div class="pulse"></div>
    <div class="lane">
      <div class="node">
        <div class="ic"><svg viewBox="0 0 24 24"><path d="M3 9h18l-1.5 10.5A2 2 0 0 1 17.5 21h-11a2 2 0 0 1-2-1.5L3 9Z"/><path d="M8 9V6a4 4 0 0 1 8 0v3"/></svg></div>
        <div class="t">Votre application</div><div class="s">boutique, app…</div>
      </div>
      <div class="wire w1"></div>
      <div class="node core">
        <div class="ic"><svg viewBox="0 0 24 24"><rect x="4" y="10" width="16" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/><path d="M12 15v2"/></svg></div>
        <div class="t" style="color:#fff">ABMCY Core</div><div class="s" style="color:#e7efff">orchestrateur sécurisé</div>
      </div>
      <div class="wire w2"></div>
      <div class="node">
        <div class="ic"><svg viewBox="0 0 24 24"><rect x="6" y="2" width="12" height="20" rx="2.5"/><path d="M11 18h2"/></svg></div>
        <div class="t">Mobile money</div><div class="s">12 pays</div>
      </div>
    </div>
    <div class="lane" style="margin-top:14px">
      <div class="node" style="opacity:.9">
        <div class="ic"><svg viewBox="0 0 24 24"><path d="M6 2h9l5 5v13a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V3a1 1 0 0 1 1-1Z"/><path d="M14 2v6h6M9 13h6M9 17h6"/></svg></div>
        <div class="t">Page de paiement</div><div class="s">hébergée à votre nom</div>
      </div>
      <div class="wire w3"></div>
      <div class="node" style="opacity:.9">
        <div class="ic"><svg viewBox="0 0 24 24"><path d="M4 12h13M12 6l6 6-6 6"/><path d="M20 4v16"/></svg></div>
        <div class="t">Reversement</div><div class="s">net marchand</div>
      </div>
      <div class="wire w1"></div>
      <div class="node" style="opacity:.9">
        <div class="ic"><svg viewBox="0 0 24 24"><path d="M9 14 4 9l5-5"/><path d="M4 9h11a5 5 0 0 1 0 10h-3"/></svg></div>
        <div class="t">Remboursement</div><div class="s">en un appel</div>
      </div>
    </div>
    <div class="cap">Signature HMAC à chaque étape · webhooks signés · anti-rejeu</div>
  </div>
</div></div>

<section id="comment"><div class="wrap">
  <div class="eyebrow">Intégration</div>
  <h2>Trois façons d'encaisser</h2>
  <p class="sub">Choisissez selon votre équipe et votre produit. Vous pouvez changer à tout moment.</p>
  <div class="grid3">
    <div class="card"><div class="ic"><svg viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 9h18M7 14h6"/></svg></div><h3>Page hébergée</h3><p>Redirigez le client vers une
      page prête à l'emploi, à votre nom. Zéro ligne de code de paiement.</p></div>
    <div class="card"><div class="ic"><svg viewBox="0 0 24 24"><path d="M8 3H5a2 2 0 0 0-2 2v3M16 3h3a2 2 0 0 1 2 2v3M8 21H5a2 2 0 0 1-2-2v-3M16 21h3a2 2 0 0 0 2-2v-3"/><circle cx="12" cy="12" r="3"/></svg></div><h3>Widget JavaScript</h3><p>Un bouton « Payer » sur
      votre site ouvre le paiement en fenêtre. Votre page se met à jour toute seule.</p></div>
    <div class="card"><div class="ic"><svg viewBox="0 0 24 24"><path d="M8 6 3 12l5 6M16 6l5 6-5 6M13 4l-2 16"/></svg></div><h3>API</h3><p>Créez un paiement depuis votre backend,
      recevez le résultat par webhook signé. Contrôle total.</p></div>
  </div>
  <ol class="steps" style="margin-top:40px">
    <li><span class="n">1</span><div><b>Créez votre passerelle</b><br>Remplissez le formulaire ci-dessous. Après validation, vous recevez vos identifiants par email.</div></li>
    <li><span class="n">2</span><div><b>Intégrez</b><br>Page hébergée, widget ou API — la documentation vous guide pas à pas.</div></li>
    <li><span class="n">3</span><div><b>Encaissez</b><br>Vos clients paient par mobile money. Vous êtes reversé, commission déduite.</div></li>
  </ol>
</div></section>

<section id="pays"><div class="wrap">
  <div class="eyebrow">Couverture</div>
  <h2>12 pays, dépôts et remboursements</h2>
  <p class="sub">Opérationnel partout ci-dessous, sans démarche supplémentaire de votre côté.</p>
  <div class="chips">
    <span class="chip"><span class="cc">BJ</span>Bénin — Moov, MTN</span>
    <span class="chip"><span class="cc">SN</span>Sénégal — Orange, Free, Wave</span>
    <span class="chip"><span class="cc">CI</span>Côte d'Ivoire — MTN, Orange</span>
    <span class="chip"><span class="cc">CM</span>Cameroun — MTN</span>
    <span class="chip"><span class="cc">CD</span>RD Congo — Airtel, Orange, M-Pesa</span>
    <span class="chip"><span class="cc">CG</span>Congo-Brazzaville — Airtel, MTN</span>
    <span class="chip"><span class="cc">GA</span>Gabon — Airtel</span>
    <span class="chip"><span class="cc">KE</span>Kenya — M-Pesa</span>
    <span class="chip"><span class="cc">RW</span>Rwanda — Airtel, MTN</span>
    <span class="chip"><span class="cc">SL</span>Sierra Leone — Orange</span>
    <span class="chip"><span class="cc">UG</span>Ouganda — Airtel, MTN</span>
    <span class="chip"><span class="cc">ZM</span>Zambie — Airtel, MTN, Zamtel</span>
  </div>
</div></section>

<section id="tarifs"><div class="wrap">
  <div class="eyebrow">Tarifs</div>
  <h2>Simple et sans surprise</h2>
  <p class="sub">Une commission par transaction, retenue sur le montant reversé. Pas d'abonnement, pas de frais fixe.</p>
  <div class="price">
    <div class="card">
      <b class="k">Sans vérification</b>
      <div class="big">200 000 FCFA</div>
      <p style="font-size:13px;color:var(--muted)">par transaction · démarrage immédiat après validation</p>
    </div>
    <div class="card">
      <b class="k">Compte vérifié</b>
      <div class="big">1 000 000 FCFA</div>
      <p style="font-size:13px;color:var(--muted)">par transaction · après envoi des justificatifs</p>
    </div>
    <div class="card">
      <b class="k">Volume &amp; intégration directe</b>
      <div class="big">Sur mesure</div>
      <p style="font-size:13px;color:var(--muted)">nous accompagnons votre entreprise à mettre en place sa propre intégration</p>
    </div>
  </div>
  <table class="fees">
    <tr><td>Commission ABMCY Core</td><td>50 FCFA par dollar de transaction</td></tr>
    <tr><td>Frais opérateur mobile money</td><td>en sus, selon l'opérateur</td></tr>
  </table>
  <p style="font-size:13px;color:var(--muted);margin-top:10px">La commission est retenue automatiquement sur le montant reversé au marchand.</p>
</div></section>

<section id="inscription"><div class="wrap">
  <div class="eyebrow">Démarrer</div>
  <h2>Créer ma passerelle de paiement</h2>
  <p class="sub">Deux minutes. Dites-nous ce que vous voulez encaisser, nous revenons vers vous avec vos accès.</p>
  <div class="formwrap">
    <form id="signup">
      <div class="r2">
        <div><label>Nom de l'entreprise / projet *</label><input name="business_name" required></div>
        <div><label>Votre nom *</label><input name="contact_name" required></div>
      </div>
      <div class="r2">
        <div><label>Email *</label><input name="email" type="email" required></div>
        <div><label>Téléphone</label><input name="phone"></div>
      </div>
      <div class="r2">
        <div><label>Site web</label><input name="website" type="url" placeholder="https://"></div>
        <div><label>Pays principal</label><input name="country" placeholder="Sénégal"></div>
      </div>
      <div><label>Que voulez-vous encaisser ?</label><textarea name="description" placeholder="Ex. abonnements, boutique en ligne, réservations…"></textarea></div>
      <div><label>Volume mensuel estimé</label><input name="expected_volume" placeholder="Ex. ~500 000 FCFA / mois"></div>
      <div id="msg"></div>
      <button class="submit" type="submit">Envoyer la demande</button>
    </form>
  </div>
</div></section>

<footer>ABMCY Core Payment · <a href="/console/login">Console</a></footer>

<script>
var f=document.getElementById('signup'),msg=document.getElementById('msg');
f.addEventListener('submit',function(e){
  e.preventDefault();
  var btn=f.querySelector('button');btn.disabled=true;msg.innerHTML='';
  var data={};new FormData(f).forEach(function(v,k){data[k]=v;});
  fetch('/public/signup',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(data)})
    .then(function(r){return r.json().then(function(b){return{ok:r.ok,b:b};});})
    .then(function(res){
      if(res.ok){msg.className='note ok';msg.textContent=res.b.message||'Demande reçue. Nous revenons vers vous par email.';f.reset();}
      else{msg.className='note err';msg.textContent=res.b.error||'Une erreur est survenue. Réessayez.';}
    })
    .catch(function(){msg.className='note err';msg.textContent='Erreur réseau. Réessayez.';})
    .finally(function(){btn.disabled=false;});
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
