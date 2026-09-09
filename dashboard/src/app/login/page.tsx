'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/bp';

export default function LoginPage() {
  const router = useRouter();
  const [token, setToken] = useState('');
  const [err, setErr] = useState('');
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr('');
    try {
      const res = await fetch(api('/api/login'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token }),
      });
      if (!res.ok) {
        const b = await res.json().catch(() => ({}));
        setErr(
          b.error === 'invalid_token'
            ? 'Jeton refusé par ABMCY Core.'
            : b.error === 'token_required'
              ? 'Saisis le jeton.'
              : `Échec: ${b.error || res.status}`,
        );
        return;
      }
      router.replace('/payments');
      router.refresh();
    } catch (e) {
      setErr(`Erreur réseau: ${String(e)}`);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="wrap login-wrap">
      <h1 style={{ fontSize: 18 }}>ABMCY Core Payment</h1>
      <p className="muted">Console d’administration de l’orchestrateur.</p>
      <form className="panel" onSubmit={submit}>
        {err && <div className="err-box">{err}</div>}
        <label htmlFor="tok">Jeton admin (ABMCY_ADMIN_TOKEN)</label>
        <input
          id="tok"
          type="password"
          autoComplete="off"
          value={token}
          onChange={(e) => setToken(e.target.value)}
          placeholder="collez le jeton"
        />
        <button style={{ marginTop: 16, width: '100%' }} disabled={busy || !token.trim()}>
          {busy ? 'Vérification…' : 'Se connecter'}
        </button>
      </form>
    </div>
  );
}
