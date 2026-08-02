import { render, screen } from "@testing-library/react"
import { MantineProvider } from "@mantine/core"
import { MemoryRouter } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { MonitorPage } from "./MonitorPage"

const LONG_TITLE = "This Is an Extremely Long Anime Series Title That Keeps Going Well Past the Column Width for Testing Purposes"
const LONG_EP = "Episode Title That Is Also Extremely Long and Descriptive About the Events That Happen During This Particular Episode of the Show"

const ITEMS = [
    { ID: 1, CreatedAt: "2026-08-01T00:00:00Z", library_id: 1, title: LONG_TITLE, episode_title: LONG_EP, season: 1, episode_number: 12, is_episode: true, is_season: false, status: "missing", quality: "1080p", subtitles: "English, Español" },
]

function renderPage() {
    return render(
        <MantineProvider>
            <MemoryRouter>
                <MonitorPage />
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
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse(ITEMS)))
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
})
