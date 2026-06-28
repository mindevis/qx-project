import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { saveTokens, clearTokens } from '@/api/client';
import {
  addVpsGameServer,
  isVpsGameServerProvisioning,
  listVpsGameServers,
  reinstallVpsGameServer,
  removeVpsGameServer,
  restartVpsGameServer,
  startVpsGameServer,
  stopVpsGameServer,
  suggestDefaultGamePort,
  updateVpsGameServer,
} from './vpsGameServers';

const sampleItem = {
  id: 'gs-1',
  name: 'Survival',
  server_type: 'forge',
  mc_version: '1.20.1',
  loader_version: '47.2.0',
  address: 'localhost',
  port: 25565,
  rcon_password: 'secret',
  rcon_port: 25575,
  status: 'running',
  created_at: 'now',
};

describe('vpsGameServers', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    saveTokens({
      access_token: 'a',
      refresh_token: 'r',
      token_type: 'Bearer',
      expires_in: 60,
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    clearTokens();
  });

  it('suggests next free game port', () => {
    expect(suggestDefaultGamePort([])).toBe(25565);
    expect(suggestDefaultGamePort([{ ...sampleItem, port: 25565 }])).toBe(25566);
  });

  it('detects provisioning statuses', () => {
    expect(isVpsGameServerProvisioning('installing')).toBe(true);
    expect(isVpsGameServerProvisioning('starting')).toBe(true);
    expect(isVpsGameServerProvisioning('running')).toBe(false);
  });

  it('lists mapped game servers', async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ items: [sampleItem] }), { status: 200 }),
    );
    const items = await listVpsGameServers('srv-1');
    expect(items[0]?.server_type).toBe('forge');
    expect(items[0]?.rcon_port).toBe(25575);
  });

  it('creates, updates, and removes game servers', async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock
      .mockResolvedValueOnce(new Response(JSON.stringify(sampleItem), { status: 201 }))
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ...sampleItem, name: 'Renamed' }), { status: 200 }),
      )
      .mockResolvedValueOnce(new Response(null, { status: 204 }));

    const created = await addVpsGameServer('srv-1', {
      name: 'Survival',
      mc_version: '1.20.1',
      loader_version: '47.2.0',
    });
    expect(created.name).toBe('Survival');

    const updated = await updateVpsGameServer('srv-1', 'gs-1', { name: 'Renamed' });
    expect(updated.name).toBe('Renamed');

    await removeVpsGameServer('srv-1', 'gs-1');
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it('runs power and reinstall actions', async () => {
    const fetchMock = vi.mocked(fetch);
    const responding = (status: string) =>
      new Response(JSON.stringify({ ...sampleItem, status }), { status: 200 });

    fetchMock
      .mockResolvedValueOnce(responding('starting'))
      .mockResolvedValueOnce(responding('stopped'))
      .mockResolvedValueOnce(responding('starting'))
      .mockResolvedValueOnce(responding('installing'));

    await expect(startVpsGameServer('srv-1', 'gs-1')).resolves.toMatchObject({ status: 'starting' });
    await expect(stopVpsGameServer('srv-1', 'gs-1')).resolves.toMatchObject({ status: 'stopped' });
    await expect(restartVpsGameServer('srv-1', 'gs-1')).resolves.toMatchObject({ status: 'starting' });
    await expect(reinstallVpsGameServer('srv-1', 'gs-1')).resolves.toMatchObject({
      status: 'installing',
    });
  });
});
