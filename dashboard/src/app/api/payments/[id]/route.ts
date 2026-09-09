import { NextResponse } from 'next/server';
import { coreFetch, type Payment } from '@/lib/core';

export async function GET(_req: Request, ctx: { params: Promise<{ id: string }> }) {
  const { id } = await ctx.params;
  const r = await coreFetch<{ payment: Payment }>(`/admin/payments/${id}`);
  if (!r.ok) return NextResponse.json({ error: r.error }, { status: r.status });
  return NextResponse.json(r.data);
}
