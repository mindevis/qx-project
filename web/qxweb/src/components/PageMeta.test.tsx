import { render, waitFor } from '@testing-library/react';
import type { ComponentProps } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, beforeEach, afterEach } from 'vitest';
import { PageMeta } from '@/components/PageMeta';
import { I18nProvider } from '@/i18n/I18nContext';

function renderPageMeta(props: ComponentProps<typeof PageMeta>) {
  return render(
    <I18nProvider>
      <MemoryRouter>
        <PageMeta {...props} />
      </MemoryRouter>
    </I18nProvider>,
  );
}

describe('PageMeta', () => {
  beforeEach(() => {
    document.head.querySelectorAll('meta[data-qx-test]').forEach((node) => node.remove());
    document.head.querySelectorAll('link[rel="canonical"]').forEach((node) => node.remove());
  });

  afterEach(() => {
    document.title = '';
  });

  it('sets document title and description meta', async () => {
    renderPageMeta({
      titleKey: 'seo.pages.home.title',
      descriptionKey: 'seo.pages.home.description',
      pathname: '/',
    });

    await waitFor(() => {
      expect(document.title).toBe('QXSystem — экосистема Minecraft');
    });

    expect(document.querySelector('meta[name="description"]')?.getAttribute('content')).toContain(
      'лаунчер',
    );
    expect(document.querySelector('meta[property="og:title"]')?.getAttribute('content')).toBe(
      'QXSystem — экосистема Minecraft',
    );
    expect(document.querySelector('link[rel="canonical"]')?.getAttribute('href')).toMatch(/\/$/);
  });

  it('sets noindex for auth routes', async () => {
    renderPageMeta({
      titleKey: 'seo.pages.auth.title',
      descriptionKey: 'seo.pages.auth.description',
      pathname: '/auth/login',
      noIndex: true,
    });

    await waitFor(() => {
      expect(document.querySelector('meta[name="robots"]')?.getAttribute('content')).toBe(
        'noindex, nofollow',
      );
    });

    expect(document.querySelector('link[rel="canonical"]')).toBeNull();
  });
});
