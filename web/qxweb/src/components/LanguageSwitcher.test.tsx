import { describe, expect, it, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { I18nProvider, useI18n } from '@/i18n/I18nContext';
import { LanguageSwitcher } from './LanguageSwitcher';
import { LOCALE_STORAGE_KEY } from '@/i18n';

function Probe() {
  const { locale, t } = useI18n();
  return (
    <div>
      <span data-testid="locale">{locale}</span>
      <span data-testid="home-title">{t('home.title')}</span>
    </div>
  );
}

describe('LanguageSwitcher', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('switches to English and persists locale', async () => {
    const user = userEvent.setup({ delay: null });

    render(
      <I18nProvider>
        <LanguageSwitcher />
        <Probe />
      </I18nProvider>,
    );

    expect(screen.getByTestId('locale')).toHaveTextContent('ru');
    expect(screen.getByTestId('home-title')).toHaveTextContent('Единая экосистема для Minecraft');

    await user.click(screen.getByText('EN'));

    expect(screen.getByTestId('locale')).toHaveTextContent('en');
    expect(screen.getByTestId('home-title')).toHaveTextContent('Unified ecosystem for Minecraft');
    expect(window.localStorage.getItem(LOCALE_STORAGE_KEY)).toBe('en');
    expect(document.documentElement.lang).toBe('en');
  });
});
