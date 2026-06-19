import { test, expect } from '@playwright/test';
import {
  createServersMockState,
  installMockWebSocket,
  installServersApiMock,
  seedAuthSession,
} from './mock-servers-api';

test.describe('servers admin flow (Flow C web)', () => {
  test('creates VPS, deploys agent, starts server and shows console', async ({ page }) => {
    const state = createServersMockState();
    await installServersApiMock(page, state);
    await installMockWebSocket(page);
    await seedAuthSession(page);

    await page.goto('/servers');
    await expect(page.getByRole('heading', { name: 'Серверы' })).toBeVisible();

    await page.getByRole('button', { name: 'Добавить VPS' }).click();
    await page.getByLabel('Название').fill('FlowC VPS');
    await page.getByLabel('SSH Host').fill('10.0.0.8');
    await page.getByLabel('SSH User').fill('ubuntu');
    await page.getByLabel('SSH Private Key').fill('-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----');
    await page.getByLabel('Путь к server.jar на VPS').fill('/opt/qx/server/server.jar');
    await page.getByRole('button', { name: 'OK' }).click();

    await expect(page).toHaveURL(/\/servers\/srv-1$/);
    await expect(page.getByRole('heading', { name: 'FlowC VPS' })).toBeVisible();
    await expect(page.getByText('Agent оффлайн')).toBeVisible();

    await page.getByRole('button', { name: 'Deploy agent' }).click();
    await expect(page.getByText('Готово')).toBeVisible();
    await expect(page.getByText('Agent подключён')).toBeVisible({ timeout: 10_000 });

    const startBtn = page.getByRole('button', { name: 'Start', exact: true });
    await expect(startBtn).toBeEnabled({ timeout: 10_000 });
    await startBtn.click();
    await expect(page.getByText('Готово')).toBeVisible();
    await expect(page.getByText('Консоль подключена')).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Done (0.05s)!')).toBeVisible();

    expect(state.servers.size).toBe(1);
    const server = state.servers.get('srv-1');
    expect(server?.agent_online).toBe(true);
    expect(server?.status).toBe('online');
  });
});
