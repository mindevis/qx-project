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

export type GuestSession = {
  guest_token: string;
  expires_in: number;
};

export type LauncherInstance = {
  id: string;
  name: string;
  mc_version: string;
  loader: string;
  created_at: string;
  updated_at: string;
};

export type OfflineProfile = {
  id: string;
  username: string;
  offline_uuid: string;
  created_at: string;
};

export type LaunchRequest = {
  id: string;
  status: string;
  instance_id: string;
  offline_profile_id?: string;
  expires_at: string;
  pid?: number;
  exit_code?: number;
  error_code?: string;
};

export type GameServer = {
  id: string;
  name: string;
  slug: string;
  server_type: string;
  status: string;
  mc_version?: string;
  config: {
    jar_path?: string;
    jvm_args?: string[];
    extra_args?: string[];
  };
  ssh: {
    host: string;
    port: number;
    username: string;
  };
  agent_online: boolean;
  minecraft_running?: boolean;
  last_seen_at?: string;
  created_at: string;
  updated_at: string;
};

export type LinkDeviceResult = {
  status: string;
  guest_token?: string;
  guest_expires_in?: number;
  owner_type: string;
};

export type UserLauncherDevice = {
  linked: boolean;
  device_id?: string;
  status?: string;
  owner_type?: string;
};

const STORAGE_KEY = 'qx.auth';
const GUEST_KEY = 'qx.guest';
const DEVICE_KEY = 'qx.device';

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

export function loadGuestSession(): GuestSession | null {
  const raw = localStorage.getItem(GUEST_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as GuestSession;
  } catch {
    return null;
  }
}

export function saveGuestSession(session: GuestSession) {
  localStorage.setItem(GUEST_KEY, JSON.stringify(session));
}

export function clearGuestSession() {
  localStorage.removeItem(GUEST_KEY);
}

export function saveLinkedDevice(deviceId: string) {
  localStorage.setItem(DEVICE_KEY, deviceId);
}

export function loadLinkedDevice(): string | null {
  return localStorage.getItem(DEVICE_KEY);
}

export function clearLinkedDevice() {
  localStorage.removeItem(DEVICE_KEY);
}

export function hasLauncherAccess(): boolean {
  return !!loadTokens()?.access_token || !!loadGuestSession()?.guest_token;
}

function launcherAuthHeader(): string | null {
  const user = loadTokens()?.access_token;
  if (user) return `Bearer ${user}`;
  const guest = loadGuestSession()?.guest_token;
  if (guest) return `Bearer ${guest}`;
  return null;
}

