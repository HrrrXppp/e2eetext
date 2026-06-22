import "@testing-library/jest-dom/vitest";

if (!HTMLElement.prototype.scrollIntoView) {
  HTMLElement.prototype.scrollIntoView = () => {};
}

class MockIntersectionObserver {
  constructor(_callback: IntersectionObserverCallback) {}

  observe = () => {};

  disconnect = () => {};

  unobserve = () => {};
}

if (!globalThis.IntersectionObserver) {
  globalThis.IntersectionObserver =
    MockIntersectionObserver as unknown as typeof IntersectionObserver;
}
