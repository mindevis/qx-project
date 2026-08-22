import { useCallback, useEffect, useState } from 'react';
import { AutoComplete, Button, Empty, Popconfirm, Space, Spin, Tag, Typography } from 'antd';
import {
  CloudDownloadOutlined,
  DeleteOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
} from '@ant-design/icons';
import { api, type OllamaStatus, type OllamaView } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { useMessage } from '@/hooks/useMessage';
import { logger } from '@/lib/logger';

const { Paragraph, Text, Title } = Typography;

const suggestedOllamaModels = [
  'qwen2.5:1.5b',
  'qwen2.5:3b',
  'qwen2.5',
  'llama3.2',
  'llama3.2:1b',
  'llama3.1',
  'gemma3',
  'gemma3:1b',
  'mistral',
  'phi4',
];

function ollamaStatusColor(status: OllamaStatus): string {
  switch (status) {
    case 'running':
      return 'success';
    case 'installing':
    case 'starting':
    case 'pulling':
      return 'processing';
    case 'error':
      return 'error';
    case 'installed':
      return 'blue';
    default:
      return 'default';
  }
}

function formatModelSize(size?: number): string {
  if (!size || size <= 0) {
    return '—';
  }
  const gb = size / 1024 / 1024 / 1024;
  if (gb >= 1) {
    return `${gb.toFixed(1)} GB`;
  }
  const mb = size / 1024 / 1024;
  return `${Math.max(1, Math.round(mb))} MB`;
}

const emptyOllama: OllamaView = { status: 'not_installed', models: [] };

