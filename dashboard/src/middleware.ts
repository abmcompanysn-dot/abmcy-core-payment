import { NextResponse, type NextRequest } from 'next/server';
import { TOKEN_COOKIE } from '@/lib/core';

// Tout est privé sauf /login et /api/login. Pas de cookie -> redirection login.
export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;
  if (pathname === '/login' || pathname === '/api/login') return NextResponse.next();

  const hasToken = Boolean(req.cookies.get(TOKEN_COOKIE)?.value);
  if (!hasToken) {
    const url = req.nextUrl.clone();
    url.pathname = '/login';
    return NextResponse.redirect(url);
  }
  return NextResponse.next();
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico).*)'],
};
