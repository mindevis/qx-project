import { useEffect, useState } from 'react';
import type { ProfileModel } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';

type ProfileModelAvatarProps = {
  model?: ProfileModel;
  size?: 'sm' | 'md';
  src?: string;
  alt?: string;
};

const BODY_SRC: Record<ProfileModel, string> = {
  steve: '/profiles/steve-body.png',
  alex: '/profiles/alex-body.png',
};

export function ProfileModelAvatar({
  model = 'steve',
  size = 'md',
  src,
  alt,
}: ProfileModelAvatarProps) {
  const { t } = useI18n();
  const resolvedModel = model === 'alex' ? 'alex' : 'steve';
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    setFailed(false);
  }, [src]);

  const resolvedSrc = src && !failed ? src : BODY_SRC[resolvedModel];
  const resolvedAlt = alt ?? t(`seo.profileAvatarAlt.${resolvedModel}`);

  return (
    <img
      src={resolvedSrc}
      alt={resolvedAlt}
      className={`profile-model-avatar profile-model-avatar--${size}`}
      draggable={false}
      loading="lazy"
      onError={() => {
        if (src && !failed) {
          setFailed(true);
        }
      }}
    />
  );
}
