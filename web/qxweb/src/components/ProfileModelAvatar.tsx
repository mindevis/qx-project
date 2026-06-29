import type { ProfileModel } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';

type ProfileModelAvatarProps = {
  model?: ProfileModel;
  size?: 'sm' | 'md';
};

const BODY_SRC: Record<ProfileModel, string> = {
  steve: '/profiles/steve-body.png',
  alex: '/profiles/alex-body.png',
};

export function ProfileModelAvatar({ model = 'steve', size = 'md' }: ProfileModelAvatarProps) {
  const { t } = useI18n();
  const resolvedModel = model === 'alex' ? 'alex' : 'steve';

  return (
    <img
      src={BODY_SRC[resolvedModel]}
      alt={t(`seo.profileAvatarAlt.${resolvedModel}`)}
      className={`profile-model-avatar profile-model-avatar--${size}`}
      draggable={false}
      loading="lazy"
    />
  );
}
