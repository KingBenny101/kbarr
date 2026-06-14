import { useCallback, useEffect, useRef, useState } from "react"
import { Badge, Card, Center, Group, MultiSelect, Pagination, ScrollArea, Select, SimpleGrid, Stack, Text, TextInput, Title } from "@mantine/core"
import { IconCompass, IconPlayerPlayFilled, IconSearch } from "@tabler/icons-react"
import { EmptyState } from "@/components"
import { ExploreDetailModal } from "@/components/ExploreDetailModal"
import {
    displayTitle,
    fetchBrowse,
    fetchFilterOptions,
    PER_PAGE,
    type AdultMode,
    type AniListMedia,
    type BrowseFilters,
    type MediaFormat,
    type MediaSeason,
    type MediaSort,
    type MediaStatus,
} from "@/lib/anilist"

const SORT_OPTIONS: { value: MediaSort; label: string }[] = [
    { value: "POPULARITY_DESC", label: "Popularity" },
    { value: "TRENDING_DESC", label: "Trending" },
    { value: "SCORE_DESC", label: "Score" },
    { value: "FAVOURITES_DESC", label: "Favourites" },
    { value: "START_DATE_DESC", label: "Newest" },
    { value: "TITLE_ROMAJI", label: "Title (A–Z)" },
]

const SEASON_OPTIONS: { value: MediaSeason; label: string }[] = [
    { value: "WINTER", label: "Winter" },
    { value: "SPRING", label: "Spring" },
    { value: "SUMMER", label: "Summer" },
    { value: "FALL", label: "Fall" },
]

const FORMAT_OPTIONS: { value: MediaFormat; label: string }[] = [
    { value: "TV", label: "TV" },
    { value: "TV_SHORT", label: "TV Short" },
    { value: "MOVIE", label: "Movie" },
    { value: "SPECIAL", label: "Special" },
    { value: "OVA", label: "OVA" },
    { value: "ONA", label: "ONA" },
    { value: "MUSIC", label: "Music" },
]

const STATUS_OPTIONS: { value: MediaStatus; label: string }[] = [
    { value: "RELEASING", label: "Releasing" },
    { value: "FINISHED", label: "Finished" },
    { value: "NOT_YET_RELEASED", label: "Not yet released" },
    { value: "CANCELLED", label: "Cancelled" },
    { value: "HIATUS", label: "Hiatus" },
]

// Match the library page card sizing and horizontal-scroll layout.
const CARD_WIDTH = 180
const GAP = 16
const ROWS = 2

const EXPLORE_CACHE_KEY = "kbarr.explore.filters.v1"

type ExploreCache = { filters: BrowseFilters; search: string; page: number; lastPage: number; media: AniListMedia[]; scrollLeft: number; scrollTop: number }

function filterKey(filters: BrowseFilters, search: string): string {
    return JSON.stringify({ filters, search: search.trim() })
}

function readExploreCache(): ExploreCache | null {
    try {
        const raw = window.localStorage.getItem(EXPLORE_CACHE_KEY)
        if (!raw) return null
        const parsed = JSON.parse(raw) as ExploreCache
        if (!parsed.filters?.sort || !Array.isArray(parsed.media)) return null
        return {
            filters: parsed.filters,
            search: typeof parsed.search === "string" ? parsed.search : "",
            page: parsed.page ?? 1,
            lastPage: parsed.lastPage ?? 1,
            media: parsed.media,
            scrollLeft: parsed.scrollLeft ?? 0,
            scrollTop: parsed.scrollTop ?? 0,
        }
    } catch {
        return null
    }
}

const CURRENT_YEAR = new Date().getFullYear()
const YEAR_OPTIONS = Array.from({ length: CURRENT_YEAR - 1940 + 2 }, (_, i) => String(CURRENT_YEAR + 1 - i))

const ADULT_OPTIONS: { value: AdultMode; label: string }[] = [
    { value: "hide", label: "Hide NSFW" },
    { value: "show", label: "Show NSFW" },
    { value: "only", label: "Only NSFW" },
]

const DEFAULT_FILTERS: BrowseFilters = {
    sort: "POPULARITY_DESC",
    genres: [],
    tags: [],
    adultMode: "hide",
}

