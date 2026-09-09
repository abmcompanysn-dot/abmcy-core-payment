import { NextResponse } from 'next/server';
import { coreFetch } from '@/lib/core';

export async function POST(_req: Request, ctx: { params: Promise<{ id: string }> }) {
  const { id } = await ctx.params;
  // ABMCY Core renvoie toujours 200 ici (échec de relais = {ok:false} dans le
  // corps), donc on relaie tel quel.
  const r = await coreFetch<unknown>(`/admin/payments/${id}/relay`, { method: 'POST' });
  if (!r.ok) return NextResponse.json({ error: r.error }, { status: r.status });
  return NextResponse.json(r.data);
}
