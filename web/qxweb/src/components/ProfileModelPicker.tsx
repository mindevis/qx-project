import type { ProfileModel } from '@/api/client';
import { useI18n } from '@/i18n/I18nContext';
import { ProfileModelAvatar } from '@/components/ProfileModelAvatar';
import { ProfileModelViewer } from '@/components/ProfileModelViewer';
import './ProfileModelPicker.css';

const OPTIONS: ProfileModel[] = ['steve', 'alex'];

type ProfileModelPickerProps = {
  value?: ProfileModel;
  onChange?: (value: ProfileModel) => void;
};

export function ProfileModelPicker({ value = 'steve', onChange }: ProfileModelPickerProps) {
  const { t } = useI18n();

  return (
    <div className="profile-model-picker" role="radiogroup" aria-label={t('launcher.profileModelLabel')}>
      <p className="profile-model-hint">{t('launcher.profileModelRotateHint')}</p>
      {OPTIONS.map((model) => {
        const selected = value === model;
        const label = t(`launcher.profileModel.${model}`);
        const gender = t(`launcher.profileGender.${model}`);
        return (
          <div
            key={model}
            className={`profile-model-option${selected ? ' profile-model-option--selected' : ''}`}
          >
            <div
              className="profile-model-preview"
              onPointerDown={(e) => e.stopPropagation()}
              onClick={(e) => e.stopPropagation()}
            >
              <ProfileModelViewer model={model} />
            </div>
            <button
              type="button"
              role="radio"
              aria-checked={selected}
              className="profile-model-select"
              onClick={() => onChange?.(model)}
              aria-label={`${label} — ${gender}`}
            >
              <span className="profile-model-label">{label}</span>
              <span className="profile-model-caption">{gender}</span>
            </button>
          </div>
        );
      })}
    </div>
  );
}

export { ProfileModelAvatar };
