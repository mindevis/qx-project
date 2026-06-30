import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { message } from 'antd';
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
    vi.spyOn(message, 'success').mockImplementation(() => undefined as never);
    vi.spyOn(message, 'error').mockImplementation(() => undefined as never);
    vi.spyOn(api, 'getCosmetics').mockResolvedValue(cosmetics);
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
    await waitFor(() => expect(screen.getByText('QX Skin Server')).toBeInTheDocument());

    await user.click(screen.getByRole('radio', { name: /Alex/i }));
    await waitFor(() =>
      expect(api.updateCosmetics).toHaveBeenCalledWith({ skin_model: 'alex' }),
    );
    expect(message.success).toHaveBeenCalledWith('Настройки скина сохранены');
  });

  it('renders embedded variant without card title', async () => {
    renderWithTheme(<CosmeticsPanel embedded />);
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /Загрузить скин/i })).toBeInTheDocument(),
    );
    expect(screen.queryByText('QX Skin Server')).not.toBeInTheDocument();
  });
});
