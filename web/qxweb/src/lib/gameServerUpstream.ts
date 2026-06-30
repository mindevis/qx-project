export type GameServerUpstreamHost =
  | 'forge'
  | 'mavenforge'
  | 'papermc'
  | 'purpur'
  | 'neoforge'
  | 'fabric'
  | 'quilt'
  | 'mohist'
  | 'magma'
  | 'arclight';

/** Same-origin path proxied by Vite (dev) and edge nginx (prod); see vite.proxy.ts / upstream-proxies.conf */
export function gameServerUpstreamUrl(host: GameServerUpstreamHost, path: string): string {
  const normalized = path.startsWith('/') ? path : `/${path}`;
  return `/upstream/${host}${normalized}`;
}
