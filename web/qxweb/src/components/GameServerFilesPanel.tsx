import { useCallback, useEffect, useState, type MouseEvent } from 'react';
import {
  Breadcrumb,
  Button,
  Empty,
  Input,
  Popconfirm,
  Space,
  Spin,
  Table,
  Typography,
} from 'antd';
import { FolderOutlined, FileOutlined, ArrowLeftOutlined, DeleteOutlined } from '@ant-design/icons';
import { api, type GameServerFileEntry } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';

type GameServerFilesPanelProps = {
  vpsId: string;
  gameServerId: string;
  agentOnline: boolean;
  rootPath?: string;
  highlightExtensions?: string[];
};

function formatFileSize(size?: number): string {
  if (size == null || size <= 0) return '—';
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function pathSegments(path: string, rootPath?: string): { label: string; path: string }[] {
  const normalizedRoot = rootPath?.replace(/^\/+|\/+$/g, '') ?? '';
  const relative = normalizedRoot && path.startsWith(normalizedRoot)
    ? path.slice(normalizedRoot.length).replace(/^\//, '')
    : path;
  if (!relative) {
    return [{ label: normalizedRoot || '/', path: normalizedRoot }];
  }
  const parts = relative.split('/').filter(Boolean);
  const crumbs = [{ label: normalizedRoot || '/', path: normalizedRoot }];
  let current = normalizedRoot;
  for (const part of parts) {
    current = current ? `${current}/${part}` : part;
    crumbs.push({ label: part, path: current });
  }
  return crumbs;
}

function isHighlighted(name: string, extensions?: string[]): boolean {
  if (!extensions?.length) return false;
  const lower = name.toLowerCase();
  return extensions.some((ext) => lower.endsWith(ext.toLowerCase()));
}

export function GameServerFilesPanel({
  vpsId,
  gameServerId,
  agentOnline,
  rootPath,
  highlightExtensions,
}: GameServerFilesPanelProps) {
  const { t } = useI18n();
  const message = useMessage();
  const initialPath = rootPath?.replace(/^\/+|\/+$/g, '') ?? '';
  const [currentPath, setCurrentPath] = useState(initialPath);
  const [entries, setEntries] = useState<GameServerFileEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [fileContent, setFileContent] = useState('');
  const [fileLoading, setFileLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deletingPath, setDeletingPath] = useState<string>();

  const normalizedRoot = rootPath?.replace(/^\/+|\/+$/g, '') ?? '';

  const navigateTo = (path: string) => {
    if (normalizedRoot && path && !path.startsWith(normalizedRoot)) {
      setCurrentPath(normalizedRoot);
      return;
    }
    setCurrentPath(path);
  };

  const loadDir = useCallback(async () => {
    if (!agentOnline) {
      setEntries([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const res = await api.listVpsGameServerFiles(vpsId, gameServerId, currentPath);
      setEntries(res.items ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.filesLoadFailed'));
    } finally {
      setLoading(false);
    }
  }, [agentOnline, currentPath, gameServerId, message, t, vpsId]);

  useEffect(() => {
    void loadDir();
  }, [loadDir]);

  const openFile = async (path: string) => {
    setSelectedFile(path);
    setFileLoading(true);
    try {
      const res = await api.readVpsGameServerFile(vpsId, gameServerId, path);
      setFileContent(res.content);
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.fileReadFailed'));
      setSelectedFile(null);
    } finally {
      setFileLoading(false);
    }
  };

  const saveFile = async () => {
    if (!selectedFile) return;
    setSaving(true);
    try {
      await api.writeVpsGameServerFile(vpsId, gameServerId, selectedFile, fileContent);
      message.success(t('gameServerDetail.fileSaved'));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('common.error'));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (row: GameServerFileEntry) => {
    setDeletingPath(row.path);
    try {
      await api.deleteVpsGameServerFile(vpsId, gameServerId, row.path);
      message.success(t('gameServerDetail.fileDeleted'));
      if (selectedFile === row.path || selectedFile?.startsWith(`${row.path}/`)) {
        setSelectedFile(null);
        setFileContent('');
      }
      await loadDir();
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('gameServerDetail.fileDeleteFailed'));
    } finally {
      setDeletingPath(undefined);
    }
  };

  if (!agentOnline) {
    return (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={t('servers.gameServersAgentRequired')}
      />
    );
  }

  if (selectedFile) {
    return (
      <div className="game-server-files-editor">
        <Space className="game-server-files-toolbar">
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => {
              setSelectedFile(null);
              setFileContent('');
            }}
          >
            {t('gameServerDetail.backToFiles')}
          </Button>
          <Typography.Text code>{selectedFile}</Typography.Text>
          <Button type="primary" loading={saving} onClick={() => void saveFile()}>
            {t('common.save')}
          </Button>
        </Space>
        {fileLoading ? (
          <div className="servers-loading">
            <Spin />
          </div>
        ) : (
          <Input.TextArea
            className="game-server-files-textarea"
            value={fileContent}
            rows={18}
            onChange={(e) => setFileContent(e.target.value)}
          />
        )}
      </div>
    );
  }

  return (
    <div className="game-server-files">
      <Breadcrumb
        className="game-server-files-breadcrumb"
        items={pathSegments(currentPath, normalizedRoot).map((crumb) => ({
          title: (
            <button
              type="button"
              className="game-server-files-crumb"
              onClick={() => navigateTo(crumb.path)}
            >
              {crumb.label}
            </button>
          ),
        }))}
      />
      {loading ? (
        <div className="servers-loading">
          <Spin />
        </div>
      ) : entries.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('gameServerDetail.filesEmpty')} />
      ) : (
        <Table
          className="game-server-files-table"
          rowKey="path"
          size="small"
          pagination={false}
          dataSource={entries}
          onRow={(row) => ({
            onClick: () => {
              if (row.dir) {
                navigateTo(row.path);
              } else {
                void openFile(row.path);
              }
            },
            className: 'game-server-files-row',
          })}
          columns={[
            {
              title: t('gameServerDetail.fileName'),
              key: 'name',
              render: (_, row) => (
                <Space>
                  {row.dir ? <FolderOutlined /> : <FileOutlined />}
                  <span
                    className={
                      !row.dir && isHighlighted(row.name, highlightExtensions)
                        ? 'game-server-files-highlight'
                        : undefined
                    }
                  >
                    {row.name}
                  </span>
                </Space>
              ),
            },
            {
              title: t('gameServerDetail.fileSize'),
              key: 'size',
              render: (_, row) => (row.dir ? '—' : formatFileSize(row.size)),
            },
            {
              title: '',
              key: 'actions',
              width: 56,
              onCell: () => ({
                onClick: (event: MouseEvent) => event.stopPropagation(),
              }),
              render: (_, row) => (
                <Popconfirm
                  title={row.dir ? t('gameServerDetail.folderDeleteTitle') : t('gameServerDetail.fileDeleteTitle')}
                  description={
                    row.dir
                      ? t('gameServerDetail.folderDeleteConfirm', { name: row.name })
                      : t('gameServerDetail.fileDeleteConfirm', { name: row.name })
                  }
                  okText={t('common.delete')}
                  cancelText={t('common.cancel')}
                  okButtonProps={{ danger: true }}
                  onConfirm={() => void handleDelete(row)}
                >
                  <Button
                    type="text"
                    danger
                    size="small"
                    icon={<DeleteOutlined />}
                    loading={deletingPath === row.path}
                    aria-label={t('common.delete')}
                    onClick={(event) => event.stopPropagation()}
                  />
                </Popconfirm>
              ),
            },
          ]}
        />
      )}
    </div>
  );
}
