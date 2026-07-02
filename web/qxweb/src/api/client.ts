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
  readonly apiCode?: string;

  constructor(message: string, code?: typeof API_ERROR_BACKEND_UNAVAILABLE, apiCode?: string) {
    super(message);
    this.name = 'ApiRequestError';
    this.code = code;
    this.apiCode = apiCode;
  }
}

export function isBackendUnavailableError(error: unknown): boolean {
  return error instanceof ApiRequestError && error.code === API_ERROR_BACKEND_UNAVAILABLE;
}

const BACKEND_UNAVAILABLE_HTTP_STATUSES = new Set([502, 503, 504]);

/** Structured API errors that must not be collapsed into backend-down. */
const UPSTREAM_API_ERROR_CODES = new Set([
  'SOURCE_UNAVAILABLE',
  'UPSTREAM_UNAVAILABLE',
  'CURSEFORGE_UNAVAILABLE',
  'CURSEFORGE_INVALID_KEY',
  'MODS_UNAVAILABLE',
]);

function throwBackendUnavailable(): never {
  throw new ApiRequestError('Backend unavailable', API_ERROR_BACKEND_UNAVAILABLE);
}

function isBackendUnavailableStatus(status: number): boolean {
  return BACKEND_UNAVAILABLE_HTTP_STATUSES.has(status);
}

function isUpstreamApiError(apiCode: string | undefined): boolean {
  return apiCode != null && UPSTREAM_API_ERROR_CODES.has(apiCode);
}

export type TokenResponse = {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
  saved_at?: number;
};

const TOKEN_REFRESH_LEEWAY_MS = 2 * 60 * 1000;

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
  max_memory_mb?: number;
  min_memory_mb?: number;
  extra_jvm_args?: string[];
  window_width?: number;
  window_height?: number;
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

export type MojangLinkStatus = {
  linked: boolean;
  username?: string;
  minecraft_uuid?: string;
  linked_at?: string;
};

export type UserCosmetics = {
  skin_model: ProfileModel;
  has_skin: boolean;
  skin_url?: string;
  has_cape: boolean;
  cape_type?: 'none' | 'qx' | 'custom';
  cape_url?: string;
  updated_at: string;
};

