import type { ProfileModel } from '@/api/client';

type ProfileModelAvatarProps = {
  model?: ProfileModel;
  size?: 'sm' | 'md';
};

const BODY_SRC: Record<ProfileModel, string> = {
  steve: '/profiles/steve-body.png',
  alex: '/profiles/alex-body.png',
};

export function ProfileModelAvatar({ model = 'steve', size = 'md' }: ProfileModelAvatarProps) {
  return (
    <img
      src={BODY_SRC[model === 'alex' ? 'alex' : 'steve']}
      alt=""
      className={`profile-model-avatar profile-model-avatar--${size}`}
      draggable={false}
      loading="lazy"
    />
  );
}
