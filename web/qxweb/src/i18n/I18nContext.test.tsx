import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { I18nProvider, useI18n } from './I18nContext';

function LocaleProbe() {
  const { locale } = useI18n();
  return <span data-testid="locale">{locale}</span>;
}

describe('I18nProvider', () => {
  it('uses stored locale', () => {
    window.localStorage.setItem('qxweb-locale', 'en');
    render(
      <I18nProvider>
        <LocaleProbe />
      </I18nProvider>,
    );
    expect(screen.getByTestId('locale')).toHaveTextContent('en');
    expect(document.documentElement.lang).toBe('en');
  });

  it('throws outside provider', () => {
    expect(() => render(<LocaleProbe />)).toThrow('useI18n must be used within I18nProvider');
  });
});
