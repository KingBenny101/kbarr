import { render, screen } from "@testing-library/react"
import { MantineProvider } from "@mantine/core"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import SystemPage from "./SystemPage"

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
    it("renders a row per cycle with state and times", async () => {
        renderPage()

        expect(await screen.findByText("Availability check")).toBeInTheDocument()
        expect(screen.getByText("Metadata refresh")).toBeInTheDocument()

        expect(screen.getByText("Idle")).toBeInTheDocument()
        expect(screen.getByText("Running now")).toBeInTheDocument()
        // Offline: availability row is core (healthy) — the offline pill comes
        // from a cycle whose service is missing from the healthy workers.
    })

    it("marks cycles of offline services as Offline", async () => {
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

        expect(await screen.findByText("AniDB title sync")).toBeInTheDocument()
        expect(screen.getByText("Offline")).toBeInTheDocument()
    })
})
