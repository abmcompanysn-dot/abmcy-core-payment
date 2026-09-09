import { NextResponse } from 'next/server';
import { coreFetch } from '@/lib/core';

export async function POST(_req: Request, ctx: { params: Promise<{ id: string }> }) {
  const { id } = await ctx.params;
  const r = await coreFetch<unknown>(`/admin/payments/${id}/refund`, { method: 'POST' });
  if (!r.ok) return NextResponse.json({ error: r.error }, { status: r.status });
  return NextResponse.json(r.data);
}
