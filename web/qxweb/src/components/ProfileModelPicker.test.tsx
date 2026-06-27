import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { I18nProvider } from '@/i18n/I18nContext';
import { ProfileModelPicker } from './ProfileModelPicker';

vi.mock('skinview3d', () => ({
  SkinViewer: vi.fn(function MockSkinViewer() {
    return {
      loadSkin: vi.fn().mockResolvedValue(undefined),
      dispose: vi.fn(),
      disposed: false,
      background: null,
      autoRotate: false,
      controls: { enableZoom: false, enablePan: false, enableRotate: true },
      resetCameraPose: vi.fn(),
    };
  }),
}));

describe('ProfileModelPicker', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders Steve and Alex options and reports selection', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    render(
      <I18nProvider>
        <ProfileModelPicker value="steve" onChange={onChange} />
      </I18nProvider>,
    );

    expect(screen.getByRole('radio', { name: /Steve/i })).toHaveAttribute('aria-checked', 'true');
    await user.click(screen.getByRole('radio', { name: /Alex/i }));
    expect(onChange).toHaveBeenCalledWith('alex');
  });
});
