import { logger } from '@/lib/logger';

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? '/api/v1';

export type ApiError = {
  error: {
    code: string;
    message: string;
  };
};

export const API_ERROR_BACKEND_UNAVAILABLE = 'BACKEND_UNAVAILABLE' as const;

export class ApiRequestError extends Error {
  readonly code?: typeof API_ERROR_BACKEND_UNAVAILABLE;

  constructor(message: string, code?: typeof API_ERROR_BACKEND_UNAVAILABLE) {
    super(message);
    this.name = 'ApiRequestError';
    this.code = code;
  }
}

export function isBackendUnavailableError(error: unknown): boolean {
  return error instanceof ApiRequestError && error.code === API_ERROR_BACKEND_UNAVAILABLE;
}

const BACKEND_UNAVAILABLE_HTTP_STATUSES = new Set([502, 503, 504]);

function throwBackendUnavailable(): never {
  throw new ApiRequestError('Backend unavailable', API_ERROR_BACKEND_UNAVAILABLE);
}

function isBackendUnavailableStatus(status: number): boolean {
  return BACKEND_UNAVAILABLE_HTTP_STATUSES.has(status);
}

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

export type LauncherInstance = {
  id: string;
  name: string;
  mc_version: string;
  loader: string;
  loader_version?: string;
  created_at: string;
  updated_at: string;
};

export type ProfileModel = 'steve' | 'alex';

export type OfflineProfile = {
  id: string;
  username: string;
  offline_uuid: string;
  model: ProfileModel;
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
    work_dir?: string;
    command?: string;
    args?: string[];
    jvm_args?: string[];
    extra_args?: string[];
  };
  ssh: {
    host: string;
    port: number;
    username: string;
  };
  agent_deployed?: boolean;
  agent_online: boolean;
  agent_version?: string;
  minecraft_running?: boolean;
  last_seen_at?: string;
  created_at: string;
  updated_at: string;
};

export type VpsGameServerInstance = {
  id: string;
  name: string;
  server_type: string;
  mc_version: string;
  loader_version?: string;
  address?: string;
  port: number;
  rcon_password?: string;
  rcon_port?: number;
  status: string;
  show_in_monitoring?: boolean;
  monitoring_description?: string;
  banner_url?: string;
  monitoring_tags?: string[];
  created_at: string;
};

export type MonitoringServer = {
  id: string;
  name: string;
  server_type: string;
  mc_version: string;
  loader_version?: string;
  address: string;
  port: number;
  status: string;
  is_online: boolean;
  is_premium: boolean;
  description?: string;
  banner_url?: string;
  tags: string[];
  mods: string[];
  plugins: string[];
  likes_count: number;
  rating_avg: number;
  rating_count: number;
};

export type GameServerProperty = {
  key: string;
  value: string;
  boolean?: boolean;
};

export type GameServerFileEntry = {
  name: string;
  path: string;
  dir: boolean;
  size?: number;
};

export type GameServerFileContent = {
  path: string;
  content: string;
  size: number;
};

export type LinkDeviceResult = {
  status: string;
  owner_type: string;
};

export type DeviceStatus = {
  status: string;
  device_id?: string;
  hostname?: string;
  os?: string;
  launcher_version?: string;
  link_expires_at?: string;
  last_seen_at?: string;
  owner_type?: string;
};

export type McVersionItem = {
  id: string;
  type: string;
};

export type McVersionsList = {
  latest?: Record<string, string>;
  items: McVersionItem[];
};

export type UserLauncherDevice = {
  linked: boolean;
  device_id?: string;
  status?: string;
  owner_type?: string;
};

const STORAGE_KEY = 'qx.auth';
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
  return !!loadTokens()?.access_token;
}

function launcherAuthHeader(): string | null {
  const user = loadTokens()?.access_token;
  return user ? `Bearer ${user}` : null;
}

