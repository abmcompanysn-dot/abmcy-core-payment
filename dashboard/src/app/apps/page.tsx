'use client';

import { useCallback, useEffect, useState } from 'react';
import { Shell } from '../shell';
import { Copy, fmtDate } from '../ui';
import type { App } from '@/lib/core';
import { api } from '@/lib/bp';

type Created = { app: App; api_key: string; hmac_secret: string };

export default function AppsPage() {
  const [apps, setApps] = useState<App[]>([]);
  const [err, setErr] = useState('');
  const [loading, setLoading] = useState(true);

  const [name, setName] = useState('');
  const [cb, setCb] = useState('');
  const [creating, setCreating] = useState(false);
  const [created, setCreated] = useState<Created | null>(null);
  const [busyId, setBusyId] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setErr('');
    const res = await fetch(api('/api/apps'));
    const b = await res.json().catch(() => ({}));
    if (!res.ok) setErr(b.error || `HTTP ${res.status}`);
    else setApps(b.apps || []);
    setLoading(false);
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    setErr('');
    try {
      const res = await fetch(api('/api/apps'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: name.trim(), default_callback_url: cb.trim() || undefined }),
      });
      const b = await res.json().catch(() => ({}));
      if (!res.ok) {
        setErr(b.error || `HTTP ${res.status}`);
        return;
      }
      setCreated(b as Created);
      setName('');
      setCb('');
      load();
    } finally {
      setCreating(false);
    }
  }

  async function toggle(a: App) {
    setBusyId(a.id);
    setErr('');
    try {
      const res = await fetch(api(`/api/apps/${a.id}/active`), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ active: !a.is_active }),
      });
      if (!res.ok) {
        const b = await res.json().catch(() => ({}));
        setErr(b.error || `HTTP ${res.status}`);
      }
      load();
    } finally {
      setBusyId('');
    }
  }

  async function toggleKyc(a: App) {
    setBusyId(a.id);
    setErr('');
    try {
      const next = a.kyc_level === 'verified' ? 'none' : 'verified';
      const res = await fetch(api(`/api/apps/${a.id}/kyc`), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ level: next }),
      });
      if (!res.ok) {
        const b = await res.json().catch(() => ({}));
        setErr(b.error || `HTTP ${res.status}`);
      }
      load();
    } finally {
      setBusyId('');
    }
  }

  return (
    <Shell>
      {err && <div className="err-box">{err}</div>}

      {created && (
        <div className="panel" style={{ borderColor: 'var(--ok)' }}>
          <h2>« {created.app.name} » créée</h2>
          <p className="muted">
            Copiez ces deux valeurs <strong>maintenant</strong> — elles ne seront plus jamais
            affichées.
          </p>
          <Copy label="Clé API (X-App-Key)" value={created.api_key} />
          <Copy label="Secret HMAC (signature des requêtes)" value={created.hmac_secret} />
          <button className="ghost" style={{ marginTop: 12 }} onClick={() => setCreated(null)}>
            J’ai copié, masquer
          </button>
        </div>
      )}

      <form className="panel" onSubmit={create}>
        <h2>Nouvelle application</h2>
        <label htmlFor="n">Nom</label>
        <input id="n" value={name} onChange={(e) => setName(e.target.value)} placeholder="Ma boutique" required />
        <label htmlFor="c">URL de callback par défaut (optionnel)</label>
        <input
          id="c"
          value={cb}
          onChange={(e) => setCb(e.target.value)}
          placeholder="https://ma-boutique.com/webhooks/abmcy"
          type="url"
        />
        <p className="muted" style={{ fontSize: 12 }}>
          Indicatif — chaque appel <code>/v1/pay</code> passe son propre <code>callback_url</code>.
        </p>
        <button style={{ marginTop: 8 }} disabled={creating || !name.trim()}>
          {creating ? 'Création…' : 'Créer'}
        </button>
      </form>

      <div className="panel">
        <h2>Applications {loading ? '…' : `(${apps.length})`}</h2>
        <p className="muted" style={{ fontSize: 12, marginTop: -6 }}>
          Sans vérification : plafond 200 000 FCFA par transaction. Vérifiée : 1 000 000 FCFA.
        </p>
        <table>
          <thead>
            <tr>
              <th>Nom</th>
              <th>ID</th>
              <th>Callback par défaut</th>
              <th>Créée</th>
              <th>État</th>
              <th>KYC</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {apps.map((a) => (
              <tr key={a.id}>
                <td>{a.name}</td>
                <td className="mono">{a.id}</td>
                <td className="mono">{a.default_callback_url || '—'}</td>
                <td className="muted">{fmtDate(a.created_at)}</td>
                <td>
                  <span className={`badge ${a.is_active ? 'b-ok' : 'b-err'}`}>
                    {a.is_active ? 'active' : 'désactivée'}
                  </span>
                </td>
                <td>
                  <span className={`badge ${a.kyc_level === 'verified' ? 'b-ok' : 'b-muted'}`}>
                    {a.kyc_level === 'verified' ? 'vérifiée' : 'non vérifiée'}
                  </span>
                </td>
                <td style={{ whiteSpace: 'nowrap' }}>
                  <button
                    className="ghost"
                    disabled={busyId === a.id}
                    onClick={() => toggleKyc(a)}
                    style={{ marginRight: 6 }}
                  >
                    {a.kyc_level === 'verified' ? 'Retirer KYC' : 'Valider KYC'}
                  </button>
                  <button
                    className={a.is_active ? 'danger' : 'ghost'}
                    disabled={busyId === a.id}
                    onClick={() => toggle(a)}
                  >
                    {a.is_active ? 'Désactiver' : 'Réactiver'}
                  </button>
                </td>
              </tr>
            ))}
            {!loading && apps.length === 0 && (
              <tr>
                <td colSpan={7} className="muted">
                  Aucune application.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </Shell>
  );
}
