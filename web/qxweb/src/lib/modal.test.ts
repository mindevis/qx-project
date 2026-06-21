import { describe, expect, it } from 'vitest';
import { getModalMotionProps, modalMotionProps } from './modal';

describe('getModalMotionProps', () => {
  it('disables motion in test mode', () => {
    expect(getModalMotionProps('test')).toEqual({
      transitionName: '',
      maskTransitionName: '',
    });
  });

  it('keeps default motion in other modes', () => {
    expect(getModalMotionProps('development')).toEqual({});
    expect(getModalMotionProps('production')).toEqual({});
  });
});

describe('modalMotionProps', () => {
  it('uses current env mode', () => {
    expect(import.meta.env.MODE).toBe('test');
    expect(modalMotionProps).toEqual(getModalMotionProps('test'));
  });
});
