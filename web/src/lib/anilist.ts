// AniList public GraphQL API. No auth required; rate limited to ~90 req/min, so
// we serialize requests through a small queue with a minimum spacing.
const ANILIST_ENDPOINT = "https://graphql.anilist.co"

async function request<T>(query: string, variables?: Record<string, unknown>): Promise<T> {
    const response = await fetch(ANILIST_ENDPOINT, {
        method: "POST",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ query, variables }),
    })
    if (!response.ok) {
        throw new Error(`AniList request failed with status ${response.status}`)
    }
    const body = (await response.json()) as { data?: T; errors?: { message?: string }[] }
    if (!body.data) {
        throw new Error(body.errors?.[0]?.message ?? "AniList request failed")
    }
    return body.data
}

// Minimum spacing between requests (~90/min => 1 per 700ms leaves headroom).
const MIN_REQUEST_SPACING_MS = 700
let requestChain: Promise<unknown> = Promise.resolve()

function throttle<T>(run: () => Promise<T>): Promise<T> {
    const result = requestChain.then(run)
    // Keep the chain alive regardless of success/failure, spaced out in time.
    requestChain = result.then(
        () => new Promise((resolve) => setTimeout(resolve, MIN_REQUEST_SPACING_MS)),
        () => new Promise((resolve) => setTimeout(resolve, MIN_REQUEST_SPACING_MS)),
    )
    return result
}

export type MediaFormat = "TV" | "TV_SHORT" | "MOVIE" | "SPECIAL" | "OVA" | "ONA" | "MUSIC"
export type MediaSeason = "WINTER" | "SPRING" | "SUMMER" | "FALL"
export type MediaStatus = "FINISHED" | "RELEASING" | "NOT_YET_RELEASED" | "CANCELLED" | "HIATUS"
export type MediaSort = "POPULARITY_DESC" | "SCORE_DESC" | "TRENDING_DESC" | "FAVOURITES_DESC" | "START_DATE_DESC" | "TITLE_ROMAJI"

export type BrowseFilters = {
    search?: string
    genres?: string[]
    tags?: string[]
    season?: MediaSeason
    seasonYear?: number
    format?: MediaFormat
    status?: MediaStatus
    sort: MediaSort
    // Adult/NSFW visibility: "hide" excludes it (default), "show" includes it
    // alongside SFW, "only" returns adult titles exclusively.
    adultMode: AdultMode
}

export type AdultMode = "hide" | "show" | "only"

export type AniListMedia = {
    id: number
    title: { romaji: string | null; english: string | null; native: string | null }
    coverImage: { large: string | null; extraLarge: string | null; color: string | null }
    bannerImage: string | null
    format: MediaFormat | null
    status: MediaStatus | null
    episodes: number | null
    averageScore: number | null
    genres: string[]
    season: MediaSeason | null
    seasonYear: number | null
    countryOfOrigin: string | null
    isAdult: boolean
}

export type AniListMediaDetails = AniListMedia & {
    description: string | null
    duration: number | null
    studios: { nodes: { name: string }[] }
    tags: { name: string; rank: number; isAdult: boolean }[]
    startDate: { year: number | null; month: number | null; day: number | null }
    siteUrl: string | null
}

export type PageInfo = {
    total: number
    currentPage: number
    lastPage: number
    hasNextPage: boolean
    perPage: number
}

export type BrowseResult = {
    pageInfo: PageInfo
    media: AniListMedia[]
}

const MEDIA_FIELDS = `
    id
    title { romaji english native }
    coverImage { large extraLarge color }
    bannerImage
    format
    status
    episodes
    averageScore
    genres
    season
    seasonYear
    countryOfOrigin
    isAdult
`

const BROWSE_QUERY = `
    query Browse(
        $page: Int
        $perPage: Int
        $search: String
        $genres: [String]
        $tags: [String]
        $season: MediaSeason
        $seasonYear: Int
        $format: MediaFormat
        $status: MediaStatus
        $sort: [MediaSort]
        $countryOfOrigin: CountryCode
        $isAdult: Boolean
    ) {
        Page(page: $page, perPage: $perPage) {
            pageInfo { total currentPage lastPage hasNextPage perPage }
            media(
                type: ANIME
                search: $search
                genre_in: $genres
                tag_in: $tags
                season: $season
                seasonYear: $seasonYear
                format: $format
                status: $status
                sort: $sort
                countryOfOrigin: $countryOfOrigin
                isAdult: $isAdult
            ) {
                ${MEDIA_FIELDS}
            }
        }
    }
`

const DETAILS_QUERY = `
    query Details($id: Int) {
        Media(id: $id, type: ANIME) {
            ${MEDIA_FIELDS}
            description(asHtml: false)
            duration
            studios(isMain: true) { nodes { name } }
            tags { name rank isAdult }
            startDate { year month day }
            siteUrl
        }
    }
`

const FILTER_OPTIONS_QUERY = `
    query FilterOptions {
        GenreCollection
        MediaTagCollection { name isAdult }
    }
`

export const PER_PAGE = 24

export async function fetchBrowse(filters: BrowseFilters, page: number): Promise<BrowseResult> {
    const variables: Record<string, unknown> = {
        page,
        perPage: PER_PAGE,
        sort: [filters.sort],
        search: filters.search?.trim() || undefined,
        genres: filters.genres?.length ? filters.genres : undefined,
        tags: filters.tags?.length ? filters.tags : undefined,
        season: filters.season,
        seasonYear: filters.seasonYear,
        format: filters.format,
        status: filters.status,
        // Always restrict to Japanese anime (the only content with AniDB mappings).
        countryOfOrigin: "JP",
        // The upstream API only filters adult content when isAdult is sent;
        // leaving it unset returns everything (SFW + NSFW mixed).
        isAdult: filters.adultMode === "show" ? undefined : filters.adultMode === "only",
    }
    const data = await throttle(() => request<{ Page: BrowseResult }>(BROWSE_QUERY, variables))
    return data.Page
}

export async function fetchMediaDetails(id: number): Promise<AniListMediaDetails> {
    const data = await throttle(() => request<{ Media: AniListMediaDetails }>(DETAILS_QUERY, { id }))
    return data.Media
}

const FILTER_CACHE_KEY = "kbarr.explore.filterOptions.v1"

export type FilterOptions = { genres: string[]; tags: string[] }

export async function fetchFilterOptions(): Promise<FilterOptions> {
    try {
        const cached = window.localStorage.getItem(FILTER_CACHE_KEY)
        if (cached) return JSON.parse(cached) as FilterOptions
    } catch {
        // ignore cache read errors and refetch
    }

    const data = await throttle(() =>
        request<{ GenreCollection: string[]; MediaTagCollection: { name: string; isAdult: boolean }[] }>(FILTER_OPTIONS_QUERY),
    )
    const options: FilterOptions = {
        genres: data.GenreCollection.filter(Boolean),
        tags: data.MediaTagCollection.map((t) => t.name),
    }
    try {
        window.localStorage.setItem(FILTER_CACHE_KEY, JSON.stringify(options))
    } catch {
        // ignore cache write errors
    }
    return options
}

export function displayTitle(media: Pick<AniListMedia, "title">): string {
    return media.title.english || media.title.romaji || media.title.native || "Untitled"
}
