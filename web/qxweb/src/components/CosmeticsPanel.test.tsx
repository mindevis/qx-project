import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { testMessage } from '@/test/test-message';
import { api } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import { CosmeticsPanel } from './CosmeticsPanel';

vi.mock('skinview3d', () => ({
  SkinViewer: vi.fn(function MockSkinViewer() {
    return {
      disposed: false,
      background: null,
      autoRotate: false,
      controls: { enableZoom: false, enablePan: false, enableRotate: true },
      loadSkin: vi.fn().mockResolvedValue(undefined),
      loadCape: vi.fn(),
      resetCameraPose: vi.fn(),
      dispose: vi.fn(),
    };
  }),
}));

const cosmetics = {
  skin_model: 'steve' as const,
  has_skin: false,
  has_cape: false,
  cape_type: 'none' as const,
  updated_at: '2026-01-01T00:00:00Z',
};

describe('CosmeticsPanel', () => {
  beforeEach(() => {
    vi.spyOn(api, 'getCosmetics').mockResolvedValue(cosmetics);
    vi.spyOn(api, 'mojangStatus').mockResolvedValue({ linked: false });
    vi.spyOn(api, 'updateCosmetics').mockResolvedValue({
      ...cosmetics,
      skin_model: 'alex',
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('loads and shows cosmetics controls', async () => {
    renderWithTheme(<CosmeticsPanel />);
    await waitFor(() => expect(screen.getByText('QX Skin Server')).toBeInTheDocument());
    expect(screen.getByRole('button', { name: /Загрузить скин/i })).toBeInTheDocument();
  });

  it('updates skin model on picker change', async () => {
    const user = userEvent.setup({ delay: null });
    renderWithTheme(<CosmeticsPanel />);
    await waitFor(() => expect(screen.getByRole('radio', { name: /Alex/i })).toBeInTheDocument());

    await user.click(screen.getByRole('radio', { name: /Alex/i }));
    await waitFor(() =>
      expect(api.updateCosmetics).toHaveBeenCalledWith({ skin_model: 'alex' }),
    );
    expect(testMessage.success).toHaveBeenCalledWith('Настройки скина сохранены');
  });

  it('shows applied status when a custom skin is equipped', async () => {
    vi.spyOn(api, 'getCosmetics').mockResolvedValue({
      ...cosmetics,
      has_skin: true,
      skin_url: '/api/v1/cosmetics/skins/1.png',
    });
    renderWithTheme(<CosmeticsPanel embedded />);
    await waitFor(() => expect(screen.getByText('Скин QX')).toBeInTheDocument());
  });

  it('shows default status without a custom skin', async () => {
    renderWithTheme(<CosmeticsPanel embedded />);
    await waitFor(() => expect(screen.getByText('Стандартный скин')).toBeInTheDocument());
  });

  it('previews the linked Microsoft skin when no QX skin is uploaded', async () => {
    vi.spyOn(api, 'mojangStatus').mockResolvedValue({
      linked: true,
      username: 'CurseSkin',
      minecraft_uuid: 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee',
    });
    renderWithTheme(<CosmeticsPanel embedded />);
    await waitFor(() => expect(screen.getByText('Скин Microsoft')).toBeInTheDocument());
    expect(screen.getByText('CurseSkin')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Скопировать скин Microsoft в QX/i })).toBeInTheDocument();
    expect(screen.queryByText('Стандартный скин')).not.toBeInTheDocument();
  });

  it('keeps a custom QX skin over the Microsoft preview', async () => {
    vi.spyOn(api, 'mojangStatus').mockResolvedValue({
      linked: true,
      username: 'CurseSkin',
      minecraft_uuid: 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee',
    });
    vi.spyOn(api, 'getCosmetics').mockResolvedValue({
      ...cosmetics,
      has_skin: true,
      skin_url: '/api/v1/cosmetics/skins/1.png',
    });
    renderWithTheme(<CosmeticsPanel embedded />);
    await waitFor(() => expect(screen.getByText('Скин QX')).toBeInTheDocument());
    expect(screen.queryByText('Скин Microsoft')).not.toBeInTheDocument();
  });

  it('copies the linked Microsoft skin into QX cosmetics', async () => {
    const user = userEvent.setup({ delay: null });
    vi.spyOn(api, 'mojangStatus').mockResolvedValue({
      linked: true,
      username: 'CurseSkin',
      minecraft_uuid: 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee',
    });
    vi.spyOn(api, 'applyCosmeticsSkin').mockResolvedValue({
      ...cosmetics,
      has_skin: true,
      skin_url: '/api/v1/cosmetics/skins/1.png',
    });
    renderWithTheme(<CosmeticsPanel embedded />);
    await user.click(await screen.findByRole('button', { name: /Скопировать скин Microsoft в QX/i }));
    await waitFor(() =>
      expect(api.applyCosmeticsSkin).toHaveBeenCalledWith({ username: 'CurseSkin' }),
    );
    expect(testMessage.success).toHaveBeenCalledWith('Скин Microsoft скопирован в QX');
  });

  it('renders embedded variant without card title', async () => {
    renderWithTheme(<CosmeticsPanel embedded />);
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Загрузить скин/i })).toBeInTheDocument(),
    );
    expect(screen.queryByText('QX Skin Server')).not.toBeInTheDocument();
  });
});
