const HOSTS = {
  forge: 'https://files.minecraftforge.net',
  mavenforge: 'https://maven.minecraftforge.net',
  papermc: 'https://api.papermc.io',
  purpur: 'https://api.purpurmc.org',
  neoforge: 'https://maven.neoforged.net',
  fabric: 'https://meta.fabricmc.net',
  quilt: 'https://meta.quiltmc.org',
  mohist: 'https://mohistmc.com',
  magma: 'https://magmafoundation.org',
  arclight: 'https://files.hypertention.cn',
} as const;

export type GameServerUpstreamHost = keyof typeof HOSTS;

export function gameServerUpstreamUrl(host: GameServerUpstreamHost, path: string): string {
  const normalized = path.startsWith('/') ? path : `/${path}`;
  if (import.meta.env.DEV) {
    return `/upstream/${host}${normalized}`;
  }
  return `${HOSTS[host]}${normalized}`;
}
