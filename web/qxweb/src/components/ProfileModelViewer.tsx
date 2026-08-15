import { useEffect, useRef, useState } from 'react';
import { SkinViewer } from 'skinview3d';
import type { ProfileModel } from '@/api/client';
import { ProfileModelAvatar } from '@/components/ProfileModelAvatar';
import { logger } from '@/lib/logger';

const SKIN_SRC: Record<ProfileModel, string> = {
  steve: '/profiles/steve-skin.png',
  alex: '/profiles/alex-skin.png',
};

const MODEL_TYPE = {
  steve: 'default',
  alex: 'slim',
} as const;

type ProfileModelViewerProps = {
  model: ProfileModel;
  width?: number;
  height?: number;
};

export function ProfileModelViewer({ model, width = 112, height = 128 }: ProfileModelViewerProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const viewerRef = useRef<SkinViewer | null>(null);
  const [useFallback, setUseFallback] = useState(false);

  useEffect(() => {
    if (useFallback) {
      return;
    }

    const canvas = canvasRef.current;
    /* v8 ignore next 3 -- @preserve canvas ref is unset only before mount */
    if (!canvas) {
      return;
    }

    let viewer: SkinViewer;
    try {
      viewer = new SkinViewer({
        canvas,
        width,
        height,
        skin: SKIN_SRC[model],
        model: MODEL_TYPE[model],
      });
    } catch (error) {
      logger.warn('profile model viewer unavailable, using static preview', {
        error: String(error),
      });
      setUseFallback(true);
      return;
    }

    viewer.background = null;
    viewer.autoRotate = false;
    viewer.controls.enableZoom = false;
    viewer.controls.enablePan = false;
    viewer.controls.enableRotate = true;
    viewer.resetCameraPose();

    viewerRef.current = viewer;

    return () => {
      viewer.dispose();
      viewerRef.current = null;
    };
  }, [height, model, useFallback, width]);

  if (useFallback) {
    return <ProfileModelAvatar model={model} size="md" />;
  }

  return (
    <canvas
      ref={canvasRef}
      className="profile-model-canvas"
      width={width}
      height={height}
      aria-hidden
    />
  );
}
