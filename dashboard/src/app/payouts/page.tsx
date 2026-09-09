'use client';

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { Shell } from '../shell';
import { StatusBadge, fmtCFA, fmtDate } from '../ui';
import { api } from '@/lib/bp';
import type { App, Payment } from '@/lib/core';

// Opérateurs mobile money par pays (jamais l'agrégateur, juste l'opérateur).
const OPERATORS: Record<string, string[]> = {
  BEN: ['MOOV_BEN', 'MTN_MOMO_BEN'],
  SEN: ['ORANGE_SEN', 'FREE_SEN', 'WAVE_SEN'],
  CIV: ['MTN_MOMO_CIV', 'ORANGE_CIV'],
  CMR: ['MTN_MOMO_CMR'],
  COD: ['AIRTEL_COD', 'ORANGE_COD', 'VODACOM_MPESA_COD'],
  COG: ['AIRTEL_COG', 'MTN_MOMO_COG'],
  GAB: ['AIRTEL_GAB'],
  KEN: ['MPESA_KEN'],
  RWA: ['AIRTEL_RWA', 'MTN_MOMO_RWA'],
  SLE: ['ORANGE_SLE'],
  UGA: ['AIRTEL_UGA', 'MTN_MOMO_UGA'],
  ZMB: ['AIRTEL_ZMB', 'MTN_MOMO_ZMB', 'ZAMTEL_ZMB'],
};

export default function PayoutsPage() {
  const [apps, setApps] = useState<App[]>([]);
  const [rows, setRows] = useState<Payment[]>([]);
  const [err, setErr] = useState('');
  const [loading, setLoading] = useState(true);

  const [appId, setAppId] = useState('');
  const [amount, setAmount] = useState('');
  const [country, setCountry] = useState('SEN');
  const [operator, setOperator] = useState('ORANGE_SEN');
  const [phone, setPhone] = useState('');
  const [desc, setDesc] = useState('');
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);

  const loadList = useCallback(async () => {
    setLoading(true);
    const res = await fetch(api(`/api/payments?status=&limit=100`));
    const b = await res.json().catch(() => ({}));
    if (res.ok) setRows((b.payments || []).filter((p: Payment) => p.type === 'payout'));
    setLoading(false);
  }, []);

  useEffect(() => {
    fetch(api('/api/apps'))
      .then((r) => r.json())
      .then((b) => setApps(b.apps || []))
      .catch(() => undefined);
    loadList();
  }, [loadList]);

  useEffect(() => {
    const ops = OPERATORS[country] || [];
    if (ops.length && !ops.includes(operator)) setOperator(ops[0]);
  }, [country, operator]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setMsg(null);
    setErr('');
    try {
      const res = await fetch(api('/api/payouts'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          app_id: appId,
          amount_cfa: Number(amount),
          recipient_phone: phone.trim(),
          recipient_operator: operator,
          country,
          description: desc.trim() || undefined,
        }),
      });
      const b = await res.json().catch(() => ({}));
      if (b.ok) {
        setMsg({
          ok: true,
          text: `Versement créé (statut ${b.payout?.status || 'en cours'}). Net envoyé : ${fmtCFA(
            b.payout?.net_cfa ?? 0,
          )}.`,
        });
        setAmount('');
        setPhone('');
        setDesc('');
        loadList();
      } else {
        setMsg({ ok: false, text: `Échec : ${b.error || `HTTP ${res.status}`}` });
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <Shell>
      {err && <div className="err-box">{err}</div>}

      <form className="panel" onSubmit={submit}>
        <h2>Nouveau versement</h2>
        {msg && (
          <div className={msg.ok ? 'ok-box' : 'err-box'} style={{ marginBottom: 12 }}>
            {msg.text}
          </div>
        )}
        <div className="row">
          <div>
            <label>Application (compte à débiter)</label>
            <select value={appId} onChange={(e) => setAppId(e.target.value)} required>
              <option value="">choisir…</option>
              {apps.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label>Montant débité (FCFA)</label>
            <input
              type="number"
              min="1"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              required
            />
          </div>
        </div>
        <div className="row">
          <div>
            <label>Pays</label>
            <select value={country} onChange={(e) => setCountry(e.target.value)}>
              {Object.keys(OPERATORS).map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label>Opérateur</label>
            <select value={operator} onChange={(e) => setOperator(e.target.value)}>
              {(OPERATORS[country] || []).map((o) => (
                <option key={o} value={o}>
                  {o}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label>Numéro du destinataire</label>
            <input value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="77xxxxxxx" required />
          </div>
        </div>
        <label>Description (optionnel)</label>
        <input value={desc} onChange={(e) => setDesc(e.target.value)} />
        <p className="muted" style={{ fontSize: 12 }}>
          La commission ABMCY Core (50 F/$) est déduite : le destinataire reçoit le montant débité
          moins la commission.
        </p>
        <button style={{ marginTop: 8 }} disabled={busy || !appId || !amount || !phone}>
          {busy ? 'Envoi…' : 'Envoyer le versement'}
        </button>
      </form>

      <div className="panel">
        <h2>Versements {loading ? '…' : `(${rows.length})`}</h2>
        <table>
          <thead>
            <tr>
              <th>Créé</th>
              <th>Application</th>
              <th>Destinataire</th>
              <th>Débité</th>
              <th>Envoyé (net)</th>
              <th>Statut</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {rows.map((p) => (
              <tr key={p.id}>
                <td className="muted">{fmtDate(p.created_at)}</td>
                <td>{p.app_name || '—'}</td>
                <td className="mono">
                  {p.recipient_phone || '—'}
                  {p.recipient_operator && (
                    <span className="muted"> · {p.recipient_operator}</span>
                  )}
                </td>
                <td>{fmtCFA(p.amount_cfa)}</td>
                <td>{p.net_cfa != null ? fmtCFA(p.net_cfa) : '—'}</td>
                <td>
                  <StatusBadge value={p.status} />
                  {p.failure_reason && (
                    <div className="muted" style={{ fontSize: 11 }}>
                      {p.failure_reason}
                    </div>
                  )}
                </td>
                <td>
                  <Link href={`/payments/${p.id}`}>détail</Link>
                </td>
              </tr>
            ))}
            {!loading && rows.length === 0 && (
              <tr>
                <td colSpan={7} className="muted">
                  Aucun versement.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </Shell>
  );
}
