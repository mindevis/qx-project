/** Ant Design modals hang in jsdom 29 leave transitions; disable motion in Vitest only. */
export function getModalMotionProps(mode: string) {
  return mode === 'test'
    ? ({ transitionName: '', maskTransitionName: '' } as const)
    : ({} as const);
}

export const modalMotionProps = getModalMotionProps(import.meta.env.MODE);
