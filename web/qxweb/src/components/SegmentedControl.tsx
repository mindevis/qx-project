import type { ReactNode } from 'react';
import './SegmentedControl.css';

export type SegmentOption<T extends string> = {
  value: T;
  label: ReactNode;
  ariaLabel: string;
};

type SegmentedControlProps<T extends string> = {
  value: T;
  options: SegmentOption<T>[];
  onChange: (value: T) => void;
  groupLabel: string;
  className?: string;
  iconOnly?: boolean;
};

export function SegmentedControl<T extends string>({
  value,
  options,
  onChange,
  groupLabel,
  className,
  iconOnly = false,
}: SegmentedControlProps<T>) {
  return (
    <div
      className={['qx-segment', iconOnly && 'qx-segment--icon', className].filter(Boolean).join(' ')}
      role="radiogroup"
      aria-label={groupLabel}
    >
      {options.map((item) => {
        const active = value === item.value;

        return (
          <button
            key={item.value}
            type="button"
            role="radio"
            aria-checked={active}
            aria-label={item.ariaLabel}
            title={item.ariaLabel}
            className={[
              'qx-segment__btn',
              iconOnly && 'qx-segment__btn--icon',
              active && 'qx-segment__btn--active',
            ]
              .filter(Boolean)
              .join(' ')}
            onClick={() => onChange(item.value)}
          >
            {item.label}
          </button>
        );
      })}
    </div>
  );
}
