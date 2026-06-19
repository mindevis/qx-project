/** MVP-supported Vanilla Minecraft releases (see docs/mvp.md). */
export const MVP_MC_VERSIONS = ['1.20.4', '1.21', '1.21.1'] as const;

export const DEFAULT_MC_VERSION: (typeof MVP_MC_VERSIONS)[number] = '1.21';