async function request<T>(
  path: string,
  init: RequestInit = {},
  auth: boolean | 'launcher' = true,
): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set('Content-Type', 'application/json');

  if (auth === true) {
    const tokens = loadTokens();
    if (tokens?.access_token) {
      headers.set('Authorization', `Bearer ${tokens.access_token}`);
    }
  } else if (auth === 'launcher') {
    const header = launcherAuthHeader();
    if (header) {
      headers.set('Authorization', header);
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

  myLauncherDevice: () => request<UserLauncherDevice>('/users/me/launcher-device'),

  changePassword: (body: { current_password: string; new_password: string }) =>
    request<void>('/users/me/password', { method: 'PATCH', body: JSON.stringify(body) }),

  changeEmail: (body: { current_password: string; email: string }) =>
    request<UserProfile>('/users/me/email', { method: 'PATCH', body: JSON.stringify(body) }),

  linkDevice: (body: { device_id: string; user_code?: string }) =>
    request<LinkDeviceResult>('/launcher/devices/link', {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  unlinkDevice: () =>
    request<{ status: string }>('/launcher/devices/unlink', { method: 'POST' }, 'launcher'),

  listInstances: () =>
    request<{ items: LauncherInstance[] }>('/instances', { method: 'GET' }, 'launcher'),

  createInstance: (body: { name: string; mc_version: string; loader?: string }) =>
    request<LauncherInstance>('/instances', { method: 'POST', body: JSON.stringify(body) }, 'launcher'),

  deleteInstance: (id: string) =>
    request<void>(`/instances/${id}`, { method: 'DELETE' }, 'launcher'),

  listProfiles: () =>
    request<{ items: OfflineProfile[] }>('/launcher/profiles', { method: 'GET' }, 'launcher'),

  createProfile: (body: { username: string }) =>
    request<OfflineProfile>('/launcher/profiles', { method: 'POST', body: JSON.stringify(body) }, 'launcher'),

  deleteProfile: (id: string) =>
    request<void>(`/launcher/profiles/${id}`, { method: 'DELETE' }, 'launcher'),

  createLaunchRequest: (body: { instance_id: string; offline_profile_id?: string }) =>
    request<LaunchRequest>('/launcher/launch-requests', {
      method: 'POST',
      body: JSON.stringify(body),
    }, 'launcher'),

  getLaunchRequest: (id: string) =>
    request<LaunchRequest>(`/launcher/launch-requests/${id}`, { method: 'GET' }, 'launcher'),

  listServers: () => request<{ items: GameServer[] }>('/servers'),

  createServer: (body: {
    name: string;
    server_type?: string;
    mc_version?: string;
    ssh: { host: string; port?: number; username: string; private_key: string };
    config?: { jar_path?: string; jvm_args?: string[]; extra_args?: string[] };
  }) => request<GameServer>('/servers', { method: 'POST', body: JSON.stringify(body) }),

  getServer: (id: string) => request<GameServer>(`/servers/${id}`),

  deleteServer: (id: string) => request<void>(`/servers/${id}`, { method: 'DELETE' }),

  deployServer: (id: string) =>
    request<GameServer>(`/servers/${id}/deploy`, { method: 'POST' }),

  startServer: (id: string) =>
    request<{ status: string }>(`/servers/${id}/start`, { method: 'POST' }),

  stopServer: (id: string) =>
    request<{ status: string }>(`/servers/${id}/stop`, { method: 'POST' }),

  restartServer: (id: string) =>
    request<{ status: string }>(`/servers/${id}/restart`, { method: 'POST' }),
};

export type ConsoleMessage = {
  type: string;
  stream?: string;
  line?: string;
  status?: string;
  detail?: string;
};

function wsBaseUrl(apiBase: string = API_BASE): string {
  const raw = apiBase;
  if (raw.startsWith('http://') || raw.startsWith('https://')) {
    const u = new URL(raw);
    u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:';
    u.pathname = '';
    u.search = '';
    return u.origin;
  }
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${window.location.host}`;
}

export { wsBaseUrl };

export function openServerConsole(
  serverId: string,
  handlers: {
    onMessage: (msg: ConsoleMessage) => void;
    onClose?: () => void;
  },
) {
  const token = loadTokens()?.access_token;
  const url = `${wsBaseUrl()}/api/v1/servers/${serverId}/console?access_token=${encodeURIComponent(token ?? '')}`;
  let closedByClient = false;
  let ws: WebSocket | null = null;

  const close = () => {
    closedByClient = true;
    const socket = ws;
    if (!socket) return;
    if (socket.readyState === WebSocket.CONNECTING) {
      // Avoid "closed before connection established" (React StrictMode cleanup).
      socket.addEventListener(
        'open',
        () => {
          socket.close();
        },
        { once: true },
      );
      return;
    }
    if (socket.readyState === WebSocket.OPEN) {
      socket.close();
    }
  };

  ws = new WebSocket(url);
  ws.onmessage = (ev) => {
    try {
      handlers.onMessage(JSON.parse(String(ev.data)) as ConsoleMessage);
    } catch {
      /* ignore malformed frame */
    }
  };
  ws.onclose = () => handlers.onClose?.();
  ws.onerror = () => {
    if (!closedByClient) {
      logger.warn('server console websocket error', { serverId });
    }
  };

  return {
    send(line: string) {
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'input', line }));
      }
    },
    close,
  };
}
