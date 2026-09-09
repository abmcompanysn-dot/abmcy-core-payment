package handler

import (
	"net/http"
	"strings"
	"time"
)

// widgetJS — SDK navigateur d'ABMCY Core. Volontairement minimal et sans
// dépendance : l'app appelle son PROPRE backend pour créer le paiement
// (là où vit le secret HMAC), le widget se contente d'ouvrir la page de
// paiement hébergée et d'écouter son issue.
//
// Contrat côté app :
//
//	AbmcyPay.mount(selector, {
//	  createPayment: async () => ({ hosted_pay_url, payment: { app_ref } }),
//	  onSuccess: (appRef) => {},
//	  onCancel:  () => {},
//	  onError:   (err) => {},
//	})
const widgetJS = `(function (w, d) {
  'use strict';
  function openCentered(url, name) {
    var width = 480, height = 700;
    var y = (w.outerHeight - height) / 2 + (w.screenY || 0);
    var x = (w.outerWidth - width) / 2 + (w.screenX || 0);
    return w.open(url, name,
      'width=' + width + ',height=' + height + ',top=' + y + ',left=' + x +
      ',resizable=yes,scrollbars=yes');
  }

  function start(opts) {
    if (!opts || typeof opts.createPayment !== 'function') {
      throw new Error('AbmcyPay: createPayment (fonction) est requis');
    }
    Promise.resolve(opts.createPayment()).then(function (res) {
      var url = res && res.hosted_pay_url;
      var appRef = res && res.payment && res.payment.app_ref;
      if (!url) throw new Error('AbmcyPay: hosted_pay_url manquant dans la réponse de createPayment');

      var origin;
      try { origin = new URL(url).origin; } catch (e) { origin = '*'; }
      var win = openCentered(url, 'abmcy_pay');
      var done = false;

      function finish(status) {
        if (done) return;
        done = true;
        w.removeEventListener('message', onMsg);
        clearInterval(poll);
        if (status === 'completed' && opts.onSuccess) opts.onSuccess(appRef);
        else if (opts.onCancel) opts.onCancel();
      }

      function onMsg(ev) {
        if (origin !== '*' && ev.origin !== origin) return;
        var data = ev.data || {};
        if (data.abmcy_pay) finish(data.status);
      }
      w.addEventListener('message', onMsg);

      // Repli : si la fenêtre est fermée sans message, on considère annulé.
      var poll = setInterval(function () {
        if (win && win.closed) finish('cancelled');
      }, 600);
    }).catch(function (err) {
      if (opts.onError) opts.onError(err);
      else if (w.console) w.console.error('AbmcyPay', err);
    });
  }

  w.AbmcyPay = {
    open: start,
    mount: function (selector, opts) {
      var el = typeof selector === 'string' ? d.querySelector(selector) : selector;
      if (!el) throw new Error('AbmcyPay: élément introuvable: ' + selector);
      el.addEventListener('click', function (e) {
        e.preventDefault();
        start(opts);
      });
    }
  };
})(window, document);
`

// WidgetJS — GET /widget/abmcy-pay.js : sert le SDK navigateur. Cacheable.
func (h *PaymentHandler) WidgetJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeContent(w, r, "abmcy-pay.js", time.Time{}, strings.NewReader(widgetJS))
}
