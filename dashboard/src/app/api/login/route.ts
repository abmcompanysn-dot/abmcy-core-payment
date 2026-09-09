// POST { token } -> vérifie le jeton contre ABMCY Core (GET /admin/apps) et,
// si OK, le pose dans un cookie httpOnly. Le jeton ne revient jamais au JS.
import { NextResponse } from 'next/server';
import { coreFetch, TOKEN_COOKIE } from '@/lib/core';

export async function POST(req: Request) {
  let body: { token?: string };
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: 'invalid_request' }, { status: 400 });
  }
  const token = (body.token || '').trim();
  if (!token) return NextResponse.json({ error: 'token_required' }, { status: 400 });

  const probe = await coreFetch('/admin/apps', { tokenOverride: token });
  if (!probe.ok) {
    return NextResponse.json(
      { error: probe.status === 401 ? 'invalid_token' : probe.error },
      { status: probe.status === 401 ? 401 : 502 },
    );
  }

  const res = NextResponse.json({ ok: true });
  res.cookies.set(TOKEN_COOKIE, token, {
    httpOnly: true,
    sameSite: 'lax',
    secure: process.env.NODE_ENV === 'production',
    path: '/',
    maxAge: 60 * 60 * 12,
  });
  return res;
}

export async function DELETE() {
  const res = NextResponse.json({ ok: true });
  res.cookies.set(TOKEN_COOKIE, '', { path: '/', maxAge: 0 });
  return res;
}
