import { NextResponse } from 'next/server';
import { coreFetch } from '@/lib/core';

export async function PUT(req: Request, ctx: { params: Promise<{ id: string }> }) {
  const { id } = await ctx.params;
  const body = await req.text();
  const r = await coreFetch<unknown>(`/admin/apps/${id}/active`, { method: 'PUT', body });
  if (!r.ok) return NextResponse.json({ error: r.error }, { status: r.status });
  return NextResponse.json(r.data);
}
