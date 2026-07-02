import { test, expect } from '@playwright/test';
import {
  createServersMockState,
  installServersApiMock,
  seedAuthSession,
} from './mock-servers-api';

test.describe('servers admin flow (Flow C web)', () => {
  test('creates dedicated server and deploys agent', async ({ page }) => {
    const state = createServersMockState();
    await installServersApiMock(page, state);
    await seedAuthSession(page);

    await page.goto('/servers');
    await expect(page.getByRole('heading', { name: 'Серверы', exact: true })).toBeVisible();

    await page.locator('.servers-hero-actions').getByRole('button', { name: 'Добавить выделенный сервер' }).click();
    await page.getByLabel('Название').fill('FlowC Dedicated');
    await page.getByLabel('SSH Host').fill('10.0.0.8');
    await page.getByLabel('SSH User').fill('ubuntu');
    await page.getByLabel('SSH Private Key').fill('-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----');
    await page.getByRole('button', { name: 'OK' }).click();

    await expect(page).toHaveURL(/\/servers\/srv-1$/);
    await expect(page.getByRole('heading', { name: 'FlowC Dedicated' })).toBeVisible();
    const agentStats = page.locator('.servers-detail-stats .servers-agent-stat-tag');
    await expect(agentStats.filter({ hasText: 'Не развёрнут' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Deploy agent' })).toBeVisible();

    await page.getByRole('button', { name: 'Deploy agent' }).click();
    await expect(agentStats.filter({ hasText: 'Развёрнут' })).toBeVisible({ timeout: 10_000 });
    await expect(agentStats.filter({ hasText: 'Онлайн' })).toBeVisible({ timeout: 10_000 });

    await expect(page.getByRole('button', { name: 'Start', exact: true })).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Stop', exact: true })).toHaveCount(0);

    expect(state.servers.size).toBe(1);
    const server = state.servers.get('srv-1');
    expect(server?.agent_online).toBe(true);
    expect(server?.status).toBe('offline');
  });
});