export function ExplorePage() {
    const [cached] = useState(() => readExploreCache())
    const [filters, setFilters] = useState<BrowseFilters>(() => cached?.filters ?? DEFAULT_FILTERS)
    const [search, setSearch] = useState(() => cached?.search ?? "")
    const [genreOptions, setGenreOptions] = useState<string[]>([])
    const [tagOptions, setTagOptions] = useState<string[]>([])
    const [media, setMedia] = useState<AniListMedia[]>(() => cached?.media ?? [])
    const [page, setPage] = useState(() => cached?.page ?? 1)
    const [lastPage, setLastPage] = useState(() => cached?.lastPage ?? 1)
    const [loading, setLoading] = useState(false)
    const [hasLoaded, setHasLoaded] = useState(() => cached !== null)
    const [selected, setSelected] = useState<AniListMedia | null>(null)
    const [modalOpened, setModalOpened] = useState(false)
    const viewportRef = useRef<HTMLDivElement>(null)
    const scrollXRef = useRef(cached?.scrollLeft ?? 0)
    const scrollYRef = useRef(cached?.scrollTop ?? 0)
    // Set when a fresh load (page/filter change) should snap the new results to
    // the start — applied only once the new media renders, not on click.
    const resetHScrollRef = useRef(false)
    // Serialized filters+search currently displayed. Initialized from cache so a
    // remount (incl. StrictMode's double effect run) doesn't reset the page.
    const loadedKeyRef = useRef<string | null>(cached ? filterKey(cached.filters, cached.search) : null)

    useEffect(() => {
        fetchFilterOptions()
            .then((opts) => {
                setGenreOptions(opts.genres)
                setTagOptions(opts.tags)
            })
            .catch((error) => console.error("Failed to load filter options:", error))
    }, [])

    const load = useCallback(async (activeFilters: BrowseFilters, activeSearch: string, activePage: number) => {
        setLoading(true)
        try {
            const result = await fetchBrowse({ ...activeFilters, search: activeSearch }, activePage)
            // Snap to the start once the new results render (after this state update).
            resetHScrollRef.current = true
            setMedia(result.media)
            setLastPage(Math.min(result.pageInfo.lastPage, Math.ceil(5000 / PER_PAGE)))
            setHasLoaded(true)
        } catch (error) {
            console.error("AniList browse failed:", error)
        } finally {
            setLoading(false)
        }
    }, [])

    // Reload (and reset to page 1) only when filters/search actually differ from
    // what's currently shown. Restored cached state matches, so it won't refetch.
    useEffect(() => {
        const key = filterKey(filters, search)
        if (loadedKeyRef.current === key) return
        const handle = setTimeout(() => {
            loadedKeyRef.current = key
            setPage(1)
            load(filters, search, 1)
        }, 350)
        return () => clearTimeout(handle)
    }, [filters, search, load])

    // Persist filters, search, page and results so navigating away and back
    // restores the exact view (including pagination).
    useEffect(() => {
        try {
            window.localStorage.setItem(EXPLORE_CACHE_KEY, JSON.stringify({ filters, search, page, lastPage, media, scrollLeft: scrollXRef.current, scrollTop: scrollYRef.current } satisfies ExploreCache))
        } catch {
            // ignore cache write errors
        }
    }, [filters, search, page, lastPage, media])

    // Restore the horizontal and vertical scroll positions once the grid renders.
    useEffect(() => {
        if (!cached || media.length === 0) return
        const frame = requestAnimationFrame(() => {
            if (cached.scrollLeft) viewportRef.current?.scrollTo({ left: cached.scrollLeft })
            if (cached.scrollTop) window.scrollTo({ top: cached.scrollTop })
        })
        return () => cancelAnimationFrame(frame)
        // Only on first mount.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    // Track the window's vertical scroll into a ref (no re-renders). The persist
    // effect reads this ref, so it never writes a spurious 0 at mount/teardown.
    useEffect(() => {
        const onScroll = () => { scrollYRef.current = window.scrollY }
        window.addEventListener("scroll", onScroll, { passive: true })
        return () => window.removeEventListener("scroll", onScroll)
    }, [])

    // Persist the latest scroll positions to the cache on unmount.
    useEffect(() => {
        return () => {
            try {
                const raw = window.localStorage.getItem(EXPLORE_CACHE_KEY)
                if (!raw) return
                const parsed = JSON.parse(raw) as ExploreCache
                parsed.scrollLeft = scrollXRef.current
                parsed.scrollTop = scrollYRef.current
                window.localStorage.setItem(EXPLORE_CACHE_KEY, JSON.stringify(parsed))
            } catch {
                // ignore cache write errors
            }
        }
    }, [])

    const handlePageChange = (next: number) => {
        setPage(next)
        // Don't reset the scroll here — the current page stays in place until the
        // new page's results arrive (handled by the media effect below).
        load(filters, search, next)
    }

    // When new results render from a load, snap the row back to the start. Skips
    // the cache-restore case so a returning view keeps its saved scroll position.
    useEffect(() => {
        if (!resetHScrollRef.current) return
        resetHScrollRef.current = false
        scrollXRef.current = 0
        viewportRef.current?.scrollTo({ left: 0 })
    }, [media])

    const update = <K extends keyof BrowseFilters>(key: K, value: BrowseFilters[K]) => {
        setFilters((prev) => ({ ...prev, [key]: value }))
    }

    const openDetails = (item: AniListMedia) => {
        setSelected(item)
        setModalOpened(true)
    }

    return (
        <Stack gap="lg">
            <Title order={1}>Explore</Title>

            <TextInput
                value={search}
                onChange={(event) => setSearch(event.currentTarget.value)}
                placeholder="Search anime..."
                leftSection={<IconSearch size={18} />}
                size="md"
            />

            <SimpleGrid cols={{ base: 1, sm: 2, lg: 3 }} spacing="sm">
                <MultiSelect label="Genres" data={genreOptions} value={filters.genres ?? []} onChange={(v) => update("genres", v)} searchable clearable placeholder="Any" />
                <MultiSelect label="Tags" data={tagOptions} value={filters.tags ?? []} onChange={(v) => update("tags", v)} searchable clearable placeholder="Any" limit={100} />
                <Select label="Sort" data={SORT_OPTIONS} value={filters.sort} onChange={(v) => v && update("sort", v as MediaSort)} allowDeselect={false} />
                <Select label="Season" data={SEASON_OPTIONS} value={filters.season ?? null} onChange={(v) => update("season", (v as MediaSeason) || undefined)} clearable placeholder="Any" />
                <Select label="Year" data={YEAR_OPTIONS} value={filters.seasonYear ? String(filters.seasonYear) : null} onChange={(v) => update("seasonYear", v ? Number(v) : undefined)} searchable clearable placeholder="Any" />
                <Select label="Format" data={FORMAT_OPTIONS} value={filters.format ?? null} onChange={(v) => update("format", (v as MediaFormat) || undefined)} clearable placeholder="Any" />
                <Select label="Status" data={STATUS_OPTIONS} value={filters.status ?? null} onChange={(v) => update("status", (v as MediaStatus) || undefined)} clearable placeholder="Any" />
                <Select label="Adult content" data={ADULT_OPTIONS} value={filters.adultMode} onChange={(v) => v && update("adultMode", v as AdultMode)} allowDeselect={false} />
            </SimpleGrid>

            {media.length > 0 ? (
                <Stack gap="lg">
                    <ScrollArea scrollbars="x" type="always" scrollbarSize={6} viewportRef={viewportRef} onScrollPositionChange={({ x }) => { scrollXRef.current = x }}>
                    <div style={{ display: "grid", gridTemplateRows: `repeat(${ROWS}, auto)`, gridAutoFlow: "column", gridAutoColumns: `${CARD_WIDTH}px`, gap: GAP, paddingBottom: GAP, width: "max-content" }}>
                        {media.map((item) => {
                            const title = displayTitle(item)
                            const poster = item.coverImage.large ?? item.coverImage.extraLarge
                            return (
                                <Card key={item.id} withBorder radius="xl" p={0} h="100%" style={{ overflow: "hidden", cursor: "pointer", position: "relative" }} onClick={() => openDetails(item)}>
                                    {item.isAdult && (
                                        <Badge color="red" size="xs" style={{ position: "absolute", top: 8, right: 8, zIndex: 1 }}>
                                            NSFW
                                        </Badge>
                                    )}
                                    {poster ? (
                                        <div style={{ width: "100%", aspectRatio: "3 / 4" }}>
                                            <img src={poster} alt={title} style={{ width: "100%", height: "100%", objectFit: "cover" }} />
                                        </div>
                                    ) : (
                                        <Center style={{ width: "100%", aspectRatio: "3 / 4", background: "rgba(255,255,255,0.04)" }}>
                                            <IconPlayerPlayFilled size={30} opacity={0.5} />
                                        </Center>
                                    )}
                                    <Stack gap={0} p="xs" style={{ minHeight: 64 }}>
                                        <Title order={6} lineClamp={2}>
                                            {title}
                                        </Title>
                                        <Text size="10px" c="dimmed">
                                            {typeof item.averageScore === "number" ? `${item.averageScore}% · ` : ""}
                                            {item.format ?? "—"}
                                        </Text>
                                    </Stack>
                                </Card>
                            )
                        })}
                    </div>
                    </ScrollArea>

                    <Group justify="space-between" align="center">
                        <Text size="sm" c="dimmed">Page {page} of {lastPage}</Text>
                        <Pagination value={page} onChange={handlePageChange} total={Math.max(lastPage, 1)} color="gray" />
                    </Group>
                </Stack>
            ) : null}

            {!loading && hasLoaded && media.length === 0 ? (
                <EmptyState icon={<IconCompass size={28} />} title="No results" description="No anime matched these filters. Try widening your selection." />
            ) : null}

            {!hasLoaded && loading ? (
                <EmptyState icon={<IconCompass size={28} />} title="Loading..." description="Fetching anime to explore." />
            ) : null}

            <ExploreDetailModal media={selected} opened={modalOpened} onClose={() => setModalOpened(false)} />
        </Stack>
    )
}
