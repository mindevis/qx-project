import { describe, expect, it } from 'vitest';
import { renderWithTheme } from '@/test/test-utils';
import { LauncherCodeSigningNotice } from './LauncherCodeSigningNotice';

describe('LauncherCodeSigningNotice', () => {
  it('shows SignPath and privacy links', () => {
    renderWithTheme(<LauncherCodeSigningNotice />);
    expect(document.body.textContent).toContain('SignPath Foundation');
    expect(document.body.querySelector('a[href="https://docs.qx-dev.ru/privacy/"]')).toBeTruthy();
  });
});
