// basePath courant, connu côté client via NEXT_PUBLIC_BASE_PATH (inliné au
// build par Next). fetch() n'est PAS préfixé automatiquement par basePath,
// contrairement à <Link>/router — d'où ce helper pour les appels /api/*.
export const BASE_PATH = process.env.NEXT_PUBLIC_BASE_PATH ?? '/console';

export function api(path: string): string {
  return `${BASE_PATH}${path}`;
}
