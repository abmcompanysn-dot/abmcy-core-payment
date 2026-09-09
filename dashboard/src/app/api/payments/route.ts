import { NextResponse } from 'next/server';
import { coreFetch, type Payment } from '@/lib/core';

export async function GET(req: Request) {
  const qs = new URL(req.url).search; // ?app_id=&status=&limit=&offset=
  const r = await coreFetch<{ payments: Payment[] }>(`/admin/payments${qs}`);
  if (!r.ok) return NextResponse.json({ error: r.error }, { status: r.status });
  return NextResponse.json(r.data);
}
