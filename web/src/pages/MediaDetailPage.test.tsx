import { render, screen, waitFor, fireEvent } from "@testing-library/react"
import { MantineProvider } from "@mantine/core"
import { MemoryRouter, Route, Routes } from "react-router"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { MediaDetailPage } from "./MediaDetailPage"
import { apiFetch } from "@/utils"
import type { Episode, MediaDetails } from "@/types"

const MEDIA: MediaDetails = {
    ID: 9,
    CreatedAt: "2026-08-01T00:00:00Z",
    UpdatedAt: "2026-08-01T00:00:00Z",
    title: "Grand Blue Season 3",
    source: "anidb",
    source_id: "19600",
    alternate_titles: "",
    description: "A test show.",
    release_date: "2026-01-01",
    end_date: "",
    genres: "Comedy",
    poster_url: "",
    total_episodes: 10,
    total_seasons: 1,
    tvdb_id: "",
    anilist_id: "",
    imdb_id: "",
    tmdb_id: "",
    mal_id: "",
    kitsu_id: "",
    animeplanet_id: "",
    anisearch_id: "",
    episodes: [],
    is_nsfw: false,
}

const EPISODES: Episode[] = [
    { ID: 1, source: "anidb", external_id: "313608", type: 1, ep_no: "1", title: "Episode 1", air_date: "" },
    { ID: 2, source: "anidb", external_id: "313609", type: 1, ep_no: "2", title: "Episode 2", air_date: "" },
    { ID: 3, source: "anidb", external_id: "313610", type: 1, ep_no: "3", title: "Episode 3", air_date: "" },
    { ID: 4, source: "anidb", external_id: "313611", type: 1, ep_no: "4", title: "Episode 4", air_date: "" },
    { ID: 5, source: "anidb", external_id: "314048", type: 1, ep_no: "5", title: "Episode 5", air_date: "" },
    { ID: 6, source: "anidb", external_id: "314049", type: 1, ep_no: "6", title: "Episode 6", air_date: "" },
    { ID: 7, source: "anidb", external_id: "314050", type: 1, ep_no: "7", title: "Episode 7", air_date: "" },
    { ID: 8, source: "anidb", external_id: "314051", type: 1, ep_no: "8", title: "Episode 8", air_date: "" },
    { ID: 9, source: "anidb", external_id: "313664", type: 3, ep_no: "C1", title: "Opening", air_date: "" },
    { ID: 10, source: "anidb", external_id: "313665", type: 3, ep_no: "C2", title: "Ending", air_date: "" },
]

type MonitoredItem = {
    external_id: string
    source: string
    is_episode: boolean
    is_season: boolean
    season: number
    episode_number: number
    monitored: boolean
}

const EPISODE_RESPONSE = {
    episodes: EPISODES,
    total: EPISODES.length,
    page: 1,
    limit: 10,
    present_types: [1, 3],
}

// Simulates the server's monitor table: POST /api/monitor upserts with the
// monitored flag as sent (Go zero-value false when the field is omitted), and
// POST /api/unmonitor flips monitored to false.
let monitored: MonitoredItem[]
const monitorPosts: { body: Record<string, unknown> }[] = []

function upsert(body: Record<string, unknown>) {
    const existing = monitored.find((m) => m.external_id === body.external_id && m.is_episode === body.is_episode)
    if (existing) {
        existing.monitored = (body.monitored as boolean) ?? false
    } else {
        monitored.push({
            external_id: String(body.external_id),
            source: String(body.source),
            is_episode: body.is_episode as boolean,
            is_season: body.is_season as boolean,
            season: body.season as number,
            episode_number: body.episode_number as number,
            monitored: (body.monitored as boolean) ?? false,
        })
    }
}

function unmonitor(body: Record<string, unknown>) {
    for (const m of monitored) {
        if (m.external_id === body.external_id) m.monitored = false
    }
}

function json(data: unknown): Response {
    return new Response(JSON.stringify(data), { status: 200, headers: { "Content-Type": "application/json" } })
}

vi.mock("@/utils", async (importOriginal) => {
    const actual = await importOriginal<typeof import("@/utils")>()
    return { ...actual, apiFetch: vi.fn() }
})

const mockedApiFetch = vi.mocked(apiFetch)

function renderPage() {
    return render(
        <MantineProvider>
            <MemoryRouter initialEntries={["/media/9"]}>
                <Routes>
                    <Route path="/media/:id" element={<MediaDetailPage />} />
                </Routes>
            </MemoryRouter>
        </MantineProvider>,
    )
}

function seasonCheckbox(): HTMLInputElement {
    const el = document.querySelector("input.mantine-Checkbox-input")
    if (!el) throw new Error("season checkbox not found")
    return el as HTMLInputElement
}

async function clickPill(label: string, index = 0) {
    const pills = await waitFor(() => screen.getAllByText(label))
    fireEvent.click(pills[index])
}

beforeEach(() => {
    monitored = []
    monitorPosts.length = 0
    mockedApiFetch.mockImplementation(async (input: string, init?: RequestInit) => {
        const url = String(input)
        const method = init?.method ?? "GET"
        const body = init?.body ? (JSON.parse(String(init.body)) as Record<string, unknown>) : null

        if (method === "POST" && url.endsWith("/api/unmonitor")) {
            unmonitor(body!)
            return new Response(null, { status: 204 })
        }
        if (method === "POST" && url.endsWith("/api/monitor")) {
            monitorPosts.push({ body: body! })
            upsert(body!)
            return new Response(null, { status: 204 })
        }
        if (url.includes("/api/library/9/monitored")) {
            return json(monitored)
        }
        if (url.includes("/api/library/9/episodes")) {
            return json(EPISODE_RESPONSE)
        }
        if (url.endsWith("/api/library/9")) {
            return json(MEDIA)
        }
        throw new Error(`unexpected request: ${method} ${url}`)
    })
})

