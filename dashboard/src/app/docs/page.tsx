'use client';

import { useEffect, useState } from 'react';
import { Shell } from '../shell';
import { Markdown } from './md';

const DOC_URL = 'https://core.diarra.app/docs/integration.md';

export default function DocsPage() {
  const [md, setMd] = useState<string>('');
  const [err, setErr] = useState('');

  useEffect(() => {
    fetch(DOC_URL)
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.text();
      })
      .then(setMd)
      .catch((e) => setErr(String(e)));
  }, []);

  return (
    <Shell>
      <style>{`
        .docbar{display:flex;align-items:center;justify-content:space-between;margin-bottom:14px}
        .docbar h2{margin:0}
        .dl{display:inline-flex;align-items:center;gap:7px;border:1px solid var(--border);border-radius:8px;padding:8px 14px;font-size:13px;font-weight:600;color:var(--ink);text-decoration:none}
        .dl:hover{border-color:var(--accent);color:var(--accent)}
        .md{max-width:820px;line-height:1.7;font-size:14.5px;color:var(--ink)}
        .mdh{font-weight:800;letter-spacing:-.01em;color:var(--ink)}
        .mdh1{font-size:26px;margin:0 0 10px}
        .mdh2{font-size:19px;margin:34px 0 10px;padding-top:14px;border-top:1px solid var(--border)}
        .mdh3{font-size:15px;margin:22px 0 6px}
        .mdh4{font-size:13px;margin:16px 0 4px;color:var(--muted);text-transform:uppercase;letter-spacing:.04em}
        .mdp{margin:10px 0}
        .mdcode{background:var(--bg);border:1px solid var(--border);border-radius:5px;padding:1px 5px;font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12.5px}
        .mdpre{position:relative;background:#0d1117;color:#e6edf3;border-radius:10px;padding:16px;overflow-x:auto;font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12.5px;line-height:1.6;margin:12px 0}
        .mdpre code{white-space:pre;background:none;border:0;color:inherit;font-size:inherit}
        .mdpre[data-lang]:not([data-lang=""])::before{content:attr(data-lang);position:absolute;top:8px;right:12px;font-size:10px;color:#7d8590;text-transform:uppercase;letter-spacing:.08em}
        .mdhr{border:0;border-top:1px solid var(--border);margin:26px 0}
        .mdlist{margin:10px 0;padding-left:22px}
        .mdlist li{margin:5px 0}
        .mdquote{border-left:3px solid var(--accent);background:var(--bg);padding:10px 14px;border-radius:0 8px 8px 0;margin:14px 0;color:var(--muted);font-size:13.5px}
        .mdtablewrap{overflow-x:auto;margin:14px 0}
        .mdtable{border-collapse:collapse;width:100%;font-size:13px}
        .mdtable th{text-align:left;background:var(--bg);border:1px solid var(--border);padding:8px 10px;font-weight:700}
        .mdtable td{border:1px solid var(--border);padding:8px 10px;vertical-align:top}
      `}</style>

      <div className="docbar">
        <h2>Documentation d’intégration</h2>
        <a className="dl" href={`${DOC_URL}?download=1`} download>
          ⬇ Télécharger le guide (.md)
        </a>
      </div>

      {err && <div className="err-box">Chargement de la doc impossible : {err}</div>}
      {!md && !err && <p className="muted">Chargement…</p>}
      {md && <Markdown source={md} />}
    </Shell>
  );
}
