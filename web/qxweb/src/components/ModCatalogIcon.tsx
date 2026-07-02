import { useState } from 'react';

type ModCatalogIconProps = {
  url?: string;
  name: string;
  size?: number;
  className?: string;
};

export function ModCatalogIcon({ url, name, size = 40, className }: ModCatalogIconProps) {
  const [failed, setFailed] = useState(false);
  const classNames = ['qxmods-catalog-icon', className].filter(Boolean).join(' ');

  if (!url?.trim() || failed) {
    return (
      <span
        className={`${classNames} qxmods-catalog-icon--placeholder`}
        style={{ width: size, height: size }}
        role="img"
        aria-label={name}
      />
    );
  }

  return (
    <img
      src={url}
      alt=""
      width={size}
      height={size}
      className={classNames}
      loading="lazy"
      decoding="async"
      referrerPolicy="no-referrer"
      onError={() => setFailed(true)}
    />
  );
}
