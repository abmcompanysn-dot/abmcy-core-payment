'use client';

import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import Link from 'next/link';
import { Shell } from '../../shell';
import { StatusBadge, RelayBadge, fmtCFA, fmtDate } from '../../ui';
import type { Payment } from '@/lib/core';
import { api } from '@/lib/bp';

export default function PaymentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [p, setP] = useState<Payment | null>(null);
  const [err, setErr] = useState('');
  const [loading, setLoading] = useState(true);
  const [relaying, setRelaying] = useState(false);
  const [relayMsg, setRelayMsg] = useState<{ ok: boolean; text: string } | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setErr('');
    const res = await fetch(api(`/api/payments/${id}`));
    const b = await res.json().catch(() => ({}));
    if (!res.ok) setErr(b.error || `HTTP ${res.status}`);
    else setP(b.payment);
    setLoading(false);
  }, [id]);

  useEffect(() => {
    load();
  }, [load]);

  async function relay() {
    setRelaying(true);
    setRelayMsg(null);
    try {
      const res = await fetch(api(`/api/payments/${id}/relay`), { method: 'POST' });
      const b = await res.json().catch(() => ({}));
      if (b.payment) setP(b.payment);
      if (b.ok) setRelayMsg({ ok: true, text: 'Relais livré à l’application.' });
      else setRelayMsg({ ok: false, text: `Relais échoué : ${b.error || 'erreur inconnue'}` });
    } catch (e) {
      setRelayMsg({ ok: false, text: `Erreur réseau : ${String(e)}` });
    } finally {
      setRelaying(false);
    }
  }

  return (
    <Shell>
      <p>
        <Link href="/payments">← Paiements</Link>
      </p>

      {err && <div className="err-box">{err}</div>}
      {loading && <p className="muted">Chargement…</p>}

      {p && (
        <>
          <div className="panel">
            <h2>
              Paiement <span className="mono">{p.app_ref}</span> &nbsp;
              <StatusBadge value={p.status} />
            </h2>
            <dl className="kv">
              <dt>ID</dt>
              <dd className="mono">{p.id}</dd>
              <dt>Application</dt>
              <dd>{p.app_name || '—'}</dd>
              <dt>Type</dt>
              <dd>{p.type}</dd>
              <dt>Montant payé</dt>
              <dd>
                {fmtCFA(p.amount_cfa)} {p.currency}
              </dd>
              <dt>Frais ABMCY Core</dt>
              <dd>
                {fmtCFA(p.fee_cfa)}
                {p.usd_rate_used ? (
                  <span className="muted"> (50 F / {p.usd_rate_used} F le $)</span>
                ) : null}
              </dd>
              <dt>Net reversé au marchand</dt>
              <dd>{p.net_cfa != null ? fmtCFA(p.net_cfa) : '—'}</dd>
              <dt>Provider</dt>
              <dd>{p.provider || '—'}</dd>
              <dt>Description</dt>
              <dd>{p.description || '—'}</dd>
              <dt>Motif d’échec</dt>
              <dd>{p.failure_reason || '—'}</dd>
              <dt>Créé</dt>
              <dd>{fmtDate(p.created_at)}</dd>
              <dt>Mis à jour</dt>
              <dd>{fmtDate(p.updated_at)}</dd>
            </dl>
          </div>

          <div className="panel">
            <h2>
              Relais vers l’application &nbsp; <RelayBadge value={p.relay_status} />
            </h2>
            <dl className="kv">
              <dt>URL de callback</dt>
              <dd className="mono">{p.callback_url || '— (aucune, rien à relayer)'}</dd>
              <dt>Tentatives</dt>
              <dd>{p.relay_attempts}</dd>
              <dt>Dernière tentative</dt>
              <dd>{fmtDate(p.relay_last_attempt_at)}</dd>
              <dt>Dernière erreur</dt>
              <dd>{p.relay_last_error || '—'}</dd>
            </dl>

            {relayMsg && (
              <div className={relayMsg.ok ? 'ok-box' : 'err-box'} style={{ marginTop: 12 }}>
                {relayMsg.text}
              </div>
            )}

            <button
              style={{ marginTop: 12 }}
              disabled={relaying || !p.callback_url}
              onClick={relay}
              title={!p.callback_url ? 'Aucune URL de callback sur ce paiement' : undefined}
            >
              {relaying ? 'Envoi…' : 'Renvoyer le relais maintenant'}
            </button>
          </div>

          {p.redirect_url && (
            <div className="panel">
              <h2>URL de paiement (PawaPay)</h2>
              <div className="copyrow">
                <code>{p.redirect_url}</code>
              </div>
              <p className="muted" style={{ fontSize: 12 }}>
                Générée par DIARRA au moment de la création. À ouvrir par l’utilisateur final.
              </p>
            </div>
          )}
        </>
      )}
    </Shell>
  );
}
