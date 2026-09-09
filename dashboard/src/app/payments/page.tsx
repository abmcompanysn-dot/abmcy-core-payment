'use client';

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { Shell } from '../shell';
import { StatusBadge, RelayBadge, fmtCFA, fmtDate } from '../ui';
import type { App, Payment } from '@/lib/core';
import { api } from '@/lib/bp';

const STATUSES = ['', 'pending', 'processing', 'completed', 'failed', 'cancelled'];

export default function PaymentsPage() {
  const [rows, setRows] = useState<Payment[]>([]);
  const [apps, setApps] = useState<App[]>([]);
  const [appId, setAppId] = useState('');
  const [status, setStatus] = useState('');
  const [err, setErr] = useState('');
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    setErr('');
    const qs = new URLSearchParams();
    if (appId) qs.set('app_id', appId);
    if (status) qs.set('status', status);
    qs.set('limit', '100');
    const res = await fetch(api(`/api/payments?${qs.toString()}`));
    const b = await res.json().catch(() => ({}));
    if (!res.ok) setErr(b.error || `HTTP ${res.status}`);
    else setRows(b.payments || []);
    setLoading(false);
  }, [appId, status]);

  useEffect(() => {
    fetch(api('/api/apps'))
      .then((r) => r.json())
      .then((b) => setApps(b.apps || []))
      .catch(() => undefined);
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <Shell>
      <div className="panel">
        <h2>Filtres</h2>
        <div className="row">
          <div>
            <label>Application</label>
            <select value={appId} onChange={(e) => setAppId(e.target.value)}>
              <option value="">toutes</option>
              {apps.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label>Statut</label>
            <select value={status} onChange={(e) => setStatus(e.target.value)}>
              {STATUSES.map((s) => (
                <option key={s} value={s}>
                  {s || 'tous'}
                </option>
              ))}
            </select>
          </div>
          <div style={{ flex: '0 0 auto' }}>
            <label>&nbsp;</label>
            <button className="ghost" onClick={load}>
              Rafraîchir
            </button>
          </div>
        </div>
      </div>

      {err && <div className="err-box">{err}</div>}

      <div className="panel">
        <h2>Paiements {loading ? '…' : `(${rows.length})`}</h2>
        <table>
          <thead>
            <tr>
              <th>Créé</th>
              <th>Application</th>
              <th>Réf. app</th>
              <th>Montant</th>
              <th>Frais</th>
              <th>Net marchand</th>
              <th>Statut</th>
              <th>Relais</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {rows.map((p) => (
              <tr key={p.id}>
                <td className="muted">{fmtDate(p.created_at)}</td>
                <td>{p.app_name || '—'}</td>
                <td className="mono">{p.app_ref}</td>
                <td>{fmtCFA(p.amount_cfa)}</td>
                <td className="muted">{fmtCFA(p.fee_cfa)}</td>
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
                  <RelayBadge value={p.relay_status} />
                  {p.relay_attempts > 0 && (
                    <span className="muted" style={{ fontSize: 11 }}>
                      {' '}
                      ×{p.relay_attempts}
                    </span>
                  )}
                </td>
                <td>
                  <Link href={`/payments/${p.id}`}>détail</Link>
                </td>
              </tr>
            ))}
            {!loading && rows.length === 0 && (
              <tr>
                <td colSpan={9} className="muted">
                  Aucun paiement.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </Shell>
  );
}
