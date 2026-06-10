import { logger } from '@/lib/logger';

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? '/api/v1';

export type ApiError = {
  error: {
    code: string;
    message: string;
  };
};

export type TokenResponse = {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
};

export type UserProfile = {
  id: string;
  email: string;
  username?: string;
  avatar_url?: string;
  tier: string;
  created_at: string;
};

const STORAGE_KEY = 'qx.auth';

export function loadTokens(): TokenResponse | null {
  const raw = localStorage.getItem(STORAGE_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as TokenResponse;
  } catch {
    return null;
  }
}

export function saveTokens(tokens: TokenResponse) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(tokens));
}

export function clearTokens() {
  localStorage.removeItem(STORAGE_KEY);
}

async function request<T>(
  path: string,
  init: RequestInit = {},
  auth = true,
): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set('Content-Type', 'application/json');

  if (auth) {
    const tokens = loadTokens();
    if (tokens?.access_token) {
      headers.set('Authorization', `Bearer ${tokens.access_token}`);
    }
  }

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers });

  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = (await res.json()) as ApiError;
      message = body.error?.message ?? message;
    } catch {
      /* ignore */
    }
    const details = { path, status: res.status, message };
    if (res.status >= 500) {
      logger.error('API request failed', details);
    } else {
      logger.warn('API request failed', details);
    }
    throw new Error(message);
  }

  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

export const api = {
  register: (body: { email: string; password: string; username?: string }) =>
    request<TokenResponse>('/auth/register', { method: 'POST', body: JSON.stringify(body) }, false),

  login: (body: { email: string; password: string }) =>
    request<TokenResponse>('/auth/login', { method: 'POST', body: JSON.stringify(body) }, false),

  logout: () => request<void>('/auth/logout', { method: 'POST' }),

  me: () => request<UserProfile>('/users/me'),

  changePassword: (body: { current_password: string; new_password: string }) =>
    request<void>('/users/me/password', { method: 'PATCH', body: JSON.stringify(body) }),

  changeEmail: (body: { current_password: string; email: string }) =>
    request<UserProfile>('/users/me/email', { method: 'PATCH', body: JSON.stringify(body) }),
};
