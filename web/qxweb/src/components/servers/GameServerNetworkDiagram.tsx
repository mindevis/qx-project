import { Link } from 'react-router-dom';
import { useI18n } from '@/i18n/I18nContext';
import type { GameServerNetworkMember } from '@/api/client';
import {
  layoutGameServerNetwork,
  nodeAnchor,
  type NetworkDiagramEdge,
  type NetworkDiagramNode,
} from '@/lib/gameServerNetworkLayout';
import './GameServerNetworkDiagram.css';

export function GameServerNetworkDiagram({
  vpsId,
  members,
}: {
  vpsId: string;
  members: GameServerNetworkMember[];
}) {
  const { t } = useI18n();
  const layout = layoutGameServerNetwork(members, {
    players: t('servers.networkDiagramPlayers'),
  });
  const byId = new Map(layout.nodes.map((node) => [node.id, node]));

  return (
    <div className="network-diagram" style={{ minHeight: layout.height }}>
      <svg
        className="network-diagram-svg"
        viewBox={`0 0 ${layout.width} ${layout.height}`}
        width={layout.width}
        height={layout.height}
        role="img"
        aria-label={t('servers.networksTitle')}
      >
        <defs>
          <marker id="network-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="8" markerHeight="8" orient="auto-start-reverse">
            <path d="M 0 0 L 10 5 L 0 10 z" className="network-diagram-arrowhead" />
          </marker>
        </defs>
        {layout.edges.map((edge) => {
          const path = edgePath(byId.get(edge.from), byId.get(edge.to), edge.kind);
          if (!path) return null;
          return (
            <g key={edge.id}>
              <path
                d={path.d}
                className={`network-diagram-edge network-diagram-edge--${edge.kind}`}
                markerEnd="url(#network-arrow)"
              />
              {path.labelX != null && path.labelY != null ? (
                <text
                  x={path.labelX}
                  y={path.labelY}
                  className="network-diagram-edge-label"
                  textAnchor="middle"
                >
                  {edgeLabel(t, edge.kind)}
                </text>
              ) : null}
            </g>
          );
        })}
      </svg>
      {layout.nodes.map((node) => {
        const className = [
          'network-diagram-node',
          `network-diagram-node--${node.kind}`,
          node.role ? `network-diagram-node--${node.role}` : '',
        ]
          .filter(Boolean)
          .join(' ');
        const inner = (
          <>
            <span className="network-diagram-node-title">{node.label}</span>
            {node.subtitle ? <span className="network-diagram-node-sub">{node.subtitle}</span> : null}
            {node.role ? (
              <span className="network-diagram-node-role">{roleLabel(t, node.role)}</span>
            ) : null}
          </>
        );
        const style = { left: node.x, top: node.y, width: node.width, height: node.height };
        if (node.kind === 'server' && node.gameServerId) {
          return (
            <Link
              key={node.id}
              to={`/servers/${vpsId}/game-servers/${node.gameServerId}`}
              className={className}
              style={style}
            >
              {inner}
            </Link>
          );
        }
        return (
          <div key={node.id} className={className} style={style}>
            {inner}
          </div>
        );
      })}
    </div>
  );
}

function roleLabel(
  t: (key: string) => string,
  role: NonNullable<NetworkDiagramNode['role']>,
): string {
  if (role === 'proxy') return t('servers.networkRoleProxy');
  if (role === 'lobby') return t('servers.networkRoleLobby');
  return t('servers.networkRoleBackend');
}

function edgeLabel(t: (key: string) => string, kind: NetworkDiagramEdge['kind']): string {
  if (kind === 'try') return t('servers.networkDiagramTry');
  if (kind === 'transfer') return t('servers.networkDiagramTransfer');
  return t('servers.networkDiagramConnect');
}

function edgePath(
  from: NetworkDiagramNode | undefined,
  to: NetworkDiagramNode | undefined,
  kind: NetworkDiagramEdge['kind'],
): { d: string; labelX: number; labelY: number } | null {
  if (!from || !to) return null;
  if (kind === 'transfer') {
    const start = nodeAnchor(from, from.x < to.x ? 'right' : 'left');
    const end = nodeAnchor(to, from.x < to.x ? 'left' : 'right');
    const midY = start.y - 18;
    const d = `M ${start.x} ${start.y} C ${start.x} ${midY}, ${end.x} ${midY}, ${end.x} ${end.y}`;
    return { d, labelX: (start.x + end.x) / 2, labelY: midY - 4 };
  }
  const start = nodeAnchor(from, 'bottom');
  const end = nodeAnchor(to, 'top');
  const midY = (start.y + end.y) / 2;
  const d = `M ${start.x} ${start.y} C ${start.x} ${midY}, ${end.x} ${midY}, ${end.x} ${end.y}`;
  return { d, labelX: (start.x + end.x) / 2, labelY: midY - 2 };
}
