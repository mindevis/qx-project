export function formatCompactCount(value: number): string {
  const abs = Math.abs(value);
  const sign = value < 0 ? '-' : '';
  if (abs < 1000) {
    return `${sign}${abs}`;
  }
  const units: Array<{ size: number; suffix: string }> = [
    { size: 1_000_000_000, suffix: 'B' },
    { size: 1_000_000, suffix: 'M' },
    { size: 1_000, suffix: 'K' },
  ];
  for (const unit of units) {
    if (abs >= unit.size) {
      const scaled = abs / unit.size;
      const digits = scaled >= 10 ? 0 : 1;
      const rounded = scaled.toFixed(digits).replace(/\.0$/, '');
      return `${sign}${rounded}${unit.suffix}`;
    }
  }
  return `${sign}${abs}`;
}
