// Client HTTP côté serveur Next vers l'API admin d'ABMCY Core.
// Le jeton admin ne transite JAMAIS par le navigateur : il est posé dans un
// cookie httpOnly à la connexion (voir app/api/login), et seules les API
// routes Next (app/api/*) le lisent pour appeler core.diarra.app. Le
// navigateur ne parle qu'à Next -> aucune config CORS sur ABMCY Core.
import { cookies } from 'next/headers';

const CORE_URL = process.env.ABMCY_CORE_URL || 'https://core.diarra.app';
export const TOKEN_COOKIE = 'abmcy_admin_token';

export type App = {
  id: string;
  name: string;
  default_callback_url?: string | null;
  is_active: boolean;
  kyc_level: 'none' | 'verified';
  created_at: string;
};

export type Payment = {
  id: string;
  app_ref: string;
  app_name?: string;
  type: string;
  provider?: string | null;
  status: string;
  failure_reason?: string | null;
  amount_cfa: number;
  fee_cfa: number;
  net_cfa?: number | null;
  usd_rate_used?: number | null;
  currency: string;
  description?: string | null;
  redirect_url?: string | null;
  callback_url?: string | null;
  return_url?: string | null;
  refund_of_payment_id?: string | null;
  relay_status: string;
  relay_attempts: number;
  relay_last_error?: string | null;
  relay_last_attempt_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type SignupRequest = {
  id: string;
  business_name: string;
  contact_name: string;
  email: string;
  phone?: string | null;
  website?: string | null;
  country?: string | null;
  description?: string | null;
  expected_volume?: string | null;
  status: 'pending' | 'approved' | 'rejected';
  review_note?: string | null;
  app_id?: string | null;
  created_at: string;
  reviewed_at?: string | null;
};

async function token(): Promise<string | null> {
  const c = await cookies();
  return c.get(TOKEN_COOKIE)?.value ?? null;
}

type CoreResult<T> = { ok: true; data: T } | { ok: false; status: number; error: string };

export async function coreFetch<T>(
  path: string,
  init?: RequestInit & { tokenOverride?: string },
): Promise<CoreResult<T>> {
  const tok = init?.tokenOverride ?? (await token());
  if (!tok) return { ok: false, status: 401, error: 'not_authenticated' };

  let res: Response;
  try {
    res = await fetch(`${CORE_URL}${path}`, {
      ...init,
      headers: {
        Authorization: `Bearer ${tok}`,
        'Content-Type': 'application/json',
        ...(init?.headers || {}),
      },
      cache: 'no-store',
    });
  } catch (e) {
    return { ok: false, status: 502, error: `core injoignable: ${String(e)}` };
  }

  const text = await res.text();
  let body: unknown = undefined;
  try {
    body = text ? JSON.parse(text) : undefined;
  } catch {
    body = text;
  }

  if (!res.ok) {
    const err =
      body && typeof body === 'object' && 'error' in body
        ? String((body as Record<string, unknown>).error)
        : `HTTP ${res.status}`;
    return { ok: false, status: res.status, error: err };
  }
  return { ok: true, data: body as T };
}

export { CORE_URL };
