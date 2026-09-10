package handler

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/abmcy/core/internal/fees"
	"github.com/abmcy/core/internal/gateway"
	"github.com/abmcy/core/internal/model"
	"github.com/abmcy/core/internal/repository"
)

// TestPayInput — corps de POST /admin/test-pay (page de test, jeton admin).
type TestPayInput struct {
	AppID     string `json:"app_id"`
	AmountCFA int    `json:"amount_cfa"`
	Country   string `json:"country"`
}

// TestPay — crée un dépôt de test au nom d'une app, autorisé par le JETON
// ADMIN (pas la signature HMAC de l'app). Sert uniquement à la page /test.
// Réutilise exactement le même chemin que Pay : plafond KYC, commission,
// création payments, appel passerelle.
func (h *PaymentHandler) TestPay(ctx context.Context, in TestPayInput) (map[string]any, error) {
	app, err := h.appRepo.FindByID(ctx, in.AppID)
	if err != nil {
		return nil, err
	}
	if in.Country == "" {
		in.Country = "SEN"
	}
	if in.AmountCFA <= 0 {
		in.AmountCFA = 500
	}
	maxAmt := model.MaxAmountFor(app.KYCLevel)
	if in.AmountCFA > maxAmt {
		return map[string]any{"error": "limit_exceeded", "max_cfa": maxAmt}, nil
	}

	fee := fees.Compute(in.AmountCFA, h.usdRate)
	appRef := "test-" + abmcyUUID()[:8]
	diarraRef := abmcyUUID()

	payment, err := h.appRepo.CreatePayment(ctx, repository.CreatePaymentParams{
		AppID:           app.ID,
		AppRef:          appRef,
		DiarraClientRef: diarraRef,
		AmountCFA:       in.AmountCFA,
		FeeCFA:          fee.FeeCFA,
		NetCFA:          fee.NetCFA,
		USDRateUsed:     fee.USDRateUsed,
		Currency:        "XOF",
	})
	if err != nil {
		return nil, err
	}

	tx, err := h.diarra.CreateDeposit(gateway.CreateDepositInput{
		ClientRef:   diarraRef,
		AmountCFA:   in.AmountCFA,
		Country:     in.Country,
		Description: "Paiement de test ABMCY Core",
		CallbackURL: h.selfCallbackURL,
		ReturnURL:   h.hostedPayURL(diarraRef) + "?done",
	})
	if err != nil {
		reason := "diarra_init_failed"
		_ = h.appRepo.UpdatePaymentStatus(ctx, payment.ID, model.PaymentFailed, nil, &reason)
		return nil, err
	}
	_ = h.appRepo.UpdatePaymentRedirect(ctx, payment.ID, tx.RedirectURL)

	return map[string]any{
		"app_ref":      appRef,
		"amount_cfa":   in.AmountCFA,
		"fee_cfa":      fee.FeeCFA,
		"redirect_url": tx.RedirectURL,
		"status_url":   h.publicBaseURL + "/pay/" + diarraRef,
	}, nil
}

type testPageData struct {
	Token string
	Apps  []testAppOpt
}
type testAppOpt struct{ ID, Name, KYC string }

