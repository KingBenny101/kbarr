import { fireEvent, render, screen } from "@testing-library/react"
import { MantineProvider } from "@mantine/core"
import { MemoryRouter } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { ExplorePage } from "./ExplorePage"
import { fetchBrowse } from "@/lib/anilist"
import type { AniListMedia, BrowseResult } from "@/lib/anilist"

const MEDIA: AniListMedia[] = [
    {
        id: 1,
        title: { romaji: "Test Anime", english: "Test Anime", native: null },
        coverImage: { large: null, extraLarge: null, color: null },
        bannerImage: null,
        format: "TV",
        status: "FINISHED",
        episodes: 12,
        averageScore: 80,
        genres: ["Action"],
        season: "WINTER",
        seasonYear: 2026,
        countryOfOrigin: "JP",
        isAdult: false,
    },
]

const BROWSE_RESULT: BrowseResult = {
    pageInfo: { total: 1, currentPage: 1, lastPage: 1, hasNextPage: false, perPage: 24 },
    media: MEDIA,
}

vi.mock("@/lib/anilist", () => ({
    displayTitle: (media: { title: { romaji: string | null; english: string | null; native: string | null } }) =>
        media.title.english || media.title.romaji || media.title.native || "Untitled",
    fetchBrowse: vi.fn(),
    fetchFilterOptions: vi.fn(async () => ({ genres: [], tags: [] })),
    PER_PAGE: 24,
}))

const mockedFetchBrowse = vi.mocked(fetchBrowse)

function renderPage() {
    return render(
        <MantineProvider>
            <MemoryRouter>
                <ExplorePage />
            </MemoryRouter>
        </MantineProvider>,
    )
}

beforeEach(() => {
    window.localStorage.clear()
    mockedFetchBrowse.mockReset()
})

afterEach(() => {
    vi.clearAllMocks()
})

describe("ExplorePage", () => {
    it("shows a visible error state when AniList refuses the request", async () => {
        mockedFetchBrowse.mockRejectedValue(new TypeError("NetworkError when attempting to fetch resource"))

        renderPage()

        expect(await screen.findByText("Couldn't load anime")).toBeInTheDocument()
        expect(screen.getByText(/AniList is refusing this connection/i)).toBeInTheDocument()
        expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument()
    })

    it("retries the failed load when the user clicks Try again", async () => {
        mockedFetchBrowse.mockRejectedValueOnce(new TypeError("NetworkError when attempting to fetch resource"))
        mockedFetchBrowse.mockResolvedValueOnce(BROWSE_RESULT)

        renderPage()

        const retry = await screen.findByRole("button", { name: "Try again" })
        fireEvent.click(retry)

        expect(await screen.findByText("Test Anime")).toBeInTheDocument()
        expect(mockedFetchBrowse).toHaveBeenCalledTimes(2)
    })

    it("renders media after a successful browse", async () => {
        mockedFetchBrowse.mockResolvedValue(BROWSE_RESULT)

        renderPage()

        expect(await screen.findByText("Test Anime")).toBeInTheDocument()
    })
})
