import { vi } from 'vitest';

export function createMockSkinViewer() {
  return {
    disposed: false,
    background: null,
    autoRotate: false,
    controls: { enableZoom: false, enablePan: false, enableRotate: true },
    loadSkin: vi.fn().mockResolvedValue(undefined),
    loadCape: vi.fn().mockResolvedValue(undefined),
    resetCameraPose: vi.fn(),
    dispose: vi.fn(),
  };
}

export const skinview3dMock = {
  SkinViewer: vi.fn(function MockSkinViewer() {
    return createMockSkinViewer();
  }),
};

/** Simulates headless CI where WebGL context creation fails synchronously. */
export const skinview3dWebGlFailureMock = {
  SkinViewer: vi.fn(function FailingSkinViewer() {
    throw new Error('Error creating WebGL context.');
  }),
};