// TestPageHTTP — GET /test : page HTML unique pour essayer un paiement réel.
func (h *PaymentHandler) TestPageHTTP(w http.ResponseWriter, r *http.Request) {
	apps, _ := h.appRepo.List(r.Context())
	data := testPageData{Token: r.URL.Query().Get("token")}
	for _, a := range apps {
		if a.IsActive {
			data.Apps = append(data.Apps, testAppOpt{a.ID, a.Name, a.KYCLevel})
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = testTmpl.Execute(w, data)
}

// TestPayHTTP — POST /test/pay (appelé par la page). Corps JSON TestPayInput.
func (h *PaymentHandler) TestPayHTTP(w http.ResponseWriter, r *http.Request) {
	var in TestPayInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "invalid_request")
		return
	}
	res, err := h.TestPay(r.Context(), in)
	if err != nil {
		writeJSONErr(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

var testTmpl = template.Must(template.New("test").Parse(`<!doctype html>
<html lang="fr"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Test de paiement — ABMCY Core</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font:15px/1.6 -apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#f4f5f7;color:#0b1220;display:flex;min-height:100vh;align-items:center;justify-content:center;padding:20px}
.card{background:#fff;border:1px solid #e3e5e8;border-radius:16px;max-width:440px;width:100%;padding:30px;box-shadow:0 20px 50px rgba(11,18,32,.08)}
h1{font-size:18px;margin-bottom:4px}
.sub{color:#6b7480;font-size:13px;margin-bottom:22px}
label{display:block;font-size:13px;font-weight:600;color:#48525e;margin:14px 0 6px}
select,input{width:100%;border:1px solid #d5dbe2;border-radius:9px;padding:11px 13px;font:inherit;background:#fff}
select:focus,input:focus{outline:0;border-color:#2f6bff;box-shadow:0 0 0 4px rgba(47,107,255,.12)}
button{width:100%;margin-top:20px;background:#2f6bff;color:#fff;border:0;border-radius:11px;padding:14px;font-weight:700;font-size:15px;cursor:pointer;box-shadow:0 8px 22px rgba(47,107,255,.3)}
button:disabled{opacity:.55;box-shadow:none}
.line{display:flex;justify-content:space-between;font-size:13px;color:#48525e;margin-top:8px}
.line b{color:#0b1220}
.msg{margin-top:16px;font-size:14px;padding:12px 14px;border-radius:9px}
.msg.ok{background:#e9f8ef;color:#1c7a43}
.msg.err{background:#fdecec;color:#b42318}
.status{margin-top:14px;font-size:13px}
.st{display:inline-block;padding:2px 9px;border-radius:999px;font-size:11px;font-weight:800;text-transform:uppercase}
.st.pending{background:#fef3c7;color:#92400e}
.st.completed{background:#dcfce7;color:#166534}
.st.failed,.st.cancelled{background:#fee2e2;color:#991b1b}
a.big{display:block;text-align:center;margin-top:12px;background:#0b1220;color:#fff;border-radius:9px;padding:12px;text-decoration:none;font-weight:600}
</style></head><body>
<div class="card">
  <h1>Test de paiement</h1>
  <div class="sub">Crée un vrai paiement mobile money au nom d'une application. Le secret HMAC reste côté serveur.</div>

  <label for="app">Application</label>
  <select id="app">
    {{range .Apps}}<option value="{{.ID}}">{{.Name}} ({{.KYC}})</option>{{end}}
  </select>

  <label for="amt">Montant (FCFA)</label>
  <input id="amt" type="number" value="500" min="100">

  <label for="pays">Pays</label>
  <select id="pays">
    <option value="SEN">Sénégal</option>
    <option value="CIV">Côte d'Ivoire</option>
    <option value="BEN">Bénin</option>
    <option value="CMR">Cameroun</option>
    <option value="COD">RD Congo</option>
    <option value="COG">Congo-Brazzaville</option>
    <option value="GAB">Gabon</option>
    <option value="KEN">Kenya</option>
    <option value="RWA">Rwanda</option>
    <option value="SLE">Sierra Leone</option>
    <option value="UGA">Ouganda</option>
    <option value="ZMB">Zambie</option>
  </select>

  <button id="go">Créer le paiement</button>
  <div id="out"></div>
</div>

<script>
var go=document.getElementById('go'),out=document.getElementById('out'),poll=null;
go.addEventListener('click',function(){
  go.disabled=true;out.innerHTML='';if(poll)clearInterval(poll);
  fetch('/test/pay',{method:'POST',headers:{'Content-Type':'application/json','X-Test-Token':'{{.Token}}'},body:JSON.stringify({
    app_id:document.getElementById('app').value,
    amount_cfa:parseInt(document.getElementById('amt').value,10),
    country:document.getElementById('pays').value
  })}).then(function(r){return r.json();}).then(function(d){
    if(d.error){out.innerHTML='<div class="msg err">'+d.error+'</div>';go.disabled=false;return;}
    out.innerHTML=
      '<div class="msg ok">Paiement créé — référence <b>'+d.app_ref+'</b></div>'+
      '<div class="line"><span>Montant</span><b>'+d.amount_cfa+' FCFA</b></div>'+
      '<div class="line"><span>Commission ABMCY Core</span><b>'+d.fee_cfa+' FCFA</b></div>'+
      '<a class="big" href="'+d.redirect_url+'" target="_blank" rel="noopener">Ouvrir la page de paiement →</a>'+
      '<div class="status" id="stat">Statut : <span class="st pending">pending</span> · en attente du paiement…</div>';
    var statusUrl=d.status_url;
    poll=setInterval(function(){
      fetch(statusUrl).then(function(r){return r.text();}).then(function(html){
        var m=html.match(/st st-(pending|processing|completed|failed|cancelled)/);
        if(m){
          var s=m[1];
          document.getElementById('stat').innerHTML='Statut : <span class="st '+s+'">'+s+'</span>';
          if(s==='completed'||s==='failed'||s==='cancelled'){clearInterval(poll);go.disabled=false;}
        }
      }).catch(function(){});
    },4000);
  }).catch(function(){out.innerHTML='<div class="msg err">Erreur réseau.</div>';go.disabled=false;});
});
</script>
</body></html>`))
