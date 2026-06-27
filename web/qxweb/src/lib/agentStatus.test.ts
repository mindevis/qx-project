import { describe, expect, it } from 'vitest';
import {
  getAgentConnectionStatus,
  getAgentDeployStatus,
  isAgentDeployed,
} from './agentStatus';

describe('isAgentDeployed', () => {
  it('uses explicit flag when present', () => {
    expect(isAgentDeployed({ agent_deployed: true, status: 'pending' })).toBe(true);
    expect(isAgentDeployed({ agent_deployed: false, status: 'pending' })).toBe(false);
  });

  it('infers deploy from lifecycle when flag is missing', () => {
    expect(isAgentDeployed({ status: 'offline' })).toBe(true);
    expect(isAgentDeployed({ status: 'pending' })).toBe(false);
  });
});

describe('getAgentDeployStatus', () => {
  it('detects deploy lifecycle', () => {
    expect(getAgentDeployStatus({ status: 'pending', agent_deployed: false })).toBe('not_deployed');
    expect(getAgentDeployStatus({ status: 'deploying', agent_deployed: true })).toBe('deploying');
    expect(getAgentDeployStatus({ status: 'offline', agent_deployed: true })).toBe('deployed');
    expect(getAgentDeployStatus({ status: 'offline' })).toBe('deployed');
  });
});

describe('getAgentConnectionStatus', () => {
  it('requires deploy before connection state', () => {
    expect(getAgentConnectionStatus({ agent_deployed: false, agent_online: false })).toBe('unavailable');
    expect(getAgentConnectionStatus({ agent_deployed: true, agent_online: true })).toBe('online');
    expect(getAgentConnectionStatus({ agent_deployed: true, agent_online: false })).toBe('offline');
  });
});
