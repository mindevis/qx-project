import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { message } from 'antd';
import { api } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import { SkinCatalogPanel } from './SkinCatalogPanel';

const catalogItems = [
  {
    id: 'skin-1',
    name: 'Notch',
    username: 'Notch',
    preview_url: 'https://example.com/notch.png',
    category: 'popular',
  },
  {
    id: 'skin-2',
    name: 'Herobrine',
    username: 'Herobrine',
    preview_url: 'https://example.com/herobrine.png',
    category: 'classic',
  },
];

describe('SkinCatalogPanel', () => {
  beforeEach(() => {
    vi.spyOn(message, 'success').mockImplementation(() => undefined as never);
    vi.spyOn(message, 'error').mockImplementation(() => undefined as never);
    vi.spyOn(api, 'listSkinCatalog').mockResolvedValue({ items: catalogItems });
    vi.spyOn(api, 'applyCosmeticsSkin').mockResolvedValue(undefined);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders catalog filter and username fields separately', async () => {
    renderWithTheme(<SkinCatalogPanel />);
    await waitFor(() => expect(screen.getByText('Notch')).toBeInTheDocument());

    expect(screen.getByPlaceholderText('Фильтр каталога…')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Ник Minecraft')).toBeInTheDocument();
  });

  it('filters catalog locally without calling apply API', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithTheme(<SkinCatalogPanel />);
    await waitFor(() => expect(screen.getByText('Notch')).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText('Фильтр каталога…'), 'Herobrine');
    await waitFor(() => {
      expect(screen.getByText('Herobrine')).toBeInTheDocument();
      expect(screen.queryByText('Notch')).not.toBeInTheDocument();
    });
    expect(api.applyCosmeticsSkin).not.toHaveBeenCalled();
  });

  it('applies skin by Mojang username', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithTheme(<SkinCatalogPanel />);
    await waitFor(() => expect(screen.getByText('Notch')).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText('Ник Minecraft'), 'Steve');
    await user.click(screen.getByRole('button', { name: 'Применить по нику' }));

    await waitFor(() =>
      expect(api.applyCosmeticsSkin).toHaveBeenCalledWith({ username: 'Steve' }),
    );
    expect(message.success).toHaveBeenCalledWith('Скин применён');
  });
  it('applies skin from catalog entry', async () => {
    const user = userEvent.setup({ delay: null });
    const onApplied = vi.fn();
    renderWithTheme(<SkinCatalogPanel onApplied={onApplied} />);
    await waitFor(() => expect(screen.getByText('Notch')).toBeInTheDocument());

    const selectButtons = screen.getAllByRole('button', {
      name: /\u0412\u044b\u0431\u0440\u0430\u0442\u044c/i,
    });
    await user.click(selectButtons[0]);

    await waitFor(() =>
      expect(api.applyCosmeticsSkin).toHaveBeenCalledWith({ catalog_id: 'skin-1' }),
    );
    expect(onApplied).toHaveBeenCalled();
  });

  it('reloads catalog when category changes', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithTheme(<SkinCatalogPanel />);
    await waitFor(() => expect(screen.getByText('Notch')).toBeInTheDocument());
    vi.mocked(api.listSkinCatalog).mockClear();

    await user.click(screen.getByRole('combobox'));
    await user.click(await screen.findByText(/\u041a\u043b\u0430\u0441\u0441\u0438\u043a\u0430/i));

    await waitFor(() =>
      expect(api.listSkinCatalog).toHaveBeenCalledWith({ category: 'classic' }),
    );
  });
});
