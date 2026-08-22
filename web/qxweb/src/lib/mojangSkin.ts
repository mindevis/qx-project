function officialAccountId(uuid?: string, username?: string): string | undefined {
  const id = uuid?.trim() || username?.trim();
  return id || undefined;
}

export function officialAccountBodyUrl(uuid?: string, username?: string): string | undefined {
  const id = officialAccountId(uuid, username);
  if (!id) {
    return undefined;
  }
  return `https://mc-heads.net/body/${encodeURIComponent(id)}`;
}

export function officialAccountSkinUrl(uuid?: string, username?: string): string | undefined {
  const id = officialAccountId(uuid, username);
  if (!id) {
    return undefined;
  }
  return `https://mc-heads.net/skin/${encodeURIComponent(id)}`;
}
