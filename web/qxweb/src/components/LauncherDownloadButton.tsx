import { Button } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import { useMessage } from '@/hooks/useMessage';

type LauncherDownloadButtonProps = {
  type?: 'primary' | 'default';
};

export function LauncherDownloadButton({ type = 'default' }: LauncherDownloadButtonProps) {
  const message = useMessage();
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
        message.info(
          'Сборка из исходников: make build-launcher (см. README). URL релиза — launcher_download_url в web.toml.',
          6,
        );
      }}
    >
      Скачать QXLauncher
    </Button>
  );
}
