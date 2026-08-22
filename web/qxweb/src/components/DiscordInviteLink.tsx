import { DiscordOutlined } from '@ant-design/icons';
import { useI18n } from '@/i18n/I18nContext';
import { DISCORD_INVITE_URL } from '@/lib/community';
import './DiscordInviteLink.css';

type DiscordInviteLinkProps = {
  size?: 'sm' | 'md';
  className?: string;
};

export function DiscordInviteLink({ size = 'md', className }: DiscordInviteLinkProps) {
  const { t } = useI18n();
  const classes = ['discord-invite', `discord-invite--${size}`, className]
    .filter(Boolean)
    .join(' ');

  return (
    <a
      className={classes}
      href={DISCORD_INVITE_URL}
      target="_blank"
      rel="noopener noreferrer"
      aria-label={t('layout.discordAria')}
    >
      <span className="discord-invite-icon" aria-hidden>
        <DiscordOutlined />
      </span>
      <span className="discord-invite-label">{t('layout.discord')}</span>
    </a>
  );
}
