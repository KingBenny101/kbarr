import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
    displayTitle,
    fetchBrowse,
    fetchFilterOptions,
    fetchMediaDetails,
    PER_PAGE,
    type BrowseFilters,
    type FilterOptions,
} from "./anilist"

const ENDPOINT = "https://graphql.anilist.co"

let fetchMock: ReturnType<typeof vi.fn>

function jsonResponse(data: unknown, status = 200): Response {
    return new Response(JSON.stringify(data), {
        status,
        headers: { "Content-Type": "application/json" },
    })
}

function lastRequestBody(): { query: string; variables: Record<string, unknown> } {
    const [, init] = fetchMock.mock.calls.at(-1) as [string | URL, RequestInit]
    return JSON.parse(String(init.body)) as { query: string; variables: Record<string, unknown> }
}

function browseFilters(overrides: Partial<BrowseFilters> = {}): BrowseFilters {
    return { sort: "POPULARITY_DESC", ...overrides } as BrowseFilters
}

beforeEach(() => {
    fetchMock = vi.fn()
    vi.stubGlobal("fetch", fetchMock)
    window.localStorage.clear()
})

afterEach(() => {
    vi.unstubAllGlobals()
})

describe("fetchBrowse", () => {
    it("posts the query and variables to the AniList endpoint", async () => {
        fetchMock.mockResolvedValue(
            jsonResponse({ data: { Page: { pageInfo: { hasNextPage: false }, media: [] } } }),
        )

        const result = await fetchBrowse(browseFilters(), 3)

        expect(fetchMock).toHaveBeenCalledTimes(1)
        const [url, init] = fetchMock.mock.calls[0] as [string | URL, RequestInit]
        expect(String(url)).toBe(ENDPOINT)
        expect(init.method).toBe("POST")
        expect(new Headers(init.headers).get("content-type")).toMatch(/^application\/json/)

        const body = lastRequestBody()
        expect(body.query).toContain("query Browse")
        expect(body.variables).toEqual({
            page: 3,
            perPage: PER_PAGE,
            sort: ["POPULARITY_DESC"],
            countryOfOrigin: "JP",
            isAdult: false,
        })
        expect(result).toEqual({ pageInfo: { hasNextPage: false }, media: [] })
    })

    it("trims the search term and omits empty filters", async () => {
        fetchMock.mockResolvedValue(jsonResponse({ data: { Page: { media: [] } } }))

        await fetchBrowse(
            browseFilters({
                search: "  One Piece  ",
                genres: [],
                tags: [],
                season: undefined,
                seasonYear: undefined,
                format: undefined,
                status: undefined,
            }),
            1,
        )

        const body = lastRequestBody()
        expect(body.variables.search).toBe("One Piece")
        expect("genres" in body.variables).toBe(false)
        expect("tags" in body.variables).toBe(false)
        expect("season" in body.variables).toBe(false)
        expect("seasonYear" in body.variables).toBe(false)
        expect("format" in body.variables).toBe(false)
        expect("status" in body.variables).toBe(false)
    })

    it("includes genres and tags when provided", async () => {
        fetchMock.mockResolvedValue(jsonResponse({ data: { Page: { media: [] } } }))

        await fetchBrowse(browseFilters({ genres: ["Action"], tags: ["a", "b"] }), 1)

        const body = lastRequestBody()
        expect(body.variables.genres).toEqual(["Action"])
        expect(body.variables.tags).toEqual(["a", "b"])
    })

    it.each([
        ["hide", false],
        ["show", "absent"],
        ["only", true],
    ] as const)("maps adultMode %s to isAdult in variables", async (adultMode, expected) => {
        fetchMock.mockResolvedValue(jsonResponse({ data: { Page: { media: [] } } }))

        await fetchBrowse(browseFilters({ adultMode }), 1)

        const body = lastRequestBody()
        if (expected === "absent") {
            expect("isAdult" in body.variables).toBe(false)
        } else {
            expect(body.variables.isAdult).toBe(expected)
        }
    })

    it("rejects on a non-ok response", async () => {
        fetchMock.mockResolvedValue(jsonResponse({}, 429))

        await expect(fetchBrowse(browseFilters(), 1)).rejects.toThrow()
    })

    it("rejects when the response contains GraphQL errors", async () => {
        fetchMock.mockResolvedValue(jsonResponse({ errors: [{ message: "boom" }] }))

        await expect(fetchBrowse(browseFilters(), 1)).rejects.toThrow()
    })
})

describe("fetchMediaDetails", () => {
    it("posts the details query with the id and returns the media", async () => {
        const media = { id: 123, title: { romaji: "Test" } }
        fetchMock.mockResolvedValue(jsonResponse({ data: { Media: media } }))

        const result = await fetchMediaDetails(123)

        const body = lastRequestBody()
        expect(body.query).toContain("query Details")
        expect(body.query).toContain("siteUrl")
        expect(body.variables).toEqual({ id: 123 })
        expect(result).toEqual(media)
    })
})

describe("fetchFilterOptions", () => {
    it("fetches once, caches in localStorage, and reuses the cache", async () => {
        fetchMock.mockResolvedValue(
            jsonResponse({
                data: {
                    GenreCollection: ["Action", "Comedy"],
                    MediaTagCollection: [
                        { name: "Tag A", isAdult: false },
                        { name: "Tag B", isAdult: true },
                    ],
                },
            }),
        )

        const first = await fetchFilterOptions()
        const second = await fetchFilterOptions()

        const expected: FilterOptions = { genres: ["Action", "Comedy"], tags: ["Tag A", "Tag B"] }
        expect(first).toEqual(expected)
        expect(second).toEqual(expected)
        expect(fetchMock).toHaveBeenCalledTimes(1)
        expect(window.localStorage.getItem("kbarr.explore.filterOptions.v1")).toBe(JSON.stringify(expected))
    })

    it("does not refetch when a valid cache entry exists", async () => {
        window.localStorage.setItem(
            "kbarr.explore.filterOptions.v1",
            JSON.stringify({ genres: ["Drama"], tags: ["Tag C"] }),
        )

        const result = await fetchFilterOptions()

        expect(fetchMock).not.toHaveBeenCalled()
        expect(result).toEqual({ genres: ["Drama"], tags: ["Tag C"] })
    })
})

describe("displayTitle", () => {
    it("prefers english, then romaji, then native, then a fallback", () => {
        expect(displayTitle({ title: { english: "Eng", romaji: "Rom", native: "Nat" } })).toBe("Eng")
        expect(displayTitle({ title: { english: null, romaji: "Rom", native: "Nat" } })).toBe("Rom")
        expect(displayTitle({ title: { english: null, romaji: null, native: "Nat" } })).toBe("Nat")
        expect(displayTitle({ title: { english: null, romaji: null, native: null } })).toBe("Untitled")
    })
})

describe("throttle", () => {
    it("spaces consecutive requests at least 650ms apart", async () => {
        const times: number[] = []
        fetchMock.mockImplementation(async () => {
            times.push(Date.now())
            return jsonResponse({ data: { Media: { id: 1 } } })
        })

        await Promise.all([fetchMediaDetails(1), fetchMediaDetails(2)])
        expect(fetchMock).toHaveBeenCalledTimes(2)
        expect(times[1] - times[0]).toBeGreaterThanOrEqual(650)
    })
})
