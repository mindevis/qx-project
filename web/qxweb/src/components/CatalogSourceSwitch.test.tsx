import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import type { ModCatalogItem } from '@/api/client';
import { renderWithTheme } from '@/test/test-utils';
import { CatalogSourceLinks, CatalogSourceSwitch } from './CatalogSourceSwitch';

const jeiMr: ModCatalogItem = {
  id: 'jei',
  source: 'modrinth',
  slug: 'jei',
  name: 'JEI',
  project_type: 'mod',
  external_url: 'https://modrinth.com/mod/jei',
};

const jeiCf: ModCatalogItem = {
  id: '238222',
  source: 'curseforge',
  slug: 'jei',
  name: 'JEI',
  project_type: 'mod',
  external_url: 'https://www.curseforge.com/minecraft/mc-mods/jei',
};

describe('CatalogSourceSwitch', () => {
  it('shows a source picker when both providers are present', () => {
    renderWithTheme(<CatalogSourceSwitch items={[jeiMr, jeiCf]} value="modrinth" onChange={() => undefined} />);
    expect(screen.getByRole('radio', { name: 'Modrinth' })).toBeInTheDocument();
    expect(screen.getByRole('radio', { name: 'CurseForge' })).toBeInTheDocument();
  });

  it('renders project links for each source', () => {
    renderWithTheme(<CatalogSourceLinks items={[jeiMr, jeiCf]} />);
    expect(screen.getByRole('link', { name: 'Открыть на Modrinth' })).toHaveAttribute(
      'href',
      'https://modrinth.com/mod/jei',
    );
    expect(screen.getByRole('link', { name: 'Открыть на CurseForge' })).toHaveAttribute(
      'href',
      'https://www.curseforge.com/minecraft/mc-mods/jei',
    );
  });
});
