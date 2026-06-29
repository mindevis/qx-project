import type { GameServer } from '@/api/client';

export type AgentDeployStatus = 'not_deployed' | 'deploying' | 'deployed';
export type AgentConnectionStatus = 'unavailable' | 'online' | 'offline';

/** True once deploy has been started (token issued) or API explicitly says so. */
export function isAgentDeployed(server: Pick<GameServer, 'agent_deployed' | 'status'>): boolean {
  if (server.agent_deployed === true) {
    return true;
  }
  if (server.status === 'pending') {
    return false;
  }
  // Deploy moves the dedicated host out of pending; supports older API responses without agent_deployed.
  return (
    server.status === 'deploying' ||
    server.status === 'offline' ||
    server.status === 'online' ||
    server.status === 'starting' ||
    server.status === 'stopping' ||
    server.status === 'error'
  );
}

export function getAgentDeployStatus(
  server: Pick<GameServer, 'status' | 'agent_deployed'>,
): AgentDeployStatus {
  if (server.status === 'deploying') {
    return 'deploying';
  }
  if (isAgentDeployed(server)) {
    return 'deployed';
  }
  return 'not_deployed';
}

export function getAgentConnectionStatus(
  server: Pick<GameServer, 'agent_deployed' | 'agent_online' | 'status'>,
): AgentConnectionStatus {
  if (!isAgentDeployed(server)) {
    return 'unavailable';
  }
  return server.agent_online ? 'online' : 'offline';
}

export function agentDeployStatusColor(status: AgentDeployStatus): string {
  switch (status) {
    case 'deployed':
      return 'success';
    case 'deploying':
      return 'processing';
    default:
      return 'default';
  }
}

export function agentConnectionStatusColor(status: AgentConnectionStatus): string {
  switch (status) {
    case 'online':
      return 'success';
    case 'offline':
      return 'warning';
    default:
      return 'default';
  }
}
