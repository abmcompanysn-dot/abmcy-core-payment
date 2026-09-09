import { NextResponse } from 'next/server';
import { coreFetch } from '@/lib/core';

export async function GET(req: Request) {
  const qs = new URL(req.url).search; // ?since=7|30|0
  const r = await coreFetch<unknown>(`/admin/stats${qs}`);
  if (!r.ok) return NextResponse.json({ error: r.error }, { status: r.status });
  return NextResponse.json(r.data);
}
