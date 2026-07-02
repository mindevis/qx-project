import { describe, expect, it } from 'vitest';
import { defaultRequiredSelection } from './ModInstallDepsModal';

describe('defaultRequiredSelection', () => {
  it('selects uninstalled required dependencies by default', () => {
    const selected = defaultRequiredSelection(
      [
        {
          source: 'curseforge',
          project_id: 'req-1',
          dependency_type: 'required',
          version_id: 'v1',
          filename: 'a.jar',
          download_url: 'https://example/a.jar',
        },
        {
          source: 'curseforge',
          project_id: 'req-2',
          dependency_type: 'required',
          version_id: 'v2',
          filename: 'b.jar',
          download_url: 'https://example/b.jar',
        },
        {
          source: 'curseforge',
          project_id: 'opt-1',
          dependency_type: 'optional',
          version_id: 'v3',
          filename: 'c.jar',
          download_url: 'https://example/c.jar',
        },
      ],
      new Set(['curseforge:req-2']),
    );

    expect(selected).toEqual(new Set(['req-1']));
  });
});
