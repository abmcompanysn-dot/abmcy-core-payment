'use client';

import { useState } from 'react';

export function StatusBadge({ value }: { value: string }) {
  const cls =
    value === 'completed'
      ? 'b-ok'
      : value === 'failed' || value === 'cancelled'
        ? 'b-err'
        : value === 'processing' || value === 'pending'
          ? 'b-warn'
          : 'b-muted';
  return <span className={`badge ${cls}`}>{value}</span>;
}

export function RelayBadge({ value }: { value: string }) {
  const cls =
    value === 'delivered'
      ? 'b-ok'
      : value === 'failed'
        ? 'b-err'
        : value === 'skipped'
          ? 'b-muted'
          : 'b-warn';
  return <span className={`badge ${cls}`}>{value}</span>;
}

export function Copy({ label, value }: { label: string; value: string }) {
  const [done, setDone] = useState(false);
  return (
    <div>
      <label>{label}</label>
      <div className="copyrow">
        <code>{value}</code>
        <button
          className="ghost"
          type="button"
          onClick={async () => {
            try {
              await navigator.clipboard.writeText(value);
              setDone(true);
              setTimeout(() => setDone(false), 1500);
            } catch {
              /* presse-papiers indispo */
            }
          }}
        >
          {done ? 'Copié' : 'Copier'}
        </button>
      </div>
    </div>
  );
}

export function fmtCFA(n: number) {
  return new Intl.NumberFormat('fr-FR').format(n) + ' F';
}

export function fmtDate(s?: string | null) {
  if (!s) return '—';
  const d = new Date(s);
  return d.toLocaleString('fr-FR', { dateStyle: 'short', timeStyle: 'short' });
}
