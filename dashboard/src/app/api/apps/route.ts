import { NextResponse } from 'next/server';
import { coreFetch, type App } from '@/lib/core';

export async function GET() {
  const r = await coreFetch<{ apps: App[] }>('/admin/apps');
  if (!r.ok) return NextResponse.json({ error: r.error }, { status: r.status });
  return NextResponse.json(r.data);
}

export async function POST(req: Request) {
  const body = await req.text();
  const r = await coreFetch<unknown>('/admin/apps', { method: 'POST', body });
  if (!r.ok) return NextResponse.json({ error: r.error }, { status: r.status });
  return NextResponse.json(r.data, { status: 201 });
}
