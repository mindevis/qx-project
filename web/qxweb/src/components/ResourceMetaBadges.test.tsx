import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ResourceMetaBadges } from '@/components/ResourceMetaBadges';
import type { InstanceResource } from '@/api/client';

const item: InstanceResource = {
  source: 'modrinth',
  project_id: 'journeymap',
  project_name: 'JourneyMap',
  version_number: 'journeymap-1.20.1-5.10.3-forge',
  filename: 'journeymap-1.20.1-5.10.3-forge.jar',
  file_size: 7_127_424,
  downloads: 338_700_000,
  resource_type: 'mod',
  installed_at: '2026-01-01T00:00:00Z',
};

const t = (key: string) => {
  if (key === 'qxmods.tabs.mod') return 'Моды';
  if (key === 'qxmods.installed.downloads') return 'скачиваний';
  return key;
};

describe('ResourceMetaBadges', () => {
  it('renders metadata as separate badges', () => {
    render(<ResourceMetaBadges item={item} t={t} />);
    expect(screen.getByText('Моды')).toBeInTheDocument();
    expect(screen.getByText('journeymap-1.20.1-5.10.3-forge')).toBeInTheDocument();
    expect(screen.getByText('journeymap-1.20.1-5.10.3-forge.jar')).toBeInTheDocument();
    expect(screen.getByText('6.8 MB')).toBeInTheDocument();
    expect(screen.getByText(/338\.7M скачиваний/)).toBeInTheDocument();
  });
});
