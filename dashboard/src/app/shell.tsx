'use client';

import { usePathname, useRouter } from 'next/navigation';
import { api, BASE_PATH } from '@/lib/bp';
import Link from 'next/link';
import type { ReactNode } from 'react';

export function Shell({ children }: { children: ReactNode }) {
  const path = usePathname(); // inclut le basePath (ex. /console/payments)
  const router = useRouter();

  async function logout() {
    await fetch(api('/api/login'), { method: 'DELETE' });
    router.replace('/login');
    router.refresh();
  }

  const link = (href: string, label: string) => (
    <Link href={href} className={path.startsWith(`${BASE_PATH}${href}`) ? 'active' : ''}>
      {label}
    </Link>
  );

  return (
    <div className="wrap">
      <header className="top">
        <h1>ABMCY Core Payment</h1>
        <nav>
          {link('/payments', 'Paiements')}
          {link('/apps', 'Applications')}
          {link('/signups', 'Demandes')}
          {link('/docs', 'Documentation')}
          <a onClick={logout} style={{ cursor: 'pointer' }}>
            Déconnexion
          </a>
        </nav>
      </header>
      {children}
    </div>
  );
}