describe("MediaDetailPage season monitor sync", () => {
    it("checks the season box when the last regular episode becomes monitored, ignoring unmonitored credits", async () => {
        // Episodes 1-7 monitored; credits and episode 8 unmonitored; no season record yet.
        const regulars = EPISODES.filter((e) => !Number.isNaN(Number.parseInt(e.ep_no, 10)))
        monitored = regulars.filter((e) => e.external_id !== "314051").map((e) => ({
            external_id: e.external_id,
            source: e.source,
            is_episode: true,
            is_season: false,
            season: 1,
            episode_number: Number.parseInt(e.ep_no, 10) || 0,
            monitored: true,
        }))

        renderPage()

        await waitFor(() => expect(screen.getAllByText("Monitored")).toHaveLength(7))

        await clickPill("Not monitored")

        await waitFor(() => expect(seasonCheckbox().checked).toBe(true))
        const seasonPost = monitorPosts.find((p) => p.body.is_season === true)
        expect(seasonPost).toBeDefined()
        expect(seasonPost!.body.monitored).toBe(true)
        expect(monitorPosts.some((p) => !p.body.is_season && (p.body.external_id === "313664" || p.body.external_id === "313665"))).toBe(false)
    })

    it("unchecks the season box and unmonitors the season record when an episode is unmonitored", async () => {
        monitored = EPISODES.map((e) => ({
            external_id: e.external_id,
            source: e.source,
            is_episode: true,
            is_season: false,
            season: 1,
            episode_number: Number.parseInt(e.ep_no, 10) || 0,
            monitored: true,
        }))
        monitored.push({
            external_id: "19600",
            source: "anidb",
            is_episode: false,
            is_season: true,
            season: 1,
            episode_number: 0,
            monitored: true,
        })

        renderPage()

        await waitFor(() => expect(screen.getAllByText("Monitored")).toHaveLength(10))

        await clickPill("Monitored")

        await waitFor(() => expect(seasonCheckbox().checked).toBe(false))
        expect(monitored.find((m) => m.is_season)?.monitored).toBe(false)
        expect(monitored.filter((m) => m.is_episode && m.monitored)).toHaveLength(9)
    })

    it("keeps the season box checked when only a credit is unmonitored", async () => {
        monitored = EPISODES.map((e) => ({
            external_id: e.external_id,
            source: e.source,
            is_episode: true,
            is_season: false,
            season: 1,
            episode_number: Number.parseInt(e.ep_no, 10) || 0,
            monitored: true,
        }))
        monitored.push({
            external_id: "19600",
            source: "anidb",
            is_episode: false,
            is_season: true,
            season: 1,
            episode_number: 0,
            monitored: true,
        })

        renderPage()

        await waitFor(() => expect(screen.getAllByText("Monitored")).toHaveLength(10))

        await clickPill("Monitored", 8)

        await waitFor(() => expect(seasonCheckbox().checked).toBe(true))
        expect(monitored.find((m) => m.is_season)?.monitored).toBe(true)
        expect(monitored.find((m) => m.is_episode && m.external_id === "313664")?.monitored).toBe(false)
    })

    it("re-checks the season box when the last episode is monitored again after an unmonitor", async () => {
        monitored = EPISODES.map((e) => ({
            external_id: e.external_id,
            source: e.source,
            is_episode: true,
            is_season: false,
            season: 1,
            episode_number: Number.parseInt(e.ep_no, 10) || 0,
            monitored: true,
        }))
        monitored.push({
            external_id: "19600",
            source: "anidb",
            is_episode: false,
            is_season: true,
            season: 1,
            episode_number: 0,
            monitored: true,
        })

        renderPage()

        await waitFor(() => expect(screen.getAllByText("Monitored")).toHaveLength(10))

        await clickPill("Monitored")
        await waitFor(() => expect(seasonCheckbox().checked).toBe(false))

        await clickPill("Not monitored")

        await waitFor(() => expect(seasonCheckbox().checked).toBe(true))
        const seasonPost = monitorPosts.find((p) => p.body.is_season === true)
        expect(seasonPost).toBeDefined()
        expect(seasonPost!.body.monitored).toBe(true)
    })

    it("shows the air date column and a type pill only for non-regular episodes", async () => {
        monitored = EPISODES.map((e) => ({
            external_id: e.external_id,
            source: e.source,
            is_episode: true,
            is_season: false,
            season: 1,
            episode_number: Number.parseInt(e.ep_no, 10) || 0,
            monitored: true,
        }))
        EPISODES[0].air_date = "2026-04-06T00:00:00Z"
        renderPage()

        await waitFor(() => expect(screen.getAllByText("Monitored")).toHaveLength(10))

        expect(screen.getByText("Air date")).toBeDefined()
        expect(screen.getByText("2026-04-06")).toBeDefined()
        // Filter chips render once per present type; the credits also get row
        // pills because they are non-regular episodes.
        expect(screen.getAllByText("Credit")).toHaveLength(3)
        expect(screen.getAllByText("Regular")).toHaveLength(1)
        expect(screen.getAllByText("EP")).toHaveLength(8)
        expect(screen.getAllByText("CR")).toHaveLength(2)
        expect(screen.queryByText("SP")).toBeNull()
    })
})