export type SkinCatalogEntry = {
  id: string;
  name: string;
  source: string;
  username: string;
  category: string;
  preview_url: string;
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
  last_error?: string;
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

export type MonitoringInstanceBinding = {
  game_server_id: string;
  instance_id: string;
  instance_name?: string;
  instance_mc_version?: string;
  instance_loader?: string;
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

export type ModSource = 'curseforge' | 'modrinth';

export type ModCatalogSourceFilter = 'all' | ModSource;

export type ModCatalogSort = 'downloads' | 'newest' | 'updated' | 'relevance';

export type ModProjectType = 'mod' | 'modpack' | 'resourcepack' | 'shader' | 'datapack' | 'plugin';

export type GameServerContentKind = 'mod' | 'plugin' | 'datapack';

export type ModCatalogItem = {
  id: string;
  source: ModSource;
  slug: string;
  name: string;
  summary?: string;
  icon_url?: string;
  downloads?: number;
  author?: string;
  project_type: ModProjectType;
  loaders?: string[];
  game_versions?: string[];
  client_side?: string;
  server_side?: string;
  external_url: string;
};

export type ModVersionFile = {
  filename: string;
  url: string;
  sha1?: string;
  size?: number;
};

export type ModVersion = {
  id: string;
  version_number: string;
  game_versions?: string[];
  loaders?: string[];
  files: ModVersionFile[];
  dependencies?: ModDependency[];
  published_at?: string;
};

export type ModDependency = {
  project_id: string;
  project_name?: string;
  source: ModSource;
  dependency_type: 'required' | 'optional' | 'embedded';
  version_id?: string;
  version_number?: string;
  filename?: string;
  download_url?: string;
  file_size?: number;
};

export type ModSyncResult = {
  status: 'queued' | 'already_installed' | 'installed';
  message?: string;
  filename?: string;
  path?: string;
};

export type GameServerContentSyncBody = {
  source: ModSource;
  project_id: string;
  version_id: string;
  filename: string;
  download_url: string;
  project_name?: string;
  version_number?: string;
};

export type InstanceResource = {
  source: ModSource;
  project_id?: string;
  project_name: string;
  version_id?: string;
  version_number?: string;
  filename: string;
  resource_type: ModProjectType;
  icon_url?: string;
  downloads?: number;
  file_size?: number;
  installed_at: string;
};

export type ModInstallRequest = {
  id: string;
  status: string;
  instance_id: string;
  source: ModSource;
  project_id: string;
  project_name: string;
  version_id: string;
  version_number?: string;
  filename: string;
  resource_type: ModProjectType;
  error_code?: string;
  expires_at: string;
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
  launcher_version?: string;
};

export type LauncherRelease = {
  version: string;
  download_url: string;
  filename: string;
};

export type LauncherUpdateRequest = {
  id: string;
  status: string;
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
  const stored: TokenResponse = { ...tokens, saved_at: Date.now() };
  localStorage.setItem(STORAGE_KEY, JSON.stringify(stored));
}

export function clearTokens() {
  localStorage.removeItem(STORAGE_KEY);
  refreshInFlight = null;
}

function accessTokenExpiresAt(accessToken: string): number | null {
  try {
    const segment = accessToken.split('.')[1];
    if (!segment) return null;
    const payload = JSON.parse(atob(segment.replace(/-/g, '+').replace(/_/g, '/'))) as {
      exp?: number;
    };
    return typeof payload.exp === 'number' ? payload.exp * 1000 : null;
  } catch {
    return null;
  }
}

function tokenExpiresAt(tokens: TokenResponse): number | null {
  if (tokens.saved_at && tokens.expires_in > 0) {
    return tokens.saved_at + tokens.expires_in * 1000;
  }
  return accessTokenExpiresAt(tokens.access_token);
}

function tokenNeedsRefresh(tokens: TokenResponse | null): boolean {
  if (!tokens?.refresh_token) return false;
  const expiresAt = tokenExpiresAt(tokens);
  if (!expiresAt) return false;
  return Date.now() + TOKEN_REFRESH_LEEWAY_MS >= expiresAt;
}

let refreshInFlight: Promise<TokenResponse | null> | null = null;

async function refreshTokens(): Promise<TokenResponse | null> {
  const current = loadTokens();
  if (!current?.refresh_token) return null;

  const res = await fetch(`${API_BASE}/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: current.refresh_token }),
  }).catch(() => null);

  if (!res?.ok) return null;

  let tokens: TokenResponse;
  try {
    tokens = (await res.json()) as TokenResponse;
  } catch {
    return null;
  }
  if (!tokens.access_token) return null;
  saveTokens(tokens);
  return tokens;
}

async function ensureFreshTokens(): Promise<void> {
  const tokens = loadTokens();
  if (!tokens || !tokenNeedsRefresh(tokens)) return;

  if (!refreshInFlight) {
    refreshInFlight = refreshTokens().finally(() => {
      refreshInFlight = null;
    });
  }
  await refreshInFlight;
}

/** Refreshes access token when close to expiry; returns false when logged out. */
export async function maintainSession(): Promise<boolean> {
  if (!loadTokens()) return false;
  await ensureFreshTokens();
  return !!loadTokens()?.access_token;
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
  retried = false,
): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set('Content-Type', 'application/json');

  if (auth === true || auth === 'launcher') {
    await ensureFreshTokens();
  }

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
    if (res.status === 401 && auth === true && !retried && loadTokens()?.refresh_token) {
      const refreshed = await refreshTokens();
      if (refreshed) {
        return request<T>(path, init, auth, true);
      }
    }

    let message = res.statusText;
    let apiCode: string | undefined;
    try {
      const body = (await res.json()) as ApiError;
      message = body.error?.message ?? message;
      apiCode = body.error?.code;
    } catch {
      /* ignore */
    }
    const details = { path, status: res.status, message };
    if (res.status >= 500) {
      logger.error('API request failed', details);
    } else {
      logger.warn('API request failed', details);
    }
    if (isBackendUnavailableStatus(res.status) && !isUpstreamApiError(apiCode)) {
      throwBackendUnavailable();
    }
    throw new ApiRequestError(message, undefined, apiCode);
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

  getLauncherRelease: () => request<LauncherRelease>('/launcher/release', {}, false),

  requestLauncherUpdate: () =>
    request<LauncherUpdateRequest>('/launcher/update-requests', { method: 'POST' }),

  changePassword: (body: { current_password: string; new_password: string }) =>
    request<void>('/users/me/password', { method: 'PATCH', body: JSON.stringify(body) }),

  changeEmail: (body: { current_password: string; email: string }) =>
    request<UserProfile>('/users/me/email', { method: 'PATCH', body: JSON.stringify(body) }),

  mojangStatus: () => request<MojangLinkStatus>('/users/me/mojang'),

  startMojangOAuth: () =>
    request<{ authorization_url: string }>('/users/me/mojang/oauth/start', { method: 'POST' }),

  unlinkMojang: () => request<void>('/users/me/mojang', { method: 'DELETE' }),

  getCosmetics: () => request<UserCosmetics>('/users/me/cosmetics'),

  updateCosmetics: (body: { skin_model?: ProfileModel; cape_type?: 'none' | 'qx' | 'custom' }) =>
    request<UserCosmetics>('/users/me/cosmetics', {
      method: 'PUT',
      body: JSON.stringify(body),
    }),

  uploadCosmeticsSkin: async (file: File) => {
    await ensureFreshTokens();
    const form = new FormData();
    form.append('skin', file);
    const headers = new Headers();
    const tokens = loadTokens();
    if (tokens?.access_token) {
      headers.set('Authorization', `Bearer ${tokens.access_token}`);
    }
    const res = await fetch(`${API_BASE}/users/me/cosmetics/skin`, {
      method: 'POST',
      headers,
      body: form,
    });
    if (!res.ok) {
      const err = (await res.json().catch(() => null)) as ApiError | null;
      throw new Error(err?.error?.message ?? `HTTP ${res.status}`);
    }
    return (await res.json()) as UserCosmetics;
  },

  deleteCosmeticsSkin: () =>
    request<UserCosmetics>('/users/me/cosmetics/skin', { method: 'DELETE' }),

  listSkinCatalog: (params?: { category?: string }) => {
    const search = new URLSearchParams();
    if (params?.category) search.set('category', params.category);
    const qs = search.toString();
    return request<{ items: SkinCatalogEntry[] }>(
      `/cosmetics/skin-catalog${qs ? `?${qs}` : ''}`,
    );
  },

  applyCosmeticsSkin: (body: { catalog_id?: string; username?: string }) =>
    request<UserCosmetics>('/users/me/cosmetics/skin/apply', {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  uploadCosmeticsCape: async (file: File) => {
    await ensureFreshTokens();
    const form = new FormData();
    form.append('cape', file);
    const headers = new Headers();
    const tokens = loadTokens();
    if (tokens?.access_token) {
      headers.set('Authorization', `Bearer ${tokens.access_token}`);
    }
    const res = await fetch(`${API_BASE}/users/me/cosmetics/cape`, {
      method: 'POST',
      headers,
      body: form,
    });
    if (!res.ok) {
      const err = (await res.json().catch(() => null)) as ApiError | null;
      throw new Error(err?.error?.message ?? `HTTP ${res.status}`);
    }
    return (await res.json()) as UserCosmetics;
  },

  deleteCosmeticsCape: () =>
    request<UserCosmetics>('/users/me/cosmetics/cape', { method: 'DELETE' }),

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

  updateInstance: (
    id: string,
    body: {
      name?: string;
      max_memory_mb?: number;
      min_memory_mb?: number;
      extra_jvm_args?: string[];
      window_width?: number | null;
      window_height?: number | null;
    },
  ) =>
    request<LauncherInstance>(`/instances/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    }, 'launcher'),

  listInstanceResources: (instanceId: string) =>
    request<{ items: InstanceResource[] }>(
      `/instances/${encodeURIComponent(instanceId)}/resources`,
      { method: 'GET' },
      'launcher',
    ),

  deleteInstanceResource: (
    instanceId: string,
    body: {
      source: ModSource;
      project_id?: string;
      filename?: string;
      resource_type: ModProjectType;
    },
  ) =>
    request<void>(
      `/instances/${encodeURIComponent(instanceId)}/resources`,
      { method: 'DELETE', body: JSON.stringify(body) },
      'launcher',
    ),

  listInstanceFiles: (instanceId: string, path = '') =>
    request<{ items: GameServerFileEntry[] }>(
      `/instances/${encodeURIComponent(instanceId)}/files?path=${encodeURIComponent(path)}`,
      { method: 'GET' },
      'launcher',
    ),

  readInstanceFile: (instanceId: string, path: string) =>
    request<GameServerFileContent>(
      `/instances/${encodeURIComponent(instanceId)}/files/content?path=${encodeURIComponent(path)}`,
      { method: 'GET' },
      'launcher',
    ),

  writeInstanceFile: (instanceId: string, path: string, content: string) =>
    request<{ status: string }>(
      `/instances/${encodeURIComponent(instanceId)}/files/content?path=${encodeURIComponent(path)}`,
      { method: 'PUT', body: JSON.stringify({ content }) },
      'launcher',
    ),

  uploadInstanceResource: async (instanceId: string, file: File, resourceType?: ModProjectType) => {
    const form = new FormData();
    form.append('file', file);
    if (resourceType) form.append('resource_type', resourceType);
    const headers = new Headers();
    const tokens = loadTokens();
    if (tokens?.access_token) {
      headers.set('Authorization', `Bearer ${tokens.access_token}`);
    }
    const res = await fetch(
      `${API_BASE}/instances/${encodeURIComponent(instanceId)}/resources/upload`,
      { method: 'POST', headers, body: form },
    );
    if (!res.ok) {
      const err = (await res.json().catch(() => null)) as ApiError | null;
      throw new Error(err?.error?.message ?? `HTTP ${res.status}`);
    }
    return (await res.json()) as { id: string; status: string; filename: string; resource_type: string };
  },

  uploadGameServerMod: async (vpsId: string, gameServerId: string, file: File) => {
    const form = new FormData();
    form.append('file', file);
    const headers = new Headers();
    const tokens = loadTokens();
    if (tokens?.access_token) {
      headers.set('Authorization', `Bearer ${tokens.access_token}`);
    }
    const res = await fetch(
      `${API_BASE}/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/mods/upload`,
      { method: 'POST', headers, body: form },
    );
    if (!res.ok) {
      const err = (await res.json().catch(() => null)) as ApiError | null;
      throw new Error(err?.error?.message ?? `HTTP ${res.status}`);
    }
    return (await res.json()) as { status: string; filename: string; path?: string };
  },

  createModInstallRequest: (body: {
    instance_id: string;
    source: ModSource;
    project_id: string;
    project_name: string;
    version_id: string;
    version_number?: string;
    filename: string;
    download_url: string;
    resource_type?: ModProjectType;
    icon_url?: string;
    downloads?: number;
    file_size?: number;
  }) =>
    request<ModInstallRequest>(
      '/launcher/mod-install-requests',
      { method: 'POST', body: JSON.stringify(body) },
      'launcher',
    ),

  getModInstallRequest: (id: string) =>
    request<ModInstallRequest>(`/launcher/mod-install-requests/${id}`, { method: 'GET' }, 'launcher'),

  listProfiles: () =>
    request<{ items: OfflineProfile[] }>('/launcher/profiles', { method: 'GET' }, 'launcher'),

  createProfile: (body: { username: string; model?: ProfileModel }) =>
    request<OfflineProfile>('/launcher/profiles', { method: 'POST', body: JSON.stringify(body) }, 'launcher'),

  deleteProfile: (id: string) =>
    request<void>(`/launcher/profiles/${id}`, { method: 'DELETE' }, 'launcher'),

  createLaunchRequest: (body: {
    instance_id: string;
    offline_profile_id?: string;
    use_mojang_account?: boolean;
    join_server_address?: string;
    join_server_port?: number;
  }) =>
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

  listVpsGameServerPlugins: (vpsId: string, gameServerId: string) =>
    request<{ items: GameServerFileEntry[] }>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/plugins`,
    ),

  listVpsGameServerDatapacks: (vpsId: string, gameServerId: string) =>
    request<{ items: GameServerFileEntry[] }>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/datapacks`,
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

  listBindableServers: (params?: { mc_version?: string; loader?: string }) => {
    const search = new URLSearchParams();
    if (params?.mc_version) search.set('mc_version', params.mc_version);
    if (params?.loader) search.set('loader', params.loader);
    const qs = search.toString();
    return request<{ items: MonitoringServer[] }>(
      `/monitoring/bindable-servers${qs ? `?${qs}` : ''}`,
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

  listMonitoringBindings: () =>
    request<{ items: MonitoringInstanceBinding[] }>('/monitoring/bindings'),

  setMonitoringBinding: (gameServerId: string, instanceId: string) =>
    request<MonitoringInstanceBinding>(
      `/monitoring/servers/${encodeURIComponent(gameServerId)}/binding`,
      { method: 'PUT', body: JSON.stringify({ instance_id: instanceId }) },
    ),

  clearMonitoringBinding: (gameServerId: string) =>
    request<void>(`/monitoring/servers/${encodeURIComponent(gameServerId)}/binding`, {
      method: 'DELETE',
    }),

  searchMods: (params: {
    q: string;
    type?: ModProjectType;
    loader?: string;
    mc_version?: string;
    limit?: number;
  }) => {
    const search = new URLSearchParams();
    search.set('q', params.q);
    if (params.type) search.set('type', params.type);
    if (params.loader) search.set('loader', params.loader);
    if (params.mc_version) search.set('mc_version', params.mc_version);
    if (params.limit != null) search.set('limit', String(params.limit));
    return request<{ items: ModCatalogItem[]; curseforge_enabled: boolean }>(
      `/mods/search?${search.toString()}`,
    );
  },

  browseMods: (params: {
    type?: ModProjectType;
    loader?: string;
    mc_version?: string;
    source?: ModCatalogSourceFilter;
    sort?: ModCatalogSort;
    limit?: number;
    offset?: number;
  }) => {
    const search = new URLSearchParams();
    if (params.type) search.set('type', params.type);
    if (params.loader) search.set('loader', params.loader);
    if (params.mc_version) search.set('mc_version', params.mc_version);
    if (params.source) search.set('source', params.source);
    if (params.sort) search.set('sort', params.sort);
    if (params.limit != null) search.set('limit', String(params.limit));
    if (params.offset != null) search.set('offset', String(params.offset));
    const qs = search.toString();
    return request<{ items: ModCatalogItem[]; has_more: boolean; curseforge_enabled: boolean }>(
      `/mods/browse${qs ? `?${qs}` : ''}`,
    );
  },

  getModProject: (source: ModSource, projectId: string) =>
    request<ModCatalogItem & { description?: string }>(
      `/mods/${encodeURIComponent(source)}/${encodeURIComponent(projectId)}`,
    ),

  listModVersions: (
    source: ModSource,
    projectId: string,
    params?: { loader?: string; mc_version?: string },
  ) => {
    const search = new URLSearchParams();
    if (params?.loader) search.set('loader', params.loader);
    if (params?.mc_version) search.set('mc_version', params.mc_version);
    const qs = search.toString();
    return request<{ items: ModVersion[] }>(
      `/mods/${encodeURIComponent(source)}/${encodeURIComponent(projectId)}/versions${qs ? `?${qs}` : ''}`,
    );
  },

  getModVersion: (
    source: ModSource,
    projectId: string,
    versionId: string,
    params?: { loader?: string; mc_version?: string },
  ) => {
    const search = new URLSearchParams();
    if (params?.loader) search.set('loader', params.loader);
    if (params?.mc_version) search.set('mc_version', params.mc_version);
    const qs = search.toString();
    return request<ModVersion>(
      `/mods/${encodeURIComponent(source)}/${encodeURIComponent(projectId)}/versions/${encodeURIComponent(versionId)}${qs ? `?${qs}` : ''}`,
    );
  },

  syncModToGameServer: (
    vpsId: string,
    gameServerId: string,
    body: GameServerContentSyncBody,
  ) =>
    request<ModSyncResult>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/mods/sync`,
      { method: 'POST', body: JSON.stringify(body) },
    ),

  syncPluginToGameServer: (
    vpsId: string,
    gameServerId: string,
    body: GameServerContentSyncBody,
  ) =>
    request<ModSyncResult>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/plugins/sync`,
      { method: 'POST', body: JSON.stringify(body) },
    ),

  syncDatapackToGameServer: (
    vpsId: string,
    gameServerId: string,
    body: GameServerContentSyncBody,
  ) =>
    request<ModSyncResult>(
      `/servers/${encodeURIComponent(vpsId)}/game-servers/${encodeURIComponent(gameServerId)}/datapacks/sync`,
      { method: 'POST', body: JSON.stringify(body) },
    ),
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
