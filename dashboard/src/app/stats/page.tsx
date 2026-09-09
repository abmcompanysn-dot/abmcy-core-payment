'use client';

import { useCallback, useEffect, useState } from 'react';
import { Shell } from '../shell';
import { fmtCFA } from '../ui';
import { api } from '@/lib/bp';

type Totals = {
  count: number;
  amount_cfa: number;
  fee_cfa: number;
  net_cfa: number;
  completed: number;
  failed: number;
  success_rate: number;
};
type ByType = { type: string; count: number; amount_cfa: number; fee_cfa: number };
type ByApp = {
  app_id: string;
  app_name: string;
  count: number;
  amount_cfa: number;
  fee_cfa: number;
  success_rate: number;
};
type ByOperator = { operator: string; count: number; amount_cfa: number };
type Dash = {
  since_days: number;
  totals: Totals;
  by_type: ByType[] | null;
  by_app: ByApp[] | null;
  by_operator: ByOperator[] | null;
};

const PERIODS = [
  { d: 7, label: '7 jours' },
  { d: 30, label: '30 jours' },
  { d: 0, label: 'Tout' },
];

const TYPE_LABEL: Record<string, string> = {
  deposit: 'Encaissements',
  payout: 'Versements',
  refund: 'Remboursements',
};

function Kpi({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="kpi">
      <div className="kpi-l">{label}</div>
      <div className="kpi-v">{value}</div>
      {sub && <div className="kpi-s">{sub}</div>}
    </div>
  );
}

export default function StatsPage() {
  const [since, setSince] = useState(30);
  const [d, setD] = useState<Dash | null>(null);
  const [err, setErr] = useState('');
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    setErr('');
    const res = await fetch(api(`/api/stats?since=${since}`));
    const b = await res.json().catch(() => ({}));
    if (!res.ok) setErr(b.error || `HTTP ${res.status}`);
    else setD(b);
    setLoading(false);
  }, [since]);

  useEffect(() => {
    load();
  }, [load]);

  const t = d?.totals;

  return (
    <Shell>
      <style>{`
        .kpis{display:grid;grid-template-columns:repeat(4,1fr);gap:14px;margin-bottom:18px}
        .kpi{border:1px solid var(--border);border-radius:10px;padding:16px;background:var(--panel)}
        .kpi-l{font-size:12px;color:var(--muted);text-transform:uppercase;letter-spacing:.04em}
        .kpi-v{font-size:22px;font-weight:800;margin-top:6px;letter-spacing:-.01em}
        .kpi-s{font-size:12px;color:var(--muted);margin-top:2px}
        .seg{display:inline-flex;border:1px solid var(--border);border-radius:8px;overflow:hidden}
        .seg button{background:transparent;border:0;padding:7px 14px;color:var(--muted);cursor:pointer;font-weight:600}
        .seg button.on{background:var(--accent);color:#fff}
        @media(max-width:780px){.kpis{grid-template-columns:repeat(2,1fr)}}
      `}</style>

      <div className="row" style={{ justifyContent: 'space-between', marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>Tableau de bord</h2>
        <div className="seg">
          {PERIODS.map((p) => (
            <button key={p.d} className={since === p.d ? 'on' : ''} onClick={() => setSince(p.d)}>
              {p.label}
            </button>
          ))}
        </div>
      </div>

      {err && <div className="err-box">{err}</div>}
      {loading && <p className="muted">Chargement…</p>}

      {t && (
        <>
          <div className="kpis">
            <Kpi label="Volume total" value={fmtCFA(t.amount_cfa)} sub={`${t.count} transactions`} />
            <Kpi
              label="Frais encaissés"
              value={fmtCFA(t.fee_cfa)}
              sub="commission ABMCY Core"
            />
            <Kpi label="Net reversé" value={fmtCFA(t.net_cfa)} sub="montant aux marchands" />
            <Kpi
              label="Taux de succès"
              value={`${Math.round(t.success_rate * 100)}%`}
              sub={`${t.completed} ok · ${t.failed} échecs`}
            />
          </div>

          <div className="panel">
            <h2>Par type</h2>
            <table>
              <thead>
                <tr>
                  <th>Type</th>
                  <th>Transactions</th>
                  <th>Volume</th>
                  <th>Frais</th>
                </tr>
              </thead>
              <tbody>
                {(d?.by_type || []).map((x) => (
                  <tr key={x.type}>
                    <td>{TYPE_LABEL[x.type] || x.type}</td>
                    <td>{x.count}</td>
                    <td>{fmtCFA(x.amount_cfa)}</td>
                    <td className="muted">{fmtCFA(x.fee_cfa)}</td>
                  </tr>
                ))}
                {(!d?.by_type || d.by_type.length === 0) && (
                  <tr>
                    <td colSpan={4} className="muted">
                      Aucune donnée sur la période.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          <div className="panel">
            <h2>Par application</h2>
            <table>
              <thead>
                <tr>
                  <th>Application</th>
                  <th>Transactions</th>
                  <th>Volume</th>
                  <th>Frais générés</th>
                  <th>Succès</th>
                </tr>
              </thead>
              <tbody>
                {(d?.by_app || []).map((x) => (
                  <tr key={x.app_id}>
                    <td>{x.app_name}</td>
                    <td>{x.count}</td>
                    <td>{fmtCFA(x.amount_cfa)}</td>
                    <td className="muted">{fmtCFA(x.fee_cfa)}</td>
                    <td>{Math.round(x.success_rate * 100)}%</td>
                  </tr>
                ))}
                {(!d?.by_app || d.by_app.length === 0) && (
                  <tr>
                    <td colSpan={5} className="muted">
                      Aucune donnée.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          <div className="panel">
            <h2>Versements par opérateur</h2>
            <table>
              <thead>
                <tr>
                  <th>Opérateur</th>
                  <th>Versements</th>
                  <th>Montant</th>
                </tr>
              </thead>
              <tbody>
                {(d?.by_operator || []).map((x) => (
                  <tr key={x.operator}>
                    <td>{x.operator}</td>
                    <td>{x.count}</td>
                    <td>{fmtCFA(x.amount_cfa)}</td>
                  </tr>
                ))}
                {(!d?.by_operator || d.by_operator.length === 0) && (
                  <tr>
                    <td colSpan={3} className="muted">
                      Aucun versement sur la période.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </>
      )}
    </Shell>
  );
}
