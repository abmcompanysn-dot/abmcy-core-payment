'use client';

import { useCallback, useEffect, useState } from 'react';
import { Shell } from '../shell';
import { Copy, fmtDate } from '../ui';
import { api } from '@/lib/bp';
import type { SignupRequest } from '@/lib/core';

type Approved = {
  app: { id: string; name: string };
  api_key: string;
  hmac_secret: string;
  email: string;
};

export default function SignupsPage() {
  const [rows, setRows] = useState<SignupRequest[]>([]);
  const [status, setStatus] = useState('pending');
  const [err, setErr] = useState('');
  const [loading, setLoading] = useState(true);
  const [busyId, setBusyId] = useState('');
  const [approved, setApproved] = useState<Approved | null>(null);
  const [noteById, setNoteById] = useState<Record<string, string>>({});

  const load = useCallback(async () => {
    setLoading(true);
    setErr('');
    const qs = status ? `?status=${status}` : '';
    const res = await fetch(api(`/api/signups${qs}`));
    const b = await res.json().catch(() => ({}));
    if (!res.ok) setErr(b.error || `HTTP ${res.status}`);
    else setRows(b.signups || []);
    setLoading(false);
  }, [status]);

  useEffect(() => {
    load();
  }, [load]);

  async function act(id: string, action: 'approve' | 'reject') {
    setBusyId(id);
    setErr('');
    try {
      const res = await fetch(api(`/api/signups/${id}/${action}`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ note: noteById[id] || '' }),
      });
      const b = await res.json().catch(() => ({}));
      if (!res.ok) {
        setErr(b.error || `HTTP ${res.status}`);
        return;
      }
      if (action === 'approve' && b.api_key) {
        setApproved({ app: b.app, api_key: b.api_key, hmac_secret: b.hmac_secret, email: b.email });
      }
      load();
    } finally {
      setBusyId('');
    }
  }

  return (
    <Shell>
      {err && <div className="err-box">{err}</div>}

      {approved && (
        <div className="panel" style={{ borderColor: 'var(--ok)' }}>
          <h2>« {approved.app.name} » approuvée</h2>
          <p className="muted">
            À transmettre à <strong>{approved.email}</strong> — ces valeurs ne seront plus jamais
            affichées.
          </p>
          <Copy label="Clé API (X-App-Key)" value={approved.api_key} />
          <Copy label="Secret HMAC" value={approved.hmac_secret} />
          <button className="ghost" style={{ marginTop: 12 }} onClick={() => setApproved(null)}>
            J’ai transmis, masquer
          </button>
        </div>
      )}

      <div className="panel">
        <div className="row" style={{ marginBottom: 14 }}>
          <div style={{ flex: '0 0 auto' }}>
            <label>Statut</label>
            <select value={status} onChange={(e) => setStatus(e.target.value)}>
              <option value="pending">en attente</option>
              <option value="approved">approuvées</option>
              <option value="rejected">rejetées</option>
              <option value="">toutes</option>
            </select>
          </div>
          <div style={{ flex: '0 0 auto' }}>
            <label>&nbsp;</label>
            <button className="ghost" onClick={load}>
              Rafraîchir
            </button>
          </div>
        </div>

        <h2>Demandes {loading ? '…' : `(${rows.length})`}</h2>
        {rows.length === 0 && !loading && <p className="muted">Aucune demande.</p>}

        {rows.map((s) => (
          <div
            key={s.id}
            style={{
              borderTop: '1px solid var(--border)',
              padding: '16px 0',
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: 16 }}>
              <div>
                <strong>{s.business_name}</strong>{' '}
                <span className="muted">· {s.contact_name} · {s.email}</span>
                {s.phone && <span className="muted"> · {s.phone}</span>}
                <div className="muted" style={{ fontSize: 13, marginTop: 4 }}>
                  {s.country && <>Pays : {s.country} · </>}
                  {s.website && (
                    <>
                      <a href={s.website} target="_blank" rel="noreferrer">
                        {s.website}
                      </a>{' '}
                      ·{' '}
                    </>
                  )}
                  reçue {fmtDate(s.created_at)}
                </div>
                {s.description && <p style={{ fontSize: 14, margin: '6px 0 0' }}>{s.description}</p>}
                {s.expected_volume && (
                  <p className="muted" style={{ fontSize: 13, margin: '2px 0 0' }}>
                    Volume estimé : {s.expected_volume}
                  </p>
                )}
                {s.review_note && (
                  <p className="muted" style={{ fontSize: 13, margin: '4px 0 0' }}>
                    Note : {s.review_note}
                  </p>
                )}
              </div>
              <div style={{ textAlign: 'right', flex: '0 0 auto' }}>
                <span
                  className={`badge ${
                    s.status === 'approved'
                      ? 'b-ok'
                      : s.status === 'rejected'
                        ? 'b-err'
                        : 'b-warn'
                  }`}
                >
                  {s.status === 'pending' ? 'en attente' : s.status === 'approved' ? 'approuvée' : 'rejetée'}
                </span>
              </div>
            </div>

            {s.status === 'pending' && (
              <div style={{ marginTop: 10, display: 'flex', gap: 8, alignItems: 'center' }}>
                <input
                  placeholder="note (optionnelle / motif de rejet)"
                  value={noteById[s.id] || ''}
                  onChange={(e) => setNoteById((m) => ({ ...m, [s.id]: e.target.value }))}
                  style={{ flex: 1 }}
                />
                <button disabled={busyId === s.id} onClick={() => act(s.id, 'approve')}>
                  Approuver
                </button>
                <button
                  className="danger"
                  disabled={busyId === s.id}
                  onClick={() => act(s.id, 'reject')}
                >
                  Rejeter
                </button>
              </div>
            )}
          </div>
        ))}
      </div>
    </Shell>
  );
}
