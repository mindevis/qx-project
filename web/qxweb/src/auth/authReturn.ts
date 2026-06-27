export function isSafeReturnPath(path: string): boolean {
  if (!path.startsWith('/') || path.startsWith('//')) {
    return false;
  }
  if (path === '/auth/login' || path === '/auth/register' || path.startsWith('/auth/')) {
    return false;
  }
  return true;
}

export function buildAuthReturnPath(pathname: string, search = '', hash = ''): string {
  const path = `${pathname}${search}${hash}`;
  return isSafeReturnPath(path) ? path : '/';
}
