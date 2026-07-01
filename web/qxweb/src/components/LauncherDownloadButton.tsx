import { Button } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import { useI18n } from '@/i18n/I18nContext';
import { openLauncherDownload, resolveLauncherDownloadUrl, type LauncherRelease } from '@/lib/launcherDownload';

type LauncherDownloadButtonProps = {
  type?: 'primary' | 'default';
  size?: 'small' | 'middle' | 'large';
  release?: LauncherRelease | null;
};

export function LauncherDownloadButton({
  type = 'default',
  size,
  release = null,
}: LauncherDownloadButtonProps) {
  const { t } = useI18n();

  return (
    <Button
      icon={<DownloadOutlined />}
      type={type}
      size={size}
      onClick={() => {
        const url = resolveLauncherDownloadUrl(release);
        openLauncherDownload(url);
      }}
    >
      {t('home.downloadLauncher')}
    </Button>
  );
}
