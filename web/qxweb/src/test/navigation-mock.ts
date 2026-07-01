import { vi } from 'vitest';

const BASE_ORIGIN = 'http://localhost:3000';
let hrefValue = `${BASE_ORIGIN}/`;

function resolveHref(url: string | URL): string {
  return new URL(String(url), hrefValue).href;
}

function locationUrl(): URL {
  return new URL(hrefValue);
}

export const testNavigation = {
  assign: vi.fn((url: string | URL) => {
    hrefValue = resolveHref(url);
  }),
  replace: vi.fn((url: string | URL) => {
    hrefValue = resolveHref(url);
  }),
  hrefSet: vi.fn((url: string) => {
    hrefValue = resolveHref(url);
  }),
  getHref: () => hrefValue,
  reset() {
    hrefValue = `${BASE_ORIGIN}/`;
    this.assign.mockClear();
    this.replace.mockClear();
    this.hrefSet.mockClear();
  },
};

let anchorClickSpy: ReturnType<typeof vi.spyOn> | undefined;
let windowOpenSpy: ReturnType<typeof vi.spyOn> | undefined;
let pushStateSpy: ReturnType<typeof vi.spyOn> | undefined;
let replaceStateSpy: ReturnType<typeof vi.spyOn> | undefined;

function syncHrefFromHistoryUrl(url: string | URL | null | undefined) {
  if (url == null || url === '') return;
  hrefValue = resolveHref(url);
}

/** Prevents jsdom "Not implemented: navigation to another Document" in tests. */
export function installNavigationMock() {
  testNavigation.reset();

  const originalPushState = window.history.pushState.bind(window.history);
  const originalReplaceState = window.history.replaceState.bind(window.history);

  if (!pushStateSpy) {
    pushStateSpy = vi.spyOn(window.history, 'pushState').mockImplementation((state, title, url) => {
      syncHrefFromHistoryUrl(url);
      return originalPushState(state, title, url);
    });
  }
  if (!replaceStateSpy) {
    replaceStateSpy = vi.spyOn(window.history, 'replaceState').mockImplementation((state, title, url) => {
      syncHrefFromHistoryUrl(url);
      return originalReplaceState(state, title, url);
    });
  }

  Object.defineProperty(window, 'location', {
    configurable: true,
    value: {
      assign: testNavigation.assign,
      replace: testNavigation.replace,
      get href() {
        return hrefValue;
      },
      set href(value: string) {
        testNavigation.hrefSet(value);
      },
      get origin() {
        return locationUrl().origin;
      },
      get pathname() {
        return locationUrl().pathname;
      },
      get search() {
        return locationUrl().search;
      },
      get hash() {
        return locationUrl().hash;
      },
      toString() {
        return hrefValue;
      },
    },
  });

  if (!anchorClickSpy) {
    anchorClickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
  }
  if (!windowOpenSpy) {
    windowOpenSpy = vi.spyOn(window, 'open').mockImplementation(() => null);
  }
}
