export type GameServerTypeGroupId = 'vanilla' | 'plugins' | 'mods' | 'hybrid' | 'proxy';

export type VpsGameServerType =
  | 'vanilla'
  | 'paper'
  | 'spigot'
  | 'purpur'
  | 'forge'
  | 'neoforge'
  | 'fabric'
  | 'quilt'
  | 'mohist'
  | 'magma'
  | 'arclight'
  | 'velocity';

export const DEFAULT_GAME_SERVER_TYPE: VpsGameServerType = 'vanilla';

export const GAME_SERVER_TYPE_GROUPS: {
  id: GameServerTypeGroupId;
  types: VpsGameServerType[];
}[] = [
  { id: 'vanilla', types: ['vanilla'] },
  { id: 'plugins', types: ['paper', 'spigot', 'purpur'] },
  { id: 'mods', types: ['forge', 'neoforge', 'fabric', 'quilt'] },
  { id: 'hybrid', types: ['mohist', 'magma', 'arclight'] },
  { id: 'proxy', types: ['velocity'] },
];

export const ALL_GAME_SERVER_TYPES: VpsGameServerType[] = GAME_SERVER_TYPE_GROUPS.flatMap(
  (group) => group.types,
);

const PLUGIN_TYPES = new Set<VpsGameServerType>(['paper', 'spigot', 'purpur', 'velocity']);
const MOD_TYPES = new Set<VpsGameServerType>(['forge', 'neoforge', 'fabric', 'quilt']);
const HYBRID_TYPES = new Set<VpsGameServerType>(['mohist', 'magma', 'arclight']);
const PROXY_TYPES = new Set<VpsGameServerType>(['velocity']);

export function isKnownGameServerType(value: string): value is VpsGameServerType {
  return (ALL_GAME_SERVER_TYPES as string[]).includes(value);
}

export function isProxyGameServerType(type: VpsGameServerType): boolean {
  return PROXY_TYPES.has(type);
}

function gameServerUsesMinecraftVersion(type: VpsGameServerType): boolean {
  return !isProxyGameServerType(type);
}

export function gameServerHasProperties(type: VpsGameServerType): boolean {
  return !isProxyGameServerType(type);
}

export function gameServerSupportsPlugins(type: VpsGameServerType): boolean {
  return PLUGIN_TYPES.has(type) || HYBRID_TYPES.has(type);
}

export function gameServerSupportsMods(type: VpsGameServerType): boolean {
  return MOD_TYPES.has(type) || HYBRID_TYPES.has(type);
}

export function gameServerSupportsDatapacks(type: VpsGameServerType): boolean {
  return !isProxyGameServerType(type);
}

export function gameServerSupportsClientContent(type: VpsGameServerType): boolean {
  return !isProxyGameServerType(type);
}

export function gameServerCatalogTabs(type: VpsGameServerType): Array<
  'mod' | 'resourcepack' | 'shader' | 'datapack'
> {
  const tabs: Array<'mod' | 'resourcepack' | 'shader' | 'datapack'> = [];
  if (gameServerSupportsMods(type)) {
    tabs.push('mod');
  }
  if (gameServerSupportsClientContent(type)) {
    tabs.push('resourcepack', 'shader');
  }
  if (gameServerSupportsDatapacks(type)) {
    tabs.push('datapack');
  }
  return tabs;
}

export function pluginLoaderForServerType(type: VpsGameServerType): string {
  switch (type) {
    case 'paper':
    case 'spigot':
    case 'purpur':
    case 'velocity':
      return type;
    case 'mohist':
    case 'magma':
    case 'arclight':
      return 'bukkit';
    default:
      return 'paper';
  }
}

export function gameServerTypeCapabilities(type: VpsGameServerType): {
  plugins: boolean;
  mods: boolean;
  datapacks: boolean;
  clientContent: boolean;
} {
  return {
    plugins: gameServerSupportsPlugins(type),
    mods: gameServerSupportsMods(type),
    datapacks: gameServerSupportsDatapacks(type),
    clientContent: gameServerSupportsClientContent(type),
  };
}

export function gameServerTypeLabelText(t: (key: string) => string, type: string | undefined): string {
  if (!type) return '—';
  if (!isKnownGameServerType(type)) return type;
  return t(`servers.gameServerType.${type}`);
}

export function gameServerPrimaryVersionI18nKey(type: VpsGameServerType | undefined): string {
  if (type && !gameServerUsesMinecraftVersion(type)) {
    return 'servers.gameServerProxyVersion';
  }
  return 'servers.gameServerMcVersion';
}

export function gameServerPrimaryVersionRequiredI18nKey(type: VpsGameServerType | undefined): string {
  if (type && !gameServerUsesMinecraftVersion(type)) {
    return 'servers.gameServerProxyVersionRequired';
  }
  return 'servers.gameServerMcVersionRequired';
}
