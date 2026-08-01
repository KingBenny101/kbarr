import { render, screen } from "@testing-library/react"
import { MantineProvider } from "@mantine/core"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import SystemPage from "./SystemPage"
import { ringProgress } from "@/lib/format"

function renderPage() {
    return render(
        <MantineProvider>
            <SystemPage />
        </MantineProvider>,
    )
}

function jsonResponse(data: unknown): Response {
    return new Response(JSON.stringify(data), {
        status: 200,
        headers: { "Content-Type": "application/json" },
    })
}

const CYCLES = [
    {
        service: "core",
        cycle: "availability",
        display_name: "Availability check",
        state: "idle",
        last_started_at: "2026-08-01T11:59:00Z",
        last_finished_at: "2026-08-01T11:59:03Z",
        last_duration_ms: 3000,
        next_run_at: "2026-08-01T12:00:03Z",
    },
    {
        service: "core",
        cycle: "metadata_refresh",
        display_name: "Metadata refresh",
        state: "running",
        last_started_at: "2026-08-01T11:58:00Z",
        last_finished_at: null,
        last_duration_ms: 0,
        next_run_at: null,
    },
]

const WORKERS = [
    { name: "core", display_name: "Core", running: true },
    { name: "metadata", display_name: "Metadata", running: false },
    { name: "indexer", display_name: "Indexer", running: true },
    { name: "downloader", display_name: "Downloader", running: true },
]

beforeEach(() => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input)
        if (url.includes("/api/cycles")) return jsonResponse({ cycles: CYCLES })
        if (url.includes("/api/workers")) return jsonResponse(WORKERS)
        return jsonResponse({})
    })
    vi.stubGlobal("fetch", fetchMock)
})

afterEach(() => {
    vi.unstubAllGlobals()
})

describe("SystemPage", () => {
    it("renders a row per cycle with times", async () => {
        renderPage()

        expect(await screen.findByText("Availability check")).toBeInTheDocument()
        expect(screen.getByText("Metadata refresh")).toBeInTheDocument()
    })

    it("renders a red ring for cycles of offline services", async () => {
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            const url = String(input)
            if (url.includes("/api/cycles")) {
                return jsonResponse({
                    cycles: [
                        {
                            ...CYCLES[0],
                            service: "metadata",
                            cycle: "anidb_sync",
                            display_name: "AniDB title sync",
                        },
                    ],
                })
            }
            if (url.includes("/api/workers")) return jsonResponse(WORKERS)
            return jsonResponse({})
        })
        vi.stubGlobal("fetch", fetchMock)

        renderPage()

        const card = (await screen.findByText("AniDB title sync")).closest("div")!
        expect(card.querySelector('circle[stroke="var(--mantine-color-red-6)"]')).not.toBeNull()
    })

    it("renders stacked cards with labeled values on mobile", async () => {
        renderPage()

        expect(await screen.findByText("Availability check")).toBeInTheDocument()
        expect(screen.getAllByText("Last run")).toHaveLength(2)
        expect(screen.getAllByText("Next run")).toHaveLength(2)
        expect(screen.getAllByText("Duration")).toHaveLength(2)
        expect(screen.queryByText("Cycle")).not.toBeInTheDocument()
    })

    it("renders the table layout on desktop", async () => {
        const original = window.matchMedia
        Object.defineProperty(window, "matchMedia", {
            writable: true,
            value: (query: string) => ({
                matches: query.includes("min-width: 62em"),
                media: query,
                onchange: null,
                addListener: () => {},
                removeListener: () => {},
                addEventListener: () => {},
                removeEventListener: () => {},
                dispatchEvent: () => false,
            }),
        })

        try {
            renderPage()

            expect(await screen.findByText("Availability check")).toBeInTheDocument()
            expect(screen.getByText("Cycle")).toBeInTheDocument()
            expect(screen.getAllByText("Duration")).toHaveLength(1)
        } finally {
            Object.defineProperty(window, "matchMedia", { writable: true, value: original })
        }
    })

    it("renders the empty state when no cycles are recorded", async () => {
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            const url = String(input)
            if (url.includes("/api/cycles")) return jsonResponse({ cycles: [] })
            if (url.includes("/api/workers")) return jsonResponse(WORKERS)
            return jsonResponse({})
        })
        vi.stubGlobal("fetch", fetchMock)

        renderPage()

        expect(await screen.findByText("No cycles recorded yet")).toBeInTheDocument()
    })
})

describe("ringProgress", () => {
    const finished = "2026-08-01T12:00:00Z"
    const next = "2026-08-01T12:10:00Z"

    it("returns null when the cycle has never finished or has no next run", () => {
        expect(ringProgress(new Date(), null, next)).toBeNull()
        expect(ringProgress(new Date(), finished, null)).toBeNull()
        expect(ringProgress(new Date(), null, null)).toBeNull()
    })

    it("returns null when next run is not after the last finish", () => {
        expect(ringProgress(new Date(), next, finished)).toBeNull()
    })

    it("returns the elapsed fraction toward the next run", () => {
        const now = new Date("2026-08-01T12:05:00Z")
        expect(ringProgress(now, finished, next)).toBeCloseTo(0.5, 5)
    })

    it("clamps outside the window", () => {
        expect(ringProgress(new Date("2026-08-01T11:55:00Z"), finished, next)).toBe(0)
        expect(ringProgress(new Date("2026-08-01T12:15:00Z"), finished, next)).toBe(1)
    })
})
