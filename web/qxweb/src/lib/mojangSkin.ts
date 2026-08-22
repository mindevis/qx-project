export function officialAccountBodyUrl(uuid?: string, username?: string): string | undefined {
  const id = uuid?.trim() || username?.trim();
  if (!id) {
    return undefined;
  }
  return `https://mc-heads.net/body/${encodeURIComponent(id)}`;
}
