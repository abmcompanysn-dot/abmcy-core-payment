import { NextResponse } from 'next/server';
import { coreFetch } from '@/lib/core';

export async function POST(req: Request) {
  const body = await req.text();
  const r = await coreFetch<unknown>('/admin/payouts', { method: 'POST', body });
  if (!r.ok) return NextResponse.json({ error: r.error }, { status: r.status });
  return NextResponse.json(r.data);
}