export function OllamaPanel({ vpsId, agentOnline }: { vpsId: string; agentOnline: boolean }) {
  const { t } = useI18n();
  const message = useMessage();
  const [view, setView] = useState<OllamaView>(emptyOllama);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [modelName, setModelName] = useState('');

  const refresh = useCallback(async () => {
    try {
      const next = await api.getVpsOllama(vpsId);
      setView(next);
    } catch (e) {
      logger.warn('failed to load ollama', { error: String(e) });
      message.error(t('servers.ollamaLoadFailed'));
    } finally {
      setLoading(false);
    }
  }, [message, t, vpsId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    if (!agentOnline) return undefined;
    const busyStatus =
      view.status === 'installing' ||
      view.status === 'starting' ||
      view.status === 'stopping' ||
      view.status === 'pulling';
    if (!busyStatus) return undefined;
    const timer = window.setInterval(() => void refresh(), 3000);
    return () => window.clearInterval(timer);
  }, [agentOnline, refresh, view.status]);

  const run = async (action: () => Promise<OllamaView>, successKey: string) => {
    setBusy(true);
    try {
      const next = await action();
      setView(next);
      message.success(t(successKey));
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('servers.ollamaActionFailed'));
    } finally {
      setBusy(false);
    }
  };

  const installed = view.status !== 'not_installed';
  const running = view.status === 'running' || view.status === 'pulling';
  const provisioning =
    view.status === 'installing' || view.status === 'starting' || view.status === 'stopping';

  return (
    <div className="servers-panel">
      <div className="servers-panel-header">
        <Title level={4} className="servers-panel-title">
          {t('servers.ollamaTitle')}
        </Title>
        <Space wrap>
          {installed ? (
            <>
              <Tag color={ollamaStatusColor(view.status)}>{t(`servers.ollamaStatus.${view.status}`)}</Tag>
              <Button
                icon={running ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
                loading={busy || provisioning}
                disabled={!agentOnline || busy || provisioning}
                onClick={() =>
                  void run(
                    () => (running ? api.stopVpsOllama(vpsId) : api.startVpsOllama(vpsId)),
                    running ? 'servers.ollamaStopped' : 'servers.ollamaStarted',
                  )
                }
              >
                {running ? t('servers.ollamaStop') : t('servers.ollamaStart')}
              </Button>
            </>
          ) : (
            <Button
              type="primary"
              icon={<CloudDownloadOutlined />}
              loading={busy || view.status === 'installing'}
              disabled={!agentOnline || busy}
              onClick={() => void run(() => api.installVpsOllama(vpsId), 'servers.ollamaInstallDone')}
            >
              {t('servers.ollamaInstall')}
            </Button>
          )}
        </Space>
      </div>
      <Paragraph type="secondary" className="servers-hint">
        {t('servers.ollamaHint')}
      </Paragraph>
      {!agentOnline ? (
        <Paragraph type="secondary" className="servers-hint">
          {t('servers.ollamaAgentRequired')}
        </Paragraph>
      ) : loading ? (
        <div className="servers-loading">
          <Spin />
        </div>
      ) : (
        <>
          {view.version || view.listen_addr ? (
            <dl className="servers-game-card-meta servers-game-card-meta--grid ollama-meta">
              {view.version ? (
                <div className="servers-game-card-meta-item">
                  <dt>{t('servers.ollamaVersion')}</dt>
                  <dd>{view.version}</dd>
                </div>
              ) : null}
              {view.listen_addr ? (
                <div className="servers-game-card-meta-item">
                  <dt>{t('servers.ollamaListen')}</dt>
                  <dd>{view.listen_addr}</dd>
                </div>
              ) : null}
            </dl>
          ) : null}
          {view.last_error ? (
            <Paragraph type="danger" className="servers-hint">
              {view.last_error}
            </Paragraph>
          ) : null}
          {view.status === 'pulling' && view.pulling_model ? (
            <Paragraph type="secondary" className="servers-hint">
              {t('servers.ollamaPulling', { name: view.pulling_model })}
            </Paragraph>
          ) : null}

          {running ? (
            <div className="ollama-models">
              <div className="ollama-models-header">
                <Text strong>{t('servers.ollamaModels')}</Text>
                <Space.Compact className="ollama-pull">
                  <AutoComplete
                    value={modelName}
                    options={suggestedOllamaModels.map((name) => ({ value: name }))}
                    placeholder={t('servers.ollamaModelPlaceholder')}
                    onChange={setModelName}
                    disabled={busy}
                    filterOption={(input, option) =>
                      String(option?.value ?? '')
                        .toLowerCase()
                        .includes(input.trim().toLowerCase())
                    }
                  />
                  <Button
                    type="primary"
                    icon={<CloudDownloadOutlined />}
                    loading={busy || view.status === 'pulling'}
                    disabled={!modelName.trim() || busy}
                    onClick={() =>
                      void run(
                        () => api.pullVpsOllamaModel(vpsId, modelName.trim()),
                        'servers.ollamaPulled',
                      )
                    }
                  >
                    {t('servers.ollamaPull')}
                  </Button>
                </Space.Compact>
              </div>
              {view.models.length === 0 ? (
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description={
                    <div className="servers-game-empty-copy">
                      <Text strong>{t('servers.ollamaModelsEmpty')}</Text>
                      <Paragraph type="secondary">{t('servers.ollamaModelsEmptyHint')}</Paragraph>
                    </div>
                  }
                />
              ) : (
                <ul className="ollama-model-list">
                  {view.models.map((model) => (
                    <li key={model.name} className="ollama-model-row">
                      <div>
                        <Text strong>{model.name}</Text>
                        <Text type="secondary" className="ollama-model-size">
                          {formatModelSize(model.size)}
                        </Text>
                      </div>
                      <Popconfirm
                        title={t('servers.ollamaDeleteModelConfirm', { name: model.name })}
                        onConfirm={() =>
                          void run(
                            () => api.deleteVpsOllamaModel(vpsId, model.name),
                            'servers.ollamaDeleted',
                          )
                        }
                      >
                        <Button
                          type="text"
                          size="small"
                          danger
                          icon={<DeleteOutlined />}
                          aria-label={t('servers.ollamaDeleteModel')}
                        />
                      </Popconfirm>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          ) : null}
        </>
      )}
    </div>
  );
}
