import "@testing-library/jest-dom/vitest"
import { cleanup } from "@testing-library/react"
import { afterEach } from "vitest"

afterEach(cleanup)

class ResizeObserverStub {
    observe() {}
    unobserve() {}
    disconnect() {}
}

if (!("ResizeObserver" in globalThis)) {
    globalThis.ResizeObserver = ResizeObserverStub
}

Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: (query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
        addEventListener: () => {},
        removeEventListener: () => {},
        dispatchEvent: () => false,
    }),
})
