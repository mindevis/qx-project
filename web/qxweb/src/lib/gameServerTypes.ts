export type GameServerTypeGroupId = 'vanilla' | 'plugins' | 'mods' | 'hybrid';

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
  | 'arclight';

export const DEFAULT_GAME_SERVER_TYPE: VpsGameServerType = 'vanilla';

export const GAME_SERVER_TYPE_GROUPS: {
  id: GameServerTypeGroupId;
  types: VpsGameServerType[];
}[] = [
  { id: 'vanilla', types: ['vanilla'] },
  { id: 'plugins', types: ['paper', 'spigot', 'purpur'] },
  { id: 'mods', types: ['forge', 'neoforge', 'fabric', 'quilt'] },
  { id: 'hybrid', types: ['mohist', 'magma', 'arclight'] },
];

export const ALL_GAME_SERVER_TYPES: VpsGameServerType[] = GAME_SERVER_TYPE_GROUPS.flatMap(
  (group) => group.types,
);

const PLUGIN_TYPES = new Set<VpsGameServerType>(['paper', 'spigot', 'purpur']);
const MOD_TYPES = new Set<VpsGameServerType>(['forge', 'neoforge', 'fabric', 'quilt']);
const HYBRID_TYPES = new Set<VpsGameServerType>(['mohist', 'magma', 'arclight']);

export function isKnownGameServerType(value: string): value is VpsGameServerType {
  return (ALL_GAME_SERVER_TYPES as string[]).includes(value);
}

export function gameServerSupportsPlugins(type: VpsGameServerType): boolean {
  return PLUGIN_TYPES.has(type) || HYBRID_TYPES.has(type);
}

export function gameServerSupportsMods(type: VpsGameServerType): boolean {
  return MOD_TYPES.has(type) || HYBRID_TYPES.has(type);
}

export function gameServerSupportsDatapacks(_type: VpsGameServerType): boolean {
  return true;
}

export function pluginLoaderForServerType(type: VpsGameServerType): string {
  switch (type) {
    case 'paper':
    case 'spigot':
    case 'purpur':
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
} {
  return {
    plugins: gameServerSupportsPlugins(type),
    mods: gameServerSupportsMods(type),
    datapacks: gameServerSupportsDatapacks(type),
  };
}

export function gameServerTypeLabelText(t: (key: string) => string, type: string | undefined): string {
  if (!type) return '—';
  if (!isKnownGameServerType(type)) return type;
  return t(`servers.gameServerType.${type}`);
}
