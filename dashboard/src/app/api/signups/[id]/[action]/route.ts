import { NextResponse } from 'next/server';
import { coreFetch } from '@/lib/core';

// action = "approve" | "reject"
export async function POST(req: Request, ctx: { params: Promise<{ id: string; action: string }> }) {
  const { id, action } = await ctx.params;
  if (action !== 'approve' && action !== 'reject') {
    return NextResponse.json({ error: 'unknown_action' }, { status: 400 });
  }
  const body = await req.text();
  const r = await coreFetch<unknown>(`/admin/signups/${id}/${action}`, { method: 'POST', body });
  if (!r.ok) return NextResponse.json({ error: r.error }, { status: r.status });
  return NextResponse.json(r.data);
}
