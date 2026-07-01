import { test, expect } from '@playwright/test';
import {
  createMockState,
  installLauncherApiMock,
  seedAuthSession,
} from './mock-api';

test.describe('launcher registered flow (Flow A web)', () => {
  test('links device to account and creates instance', async ({ page }) => {
    const state = createMockState();
    await installLauncherApiMock(page, state);
    await seedAuthSession(page);

    await page.goto('/launcher/link?device=flowa-dev');
    const linkButton = page.getByRole('button', { name: 'Связать устройство' });
    await expect(linkButton).toBeEnabled();
    await linkButton.click();
    await expect(page.getByText('Устройство связано')).toBeVisible();
    await expect(
      page.getByText('QXLauncher привязан к вашему аккаунту'),
    ).toBeVisible();
    await page.getByRole('link', { name: 'Перейти к инстансам' }).click();

    await expect(page.getByText('Аккаунт e2e@test.com')).toBeVisible();
    await expect(page.getByText('QXLauncher связан (flowa-dev)')).toBeVisible();
    await page.getByRole('button', { name: /Создать/ }).click();
    await page.getByLabel('Название').fill('FlowA Survival');
    await page.getByRole('button', { name: 'Создать инстанс' }).click();

    await expect(page.getByText('FlowA Survival')).toBeVisible();
    expect(state.instances).toHaveLength(1);
    expect(state.linkedDevice?.owner_type).toBe('user');
  });

  test('plays instance with offline profile', async ({ page }) => {
    const state = createMockState();
    state.linkedDevice = { device_id: 'flowa-dev', owner_type: 'user' };
    state.instances.push({
      id: 'inst-1',
      name: 'Registered Play',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: '2026-06-10T00:00:00Z',
      updated_at: '2026-06-10T00:00:00Z',
    });
    state.profiles.push({
      id: 'prof-1',
      username: 'FlowAPlayer',
      offline_uuid: '00000000-0000-0000-0000-000000000002',
      created_at: '2026-06-10T00:00:00Z',
    });
    await installLauncherApiMock(page, state);
    await seedAuthSession(page);

    await page.goto('/launcher');
    await expect(page.getByText('Мои инстансы')).toBeVisible();
    await expect(page.getByText('Registered Play')).toBeVisible();
    await page.getByRole('button', { name: 'Играть' }).click();
    await expect.poll(() => state.launchRequests.size).toBe(1);
    const launch = [...state.launchRequests.values()][0];
    expect(launch.offline_profile_id).toBe('prof-1');
  });
});