export async function checkBackendHealth(): Promise<boolean> {
  try {
    const res = await fetch(`${API_BASE}/health`, {
      method: 'GET',
      cache: 'no-store',
    });
    return res.ok;
  } catch {
    logger.warn('backend health check failed');
    return false;
  }
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

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers }).catch(() => {
    throwBackendUnavailable();
  });

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
    if (isBackendUnavailableStatus(res.status)) {
      throwBackendUnavailable();
    }
    throw new ApiRequestError(message);
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

  linkDevice: (body: { device_id: string }) =>
    request<LinkDeviceResult>('/launcher/devices/link', {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  deviceStatus: (deviceId: string) =>
    request<DeviceStatus>(`/launcher/devices/${encodeURIComponent(deviceId)}/status`, {}, false),

  listMcVersions: () => request<McVersionsList>('/launcher/mc-versions', {}, false),

  unlinkDevice: () =>
    request<{ status: string }>('/launcher/devices/unlink', { method: 'POST' }, 'launcher'),

  listInstances: () =>
    request<{ items: LauncherInstance[] }>('/instances', { method: 'GET' }, 'launcher'),

  createInstance: (body: {
    name: string;
    mc_version: string;
    loader?: string;
    loader_version?: string;
  }) =>
    request<LauncherInstance>('/instances', { method: 'POST', body: JSON.stringify(body) }, 'launcher'),

  deleteInstance: (id: string) =>
    request<void>(`/instances/${id}`, { method: 'DELETE' }, 'launcher'),

  listProfiles: () =>
    request<{ items: OfflineProfile[] }>('/launcher/profiles', { method: 'GET' }, 'launcher'),

  createProfile: (body: { username: string; model?: ProfileModel }) =>
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
    ssh: {
      host: string;
      port?: number;
      username: string;
      private_key: string;
      private_key_passphrase?: string;
    };
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

  listVpsGameServers: (vpsId: string) =>
    request<{ items: VpsGameServerInstance[] }>(`/servers/${encodeURIComponent(vpsId)}/game-servers`),

  createVpsGameServer: (
    vpsId: string,
    body: {
      name: string;
      server_type: string;
      mc_version: string;
      loader_version?: string;
      address?: string;
      port?: number;
      show_in_monitoring?: boolean;
      monitoring_description?: string;
      banner_url?: string;
      monitoring_tags?: string[];
    },
  ) =>
    request<VpsGameServerInstance>(`/servers/${encodeURIComponent(vpsId)}/game-servers`, {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  updateVpsGameServer: (
    vpsId: string,
    gameServerId: string,
    body: {
      name?: string;
      address?: string;
      port?: number;
      show_in_monitoring?: boolean;
      monitoring_description?: string;
      banner_url?: string;
      monitoring_tags?: string[];
    },
  ) =>
    request<VpsGameServerInstance>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}`,
      { method: 'PATCH', body: JSON.stringify(body) },
    ),

  deleteVpsGameServer: (vpsId: string, gameServerId: string) =>
    request<void>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}`,
      { method: 'DELETE' },
    ),

  reinstallVpsGameServer: (vpsId: string, gameServerId: string) =>
    request<VpsGameServerInstance>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/reinstall`,
      { method: 'POST' },
    ),

  startVpsGameServer: (vpsId: string, gameServerId: string) =>
    request<VpsGameServerInstance>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/start`,
      { method: 'POST' },
    ),

  stopVpsGameServer: (vpsId: string, gameServerId: string) =>
    request<VpsGameServerInstance>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/stop`,
      { method: 'POST' },
    ),

  restartVpsGameServer: (vpsId: string, gameServerId: string) =>
    request<VpsGameServerInstance>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/restart`,
      { method: 'POST' },
    ),

  getVpsGameServer: (vpsId: string, gameServerId: string) =>
    request<VpsGameServerInstance>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}`,
    ),

  getVpsGameServerProperties: (vpsId: string, gameServerId: string) =>
    request<{ properties: GameServerProperty[] }>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/properties`,
    ),

  patchVpsGameServerProperties: (
    vpsId: string,
    gameServerId: string,
    updates: Record<string, string>,
  ) =>
    request<{ status: string }>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/properties`,
      { method: 'PATCH', body: JSON.stringify({ updates }) },
    ),

  listVpsGameServerMods: (vpsId: string, gameServerId: string) =>
    request<{ items: GameServerFileEntry[] }>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/mods`,
    ),

  listVpsGameServerFiles: (vpsId: string, gameServerId: string, path = '') =>
    request<{ items: GameServerFileEntry[] }>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/files?path=${encodeURIComponent(path)}`,
    ),

  readVpsGameServerFile: (vpsId: string, gameServerId: string, path: string) =>
    request<GameServerFileContent>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/files/content?path=${encodeURIComponent(path)}`,
    ),

  writeVpsGameServerFile: (
    vpsId: string,
    gameServerId: string,
    path: string,
    content: string,
  ) =>
    request<{ status: string }>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/files/content?path=${encodeURIComponent(path)}`,
      { method: 'PUT', body: JSON.stringify({ content }) },
    ),

  listMonitoringServers: (params?: {
    mc_version?: string;
    loader?: string;
    mod?: string;
    plugin?: string;
    q?: string;
  }) => {
    const search = new URLSearchParams();
    if (params?.mc_version) search.set('mc_version', params.mc_version);
    if (params?.loader) search.set('loader', params.loader);
    if (params?.mod) search.set('mod', params.mod);
    if (params?.plugin) search.set('plugin', params.plugin);
    if (params?.q) search.set('q', params.q);
    const qs = search.toString();
    return request<{ items: MonitoringServer[] }>(
      `/monitoring/servers${qs ? `?${qs}` : ''}`,
      {},
      false,
    );
  },

  likeMonitoringServer: (gameServerId: string) =>
    request<MonitoringServer>(`/monitoring/servers/${encodeURIComponent(gameServerId)}/like`, {
      method: 'POST',
    }),

  rateMonitoringServer: (gameServerId: string, rating: number) =>
    request<MonitoringServer>(`/monitoring/servers/${encodeURIComponent(gameServerId)}/rate`, {
      method: 'POST',
      body: JSON.stringify({ rating }),
    }),
};

export type ConsoleMessage = {
  type: string;
  stream?: string;
  line?: string;
  status?: string;
  detail?: string;
  game_server_id?: string;
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
  gameServerId?: string,
) {
  const token = loadTokens()?.access_token;
  const params = new URLSearchParams();
  if (token) params.set('access_token', token);
  if (gameServerId) params.set('game_server_id', gameServerId);
  const url = `${wsBaseUrl()}/api/v1/servers/${serverId}/console?${params.toString()}`;
  let closedByClient = false;
  const ws = new WebSocket(url);

  const close = () => {
    closedByClient = true;
    if (ws.readyState === WebSocket.CONNECTING) {
      // Avoid "closed before connection established" (React StrictMode cleanup).
      ws.addEventListener(
        'open',
        () => {
          ws.close();
        },
        { once: true },
      );
      return;
    }
    if (ws.readyState === WebSocket.OPEN) {
      ws.close();
    }
  };

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
