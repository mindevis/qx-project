import { Typography } from 'antd';

type Props = {
  title: string;
  phase: string;
};

export function PlaceholderPage({ title, phase }: Props) {
  return (
    <div style={{ maxWidth: 560 }}>
      <Typography.Title level={3}>{title}</Typography.Title>
      <Typography.Paragraph type="secondary">
        Раздел будет реализован в {phase}. См. docs/mvp.md.
      </Typography.Paragraph>
    </div>
  );
}
