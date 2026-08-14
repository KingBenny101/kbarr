import { render, screen, waitFor, fireEvent } from "@testing-library/react"
import { MantineProvider } from "@mantine/core"
import { MemoryRouter, Route, Routes } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { MonitorPage } from "./MonitorPage"
import { apiFetch } from "@/utils"

const mockedApiFetch = vi.mocked(apiFetch)

const LONG_TITLE = "This Is an Extremely Long Anime Series Title That Keeps Going Well Past the Column Width for Testing Purposes"
const LONG_EP = "Episode Title That Is Also Extremely Long and Descriptive About the Events That Happen During This Particular Episode of the Show"

const ITEMS = [
    { ID: 1, CreatedAt: "2026-08-01T00:00:00Z", library_id: 1, title: LONG_TITLE, episode_title: LONG_EP, season: 1, episode_number: 12, is_episode: true, is_season: false, status: "missing", quality: "1080p", subtitles: "English, Español" },
]

vi.mock("@/utils", async (importOriginal) => {
    const actual = await importOriginal<typeof import("@/utils")>()
    return { ...actual, apiFetch: vi.fn() }
})

function renderPage() {
    return render(
        <MantineProvider>
            <MemoryRouter initialEntries={["/monitored"]}>
                <Routes>
                    <Route path="/monitored" element={<MonitorPage />} />
                </Routes>
            </MemoryRouter>
        </MantineProvider>,
    )
}

function jsonResponse(data: unknown): Response {
    return new Response(JSON.stringify(data), {
        status: 200,
        headers: { "Content-Type": "application/json" },
    })
}

beforeEach(() => {
    vi.clearAllMocks()
    // Default mock for initial GET /api/monitor
    mockedApiFetch.mockResolvedValueOnce(jsonResponse(ITEMS))
})

afterEach(() => {
    vi.unstubAllGlobals()
})

describe("MonitorPage", () => {
    it("renders long titles as a block so truncation applies", async () => {
        renderPage()

        const title = await screen.findByText(LONG_TITLE)
        expect(title).toHaveStyle({ display: "block" })
    })

    it("fixes the table width so percentage columns resolve to real proportions", async () => {
        renderPage()

        await screen.findByText(LONG_TITLE)
        const table = document.querySelector("table")
        expect(table).toHaveStyle({ tableLayout: "fixed", width: "100%" })
    })

    it("shows Search button in each row and calls API on click", async () => {
        // Initial GET is mocked in beforeEach, now mock the POST for search
        mockedApiFetch.mockResolvedValueOnce(new Response(null, { status: 204 }))

        renderPage()

        // Wait for table to load - wait for the row to exist
        await waitFor(() => expect(screen.getByRole("row", { name: /missing/i })).toBeInTheDocument())

        // Find and click Search button for first row
        const searchButtons = screen.getAllByRole("button", { name: /search/i })
        expect(searchButtons).toHaveLength(1) // one per visible row

        fireEvent.click(searchButtons[0])

        // Verify API called
        await waitFor(() => expect(mockedApiFetch).toHaveBeenCalledWith(
            expect.stringContaining("/api/monitor/"),
            expect.objectContaining({ method: "POST" })
        ))
    })

    it("shows Search button on mobile card and calls API on click", async () => {
        // Simulate mobile viewport
        const original = window.matchMedia
        Object.defineProperty(window, "matchMedia", {
            writable: true,
            value: (query: string) => ({
                matches: query.includes("max-width: 768px"),
                media: query,
                onchange: null,
                addListener: () => {},
                removeListener: () => {},
                addEventListener: () => {},
                removeEventListener: () => {},
                dispatchEvent: () => false,
            }),
        })

        // Initial GET is mocked in beforeEach, now mock the POST for search
        mockedApiFetch.mockResolvedValueOnce(new Response(null, { status: 204 }))

        renderPage()

        // Wait for card to load
        await waitFor(() => expect(screen.getByText(LONG_TITLE)).toBeInTheDocument())

        // Find and click Search button in mobile card
        const searchButton = screen.getByRole("button", { name: "Search now" })
        fireEvent.click(searchButton)

        // Verify API called
        await waitFor(() => expect(mockedApiFetch).toHaveBeenCalledWith(
            expect.stringContaining("/api/monitor/"),
            expect.objectContaining({ method: "POST" })
        ))

        // Restore matchMedia
        Object.defineProperty(window, "matchMedia", { writable: true, value: original })
    })
})
