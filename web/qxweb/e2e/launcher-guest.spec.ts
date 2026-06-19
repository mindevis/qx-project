import { test, expect } from '@playwright/test';
import {
  createMockState,
  installLauncherApiMock,
  seedGuestSession,
} from './mock-api';

test.describe('launcher guest flow (Flow B web)', () => {
  test('links device and creates instance', async ({ page }) => {
    const state = createMockState();
    await installLauncherApiMock(page, state);
    await seedGuestSession(page);

    await page.goto('/launcher/link?device=dev-e2e');
    await page.getByRole('button', { name: /Продолжить как гость/ }).click();
    await expect(page.getByText('Устройство связано')).toBeVisible();
    await page.getByRole('link', { name: 'Перейти к инстансам' }).click();

    await expect(page.getByText('Гостевой режим')).toBeVisible();
    await page.getByRole('button', { name: /Создать/ }).click();
    await page.getByLabel('Название').fill('E2E Survival');
    await page.getByRole('button', { name: 'Создать Vanilla' }).click();

    await expect(page.getByText('E2E Survival')).toBeVisible();
    expect(state.instances).toHaveLength(1);
  });

  test('plays instance via launch-bridge poll', async ({ page }) => {
    const state = createMockState();
    state.instances.push({
      id: 'inst-1',
      name: 'Play Test',
      mc_version: '1.21',
      loader: 'vanilla',
      created_at: '2026-06-10T00:00:00Z',
      updated_at: '2026-06-10T00:00:00Z',
    });
    await installLauncherApiMock(page, state);
    await seedGuestSession(page);

    await page.goto('/launcher');
    await expect(page.getByText('Play Test')).toBeVisible();
    await page.getByRole('button', { name: 'Играть' }).click();
    await expect(page.getByText('Игра запущена')).toBeVisible({ timeout: 15_000 });
    expect(state.launchRequests.size).toBe(1);
  });
});
