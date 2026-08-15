import { describe, expect, it } from 'vitest';
import { getServerPropertyHint, getServerPropertyMeta } from '@/lib/serverPropertyHints';

describe('serverPropertyHints', () => {
  it('returns a Russian title and description for known keys', () => {
    const motd = getServerPropertyMeta('ru', 'motd');
    expect(motd.title).toBe('Сообщение дня');
    expect(motd.description).toMatch(/списке серверов/i);
    expect(getServerPropertyMeta('ru', 'online-mode').title).toMatch(/лицензион/i);
    expect(getServerPropertyHint('ru', 'online-mode')).toMatch(/Mojang/i);
  });

  it('falls back to the raw key when the setting is unknown', () => {
    expect(getServerPropertyMeta('ru', 'custom-unknown-key')).toEqual({
      title: 'custom-unknown-key',
    });
    expect(getServerPropertyHint('ru', 'custom-unknown-key')).toBeUndefined();
  });
});
