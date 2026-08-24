import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { ResourceMetaBadges } from '@/components/ResourceMetaBadges';
import type { InstanceResource } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';

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

describe('ResourceMetaBadges', () => {
  it('renders the version and channel badges for installed resources', () => {
    renderWithTheme(<ResourceMetaBadges item={item} />);
    expect(screen.getByText('journeymap-1.20.1-5.10.3-forge')).toBeInTheDocument();
    expect(screen.getByText('Stable')).toBeInTheDocument();
    expect(screen.queryByText('Моды')).not.toBeInTheDocument();
    expect(screen.queryByText('journeymap-1.20.1-5.10.3-forge.jar')).not.toBeInTheDocument();
    expect(screen.queryByText('6.8 MB')).not.toBeInTheDocument();
  });
});
