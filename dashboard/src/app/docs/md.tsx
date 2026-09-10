'use client';

import { type ReactNode } from 'react';

// Rendu Markdown minimal, sans dépendance — couvre ce qu'utilise
// INTEGRATION.md : titres, code fence, code inline, tables GFM, listes,
// citations, hr, gras. Volontairement simple et lisible.

function inline(text: string, keyPrefix: string): ReactNode[] {
  const out: ReactNode[] = [];
  // découpe sur `code`, **gras**
  const re = /(`[^`]+`|\*\*[^*]+\*\*)/g;
  let last = 0;
  let m: RegExpExecArray | null;
  let i = 0;
  while ((m = re.exec(text))) {
    if (m.index > last) out.push(text.slice(last, m.index));
    const tok = m[0];
    if (tok.startsWith('`')) {
      out.push(
        <code key={`${keyPrefix}-c${i++}`} className="mdcode">
          {tok.slice(1, -1)}
        </code>,
      );
    } else {
      out.push(<strong key={`${keyPrefix}-b${i++}`}>{tok.slice(2, -2)}</strong>);
    }
    last = m.index + tok.length;
  }
  if (last < text.length) out.push(text.slice(last));
  return out;
}

export function Markdown({ source }: { source: string }) {
  const lines = source.replace(/\r\n/g, '\n').split('\n');
  const blocks: ReactNode[] = [];
  let i = 0;
  let k = 0;

  while (i < lines.length) {
    const line = lines[i];

    // code fence
    if (line.startsWith('```')) {
      const lang = line.slice(3).trim();
      const buf: string[] = [];
      i++;
      while (i < lines.length && !lines[i].startsWith('```')) {
        buf.push(lines[i]);
        i++;
      }
      i++; // ferme
      blocks.push(
        <pre key={k++} className="mdpre" data-lang={lang}>
          <code>{buf.join('\n')}</code>
        </pre>,
      );
      continue;
    }

    // hr
    if (/^---+\s*$/.test(line)) {
      blocks.push(<hr key={k++} className="mdhr" />);
      i++;
      continue;
    }

    // heading
    const h = /^(#{1,4})\s+(.*)$/.exec(line);
    if (h) {
      const level = h[1].length;
      const content = inline(h[2], `h${k}`);
      const cls = `mdh mdh${level}`;
      blocks.push(
        level === 1 ? (
          <h1 key={k++} className={cls}>
            {content}
          </h1>
        ) : level === 2 ? (
          <h2 key={k++} className={cls}>
            {content}
          </h2>
        ) : level === 3 ? (
          <h3 key={k++} className={cls}>
            {content}
          </h3>
        ) : (
          <h4 key={k++} className={cls}>
            {content}
          </h4>
        ),
      );
      i++;
      continue;
    }

    // table GFM
    if (line.includes('|') && i + 1 < lines.length && /^\s*\|?[\s:|-]+\|?\s*$/.test(lines[i + 1])) {
      const parseRow = (r: string) =>
        r
          .trim()
          .replace(/^\|/, '')
          .replace(/\|$/, '')
          .split('|')
          .map((c) => c.trim());
      const head = parseRow(line);
      i += 2;
      const rows: string[][] = [];
      while (i < lines.length && lines[i].includes('|')) {
        rows.push(parseRow(lines[i]));
        i++;
      }
      blocks.push(
        <div key={k++} className="mdtablewrap">
          <table className="mdtable">
            <thead>
              <tr>
                {head.map((c, ci) => (
                  <th key={ci}>{inline(c, `th${k}-${ci}`)}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((r, ri) => (
                <tr key={ri}>
                  {r.map((c, ci) => (
                    <td key={ci}>{inline(c, `td${k}-${ri}-${ci}`)}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>,
      );
      continue;
    }

    // blockquote
    if (line.startsWith('>')) {
      const buf: string[] = [];
      while (i < lines.length && lines[i].startsWith('>')) {
        buf.push(lines[i].replace(/^>\s?/, ''));
        i++;
      }
      blocks.push(
        <blockquote key={k++} className="mdquote">
          {inline(buf.join(' '), `q${k}`)}
        </blockquote>,
      );
      continue;
    }

    // liste (- ou 1.)
    if (/^\s*([-*]|\d+\.)\s+/.test(line)) {
      const ordered = /^\s*\d+\.\s+/.test(line);
      const items: string[] = [];
      while (i < lines.length && /^\s*([-*]|\d+\.)\s+/.test(lines[i])) {
        items.push(lines[i].replace(/^\s*([-*]|\d+\.)\s+/, ''));
        i++;
      }
      const inner = items.map((it, ii) => <li key={ii}>{inline(it, `li${k}-${ii}`)}</li>);
      blocks.push(
        ordered ? (
          <ol key={k++} className="mdlist">
            {inner}
          </ol>
        ) : (
          <ul key={k++} className="mdlist">
            {inner}
          </ul>
        ),
      );
      continue;
    }

    // ligne vide
    if (line.trim() === '') {
      i++;
      continue;
    }

    // paragraphe (fusionne les lignes consécutives)
    const buf: string[] = [line];
    i++;
    while (
      i < lines.length &&
      lines[i].trim() !== '' &&
      !lines[i].startsWith('#') &&
      !lines[i].startsWith('```') &&
      !lines[i].startsWith('>') &&
      !/^\s*([-*]|\d+\.)\s+/.test(lines[i]) &&
      !/^---+\s*$/.test(lines[i])
    ) {
      buf.push(lines[i]);
      i++;
    }
    blocks.push(
      <p key={k++} className="mdp">
        {inline(buf.join(' '), `p${k}`)}
      </p>,
    );
  }

  return <div className="md">{blocks}</div>;
}
