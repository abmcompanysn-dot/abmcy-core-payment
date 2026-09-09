import { NextResponse } from 'next/server';
import { coreFetch } from '@/lib/core';

export async function GET(req: Request) {
  const qs = new URL(req.url).search;
  const r = await coreFetch<unknown>(`/admin/signups${qs}`);
  if (!r.ok) return NextResponse.json({ error: r.error }, { status: r.status });
  return NextResponse.json(r.data);
}
