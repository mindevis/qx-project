import { Button } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import { useMessage } from '@/hooks/useMessage';
import { useI18n } from '@/i18n/I18nContext';

type LauncherDownloadButtonProps = {
  type?: 'primary' | 'default';
};

export function LauncherDownloadButton({ type = 'default' }: LauncherDownloadButtonProps) {
  const message = useMessage();
  const { t } = useI18n();

  return (
    <Button
      icon={<DownloadOutlined />}
      type={type}
      onClick={() => {
        const url = import.meta.env.VITE_LAUNCHER_DOWNLOAD_URL?.trim();
        if (url) {
          window.open(url, '_blank', 'noopener,noreferrer');
          return;
        }
        message.info(t('home.downloadHint'), 6);
      }}
    >
      {t('home.downloadLauncher')}
    </Button>
  );
}
