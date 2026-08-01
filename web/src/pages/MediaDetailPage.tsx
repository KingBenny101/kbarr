import { useCallback, useEffect, useRef, useState } from "react"
import { useMediaQuery } from "@mantine/hooks"
import { useNavigate, useParams } from "react-router"
import { ActionIcon, Anchor, Button, Card, Checkbox, Grid, Group, Image, Pagination, ScrollArea, Stack, Switch, Table, Text, TextInput, Title } from "@mantine/core"
import { modals } from "@mantine/modals"
import { IconArrowLeft, IconBell, IconExternalLink, IconInfoCircle, IconRefresh, IconTrash } from "@tabler/icons-react"
import { API_URL, apiFetch, resolvePosterUrl, showToast } from "@/utils"
import type { Episode, MediaDetails } from "@/types"
import { StatusPill } from "@/components"

type SortField = "ep_no" | "title"
type SortDir = "asc" | "desc"

const ALL_TYPES = [1, 2, 3, 4, 5] as const
const TYPE_LABELS: Record<number, string> = { 1: "Regular", 2: "Special", 3: "Credit", 4: "Trailer", 5: "Parody" }

interface EpisodesResponse {
    episodes: Episode[]
    total: number
    page: number
    limit: number
    present_types: number[]
}

interface MonitoredItem {
    id?: number
    external_id: string
    source: string
    is_episode: boolean
    is_season: boolean
    season: number
    status?: string
    available?: boolean
    monitored?: boolean
    quality?: string
    subtitles?: string
}

// DeleteConfirmBody owns the "delete files" checkbox state. The confirm modal
// captures its children once, so the checkbox can't bind to parent state; it
// keeps its own and reports changes upward through onChange.
function DeleteConfirmBody({ title, onChange }: { title: string; onChange: (v: boolean) => void }) {
    const [checked, setChecked] = useState(false)
    return (
        <Stack gap="sm">
            <Text size="sm">This removes <strong>{title}</strong> from your library and cannot be undone.</Text>
            <Checkbox
                label="Also delete downloaded files from the media folder"
                description="Removes the show's organised folder and its hardlinks."
                checked={checked}
                onChange={(e) => { setChecked(e.currentTarget.checked); onChange(e.currentTarget.checked) }}
            />
        </Stack>
    )
}

export function MediaDetailPage() {
    const { id } = useParams<{ id: string }>()
    const navigate = useNavigate()
    const [media, setMedia] = useState<MediaDetails | null>(null)
    const [loading, setLoading] = useState(true)
    const [monitoredItems, setMonitoredItems] = useState<MonitoredItem[]>([])
    const [rangeInput, setRangeInput] = useState("")
    const [monitorEntireSeason, setMonitorEntireSeason] = useState(false)
    const [page, setPage] = useState(1)
    const [sortField, setSortField] = useState<SortField>("ep_no")
    const [sortDir, setSortDir] = useState<SortDir>("asc")
    const [activeTypes, setActiveTypes] = useState<Set<number>>(() => new Set(ALL_TYPES))
    const [episodesData, setEpisodesData] = useState<EpisodesResponse | null>(null)
    const [refreshing, setRefreshing] = useState(false)
    const [refreshNonce, setRefreshNonce] = useState(0)
    const deleteFilesRef = useRef(false)

    // Show the spinner when navigating between media in the same component
    // instance; the initial value already starts at true for first mount.
    const [prevId, setPrevId] = useState(id)
    if (prevId !== id) {
        setPrevId(id)
        setLoading(true)
    }

    useEffect(() => {
        if (!id) return
        apiFetch(`${API_URL}/api/library/${id}`)
            .then(async (response) => {
                if (!response.ok) throw new Error("Failed to fetch media details")
                return response.json()
            })
            .then((data: MediaDetails) => setMedia(data))
            .catch((error) => {
                console.error(error)
                showToast("Error loading media details", "error")
            })
            .finally(() => { setLoading(false) })
    }, [id])

    const refreshDetails = () => {
        if (!id) return
        apiFetch(`${API_URL}/api/library/${id}`)
            .then(async (response) => {
                if (!response.ok) throw new Error("Failed to fetch media details")
                return response.json()
            })
            .then((data: MediaDetails) => setMedia(data))
            .catch((error) => {
                console.error(error)
                showToast("Error loading media details", "error")
            })
    }

    const handleRefresh = async () => {
        if (!id) return
        setRefreshing(true)
        try {
            const res = await apiFetch(`${API_URL}/api/library/${id}/refresh`, { method: "POST" })
            if (!res.ok) throw new Error(res.statusText)
            const data = await res.json() as { message?: string; newEpisodes?: number }
            showToast(data.message || "Refreshed", "success")
            refreshDetails()
            fetchMonitored()
            setRefreshNonce((n) => n + 1)
        } catch (error) {
            console.error("Failed to refresh metadata", error)
            showToast("Failed to refresh metadata", "error")
        } finally {
            setRefreshing(false)
        }
    }

    const fetchMonitored = useCallback(() => {
        if (!id) return
        apiFetch(`${API_URL}/api/library/${id}/monitored`)
            .then((response) => {
                if (!response.ok) throw new Error(response.statusText)
                return response.json()
            })
            .then((data: MonitoredItem[]) => {
                setMonitoredItems(data || [])
                setMonitorEntireSeason(data?.some((item) => item.is_season && item.season === 1 && item.monitored) ?? false)
            })
            .catch((error) => console.error("Failed to fetch monitored items", error))
    }, [id])

    useEffect(() => {
        if (!media || !id) return
        fetchMonitored()
    }, [id, media, fetchMonitored])

    const parseRange = (input: string): number[] => {
        const values = new Set<number>()
        for (const part of input.split(",").map((entry) => entry.trim()).filter(Boolean)) {
            if (part.includes("-")) {
                const [start, end] = part.split("-").map((entry) => Number.parseInt(entry.trim(), 10))
                if (!Number.isNaN(start) && !Number.isNaN(end)) {
                    for (let current = Math.min(start, end); current <= Math.max(start, end); current += 1) {
                        values.add(current)
                    }
                }
            } else {
                const value = Number.parseInt(part, 10)
                if (!Number.isNaN(value)) values.add(value)
            }
        }
        return [...values].sort((left, right) => left - right)
    }

    const handleBulkMonitor = async () => {
        if (!media || !id) return

        try {
            // Fetch all episodes to resolve external_ids for any episode number
            const allEpsRes = await apiFetch(`${API_URL}/api/library/${id}/episodes?limit=10000`)
            const allEpsData = await allEpsRes.json() as EpisodesResponse
            const allEpisodes = allEpsData.episodes ?? []

            const buildPayload = (episodeNumbers: number[], includeSeason: boolean) => [
                ...episodeNumbers.map((episodeNumber) => {
                    const episode = allEpisodes.find((e) => Number.parseInt(e.ep_no, 10) === episodeNumber)
                    return {
                        library_id: Number(id),
                        title: media.title,
                        episode_title: episode?.title || `Episode ${episodeNumber}`,
                        season: 1,
                        episode_number: episodeNumber,
                        is_episode: true,
                        is_season: false,
                        source: episode?.source || media.source,
                        external_id: episode?.external_id || "",
                        monitored: true,
                    }
                }),
                ...(includeSeason ? [{
                    library_id: Number(id),
                    title: media.title,
                    episode_title: "",
                    season: 1,
                    episode_number: 0,
                    is_episode: false,
                    is_season: true,
                    source: media.source,
                    external_id: media.source_id,
                    monitored: true,
                }] : []),
            ]

            if (monitorEntireSeason) {
                const episodeNumbers = allEpisodes.map((e) => Number.parseInt(e.ep_no, 10)).filter((v) => !Number.isNaN(v))
                const response = await apiFetch(`${API_URL}/api/monitor/bulk`, {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify(buildPayload(episodeNumbers, true)),
                })
                if (response.ok) {
                    showToast("Season monitoring applied", "success")
                    setRangeInput("")
                    fetchMonitored()
                } else {
                    showToast("Failed to apply season monitoring", "error")
                }
            } else if (rangeInput.trim()) {
                const episodeNumbers = parseRange(rangeInput)
                const response = await apiFetch(`${API_URL}/api/monitor/bulk`, {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify(buildPayload(episodeNumbers, false)),
                })
                if (response.ok) {
                    showToast("Episode monitoring applied", "success")
                    setRangeInput("")
                    fetchMonitored()
                } else {
                    showToast("Failed to apply episode monitoring", "error")
                }
            } else if (monitoredItems.some((item) => item.is_season && item.season === 1 && item.monitored)) {
                const response = await apiFetch(`${API_URL}/api/unmonitor/season`, {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ library_id: Number(id), season: 1 }),
                })
                if (response.ok) {
                    showToast("Stopped monitoring entire season", "success")
                    fetchMonitored()
                } else {
                    showToast("Failed to stop monitoring", "error")
                }
            } else {
                showToast("Enter a range or select entire season", "error")
            }
        } catch (error) {
            console.error(error)
            showToast("Failed to apply monitor settings", "error")
        }
    }

    const handleMonitorEpisode = async (episode: Episode) => {
        if (!media || !id) return
        const monRes = await apiFetch(`${API_URL}/api/monitor`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                library_id: Number(id),
                title: media.title,
                episode_title: episode.title,
                season: 1,
                episode_number: Number.parseInt(episode.ep_no, 10) || 0,
                is_episode: true,
                is_season: false,
                source: episode.source,
                external_id: episode.external_id,
                monitored: true,
            }),
        })
        if (!monRes.ok) {
            showToast("Failed to monitor episode", "error")
            return
        }

        // Immediately reflect the change in local state
        const existing = monitoredItems.find((i) => i.external_id === episode.external_id && i.is_episode)
        const updatedMonitored = existing
            ? monitoredItems.map((i) => i.external_id === episode.external_id && i.is_episode ? { ...i, monitored: true } : i)
            : [...monitoredItems, { external_id: episode.external_id, source: episode.source, is_episode: true, is_season: false, season: 1, monitored: true }]

        // Check if all episodes are now monitored — if so, add season monitor
        const allEpsRes = await apiFetch(`${API_URL}/api/library/${id}/episodes?limit=10000`)
        const allEpsData = await allEpsRes.json() as EpisodesResponse
        const allEpisodes = allEpsData.episodes ?? []
        const monitoredIds = new Set(updatedMonitored.filter((i) => i.is_episode && i.monitored).map((i) => i.external_id))
        const allMonitored = allEpisodes.length > 0 && allEpisodes.every((ep) => monitoredIds.has(ep.external_id))

        if (allMonitored && !monitoredItems.some((i) => i.is_season && i.monitored)) {
            await apiFetch(`${API_URL}/api/monitor`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    library_id: Number(id),
                    title: media.title,
                    episode_title: "",
                    season: 1,
                    episode_number: 0,
                    is_episode: false,
                    is_season: true,
                    source: media.source,
                    external_id: media.source_id,
                }),
            })
            fetchMonitored()
        } else {
            setMonitoredItems(updatedMonitored)
        }
    }

    const handleUnmonitorEpisode = async (episode: Episode) => {
        if (!media || !id) return
        const seasonMonitor = monitoredItems.find((item) => item.is_season)

        const unmonRes = await apiFetch(`${API_URL}/api/unmonitor`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ library_id: Number(id), external_id: episode.external_id }),
        })
        if (!unmonRes.ok) {
            showToast("Failed to unmonitor episode", "error")
            return
        }

        if (seasonMonitor) {
            // Remove only the season-level monitor record, not all episode monitors
            await apiFetch(`${API_URL}/api/unmonitor`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ library_id: Number(id), external_id: seasonMonitor.external_id }),
            })
            setMonitorEntireSeason(false)
        }

        // Immediately reflect unmonitor in local state by toggling monitored=false
        setMonitoredItems((prev) =>
            prev.map((item) => {
                if (item.external_id === episode.external_id && item.is_episode) return { ...item, monitored: false }
                if (seasonMonitor && item.is_season && item.external_id === seasonMonitor.external_id) return { ...item, monitored: false }
                return item
            })
        )
    }

    useEffect(() => {
        if (!id) return
        const params = new URLSearchParams({
            sort: sortField,
            order: sortDir,
            page: String(page),
            limit: "10",
        })
        if (activeTypes.size < ALL_TYPES.length) {
            params.set("types", [...activeTypes].join(","))
        }
        apiFetch(`${API_URL}/api/library/${id}/episodes?${params}`)
            .then((r) => {
                if (!r.ok) throw new Error(r.statusText)
                return r.json()
            })
            .then((data: EpisodesResponse) => setEpisodesData(data))
            .catch((error) => console.error("Failed to fetch episodes", error))
    }, [id, page, sortField, sortDir, activeTypes, refreshNonce])

    const handleSort = (field: SortField) => {
        if (sortField === field) {
            setSortDir((d) => (d === "asc" ? "desc" : "asc"))
        } else {
            setSortField(field)
            setSortDir("asc")
        }
        setPage(1)
    }

    const toggleType = (type: number) => {
        setActiveTypes((prev) => {
            const next = new Set(prev)
            if (next.has(type)) {
                if (next.size > 1) next.delete(type)
            } else {
                next.add(type)
            }
            return next
        })
        setPage(1)
    }

    const totalPages = Math.ceil((episodesData?.total ?? 0) / 10)

    const deleteMedia = async (deleteFiles: boolean) => {
        if (!media || !id) return
        try {
            const url = `${API_URL}/api/library/${id}${deleteFiles ? "?deleteFiles=true" : ""}`
            const response = await apiFetch(url, { method: "DELETE" })
            if (response.ok) {
                showToast("Media deleted", "success")
                navigate("/")
            } else {
                showToast("Failed to delete media", "error")
            }
        } catch (error) {
            console.error(error)
            showToast("Failed to delete media", "error")
        }
    }

    const handleNSFWToggle = async (nsfw: boolean) => {
        if (!media || !id) return
        setMedia({ ...media, is_nsfw: nsfw })
        try {
            await apiFetch(`${API_URL}/api/library/${id}/nsfw`, {
                method: "PATCH",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ nsfw }),
            })
        } catch (error) {
            console.error(error)
            setMedia({ ...media, is_nsfw: !nsfw })
            showToast("Failed to update NSFW status", "error")
        }
    }

    if (loading) {
        return <Text c="dimmed" ta="center" py="xl">Loading...</Text>
    }

    if (!media) {
        return <Text c="red" fw={700} ta="center" py="xl">Media not found</Text>
    }

    return (
        <Stack gap="lg">
            <Group align="start" justify="space-between">
                <Group align="start" gap="md">
                    <ActionIcon variant="light" size="lg" onClick={() => navigate("/")} aria-label="Go back">
                        <IconArrowLeft size={18} />
                    </ActionIcon>
                    <Stack gap={4}>
                        <Title order={1}>{media.title}</Title>
                        {media.alternate_titles ? (
                            <Text size="sm" c="dimmed">
                                {media.alternate_titles.split("|").filter((t) => t.trim() && t.trim() !== media.title).join(" · ")}
                            </Text>
                        ) : null}
                    </Stack>
                </Group>
            </Group>

            <Grid>
                <Grid.Col span={{ base: 12, md: 8 }}>
                    <Stack gap="lg">
                        <Card withBorder radius="xl">
                            <Stack gap="sm">
                                <Title order={3}>Overview</Title>
                                <Text c="dimmed" lh={1.7}>
                                    {media.description || "No description available."}
                                </Text>
                            </Stack>
                        </Card>

                        <Card withBorder radius="xl">
                            <Stack gap="sm">
                                <Title order={3}>Information</Title>
                                {media.total_episodes > 0 ? <SimpleKeyValue label="Episodes" value={String(media.total_episodes)} /> : null}
                                {media.total_seasons > 0 ? <SimpleKeyValue label="Seasons" value={String(media.total_seasons)} /> : null}
                                {media.genres ? <SimpleKeyValue label="Genres" value={media.genres} /> : null}
                                {media.release_date ? <SimpleKeyValue label="Released" value={media.release_date} /> : null}
                                <SimpleKeyValue label={media.end_date ? "Ended" : "Status"} value={media.end_date || airingStatus(media)} />
                                <Group justify="space-between" align="center" mt={4}>
                                    <Text c="dimmed" size="sm">NSFW</Text>
                                    <Switch
                                        checked={media.is_nsfw}
                                        onChange={(e) => handleNSFWToggle(e.currentTarget.checked)}
                                        color="red"
                                    />
                                </Group>
                                {externalLinks(media).length > 0 && (
                                    <Group gap="xs" mt="xs">
                                        {externalLinks(media).map((link) => (
                                            <Button
                                                key={link.label}
                                                component="a"
                                                href={link.href}
                                                target="_blank"
                                                rel="noreferrer"
                                                variant="outline"
                                                color="gray"
                                                leftSection={<IconExternalLink size={14} />}
                                                size="xs"
                                            >
                                                {link.label}
                                            </Button>
                                        ))}
                                    </Group>
                                )}
                            </Stack>
                        </Card>

                        <Card withBorder radius="xl">
                            <Stack gap="md">
                                <Group align="start" justify="space-between">
                                    <div>
                                        <Title order={3}>Monitor anime</Title>
                                        <Text size="sm" c="dimmed">
                                            Choose whole-season tracking or pass specific episode ranges.
                                        </Text>
                                    </div>
                                    <IconBell size={20} color="var(--mantine-color-dimmed)" />
                                </Group>

                                <Group justify="space-between" align="center">
                                    <div>
                                        <Text fw={700}>Monitor entire season</Text>
                                        <Text size="sm" c="dimmed">Automatically track all episodes in this season.</Text>
                                    </div>
                                    <Checkbox checked={monitorEntireSeason} onChange={(event) => setMonitorEntireSeason(event.currentTarget.checked)} />
                                </Group>

                                <Stack gap={6}>
                                    <Text fw={700}>Monitor specific episodes</Text>
                                    <TextInput value={rangeInput} onChange={(event) => setRangeInput(event.currentTarget.value)} placeholder="ex: 1-5, 8, 10-12" />
                                    <Group gap={6} align="center">
                                        <IconInfoCircle size={14} />
                                        <Text size="xs" c="dimmed">
                                            Use comma separated numbers or ranges like 1-10, 15, 20-25.
                                        </Text>
                                    </Group>
                                </Stack>

                                <Button color="gray" onClick={handleBulkMonitor}>Apply changes</Button>
                            </Stack>
                        </Card>

                        {(episodesData?.total ?? 0) > 0 || episodesData !== null ? (
                            <Card withBorder radius="xl">
                                <Stack gap="md" style={{ minHeight: (episodesData?.limit ?? 10) * 41 + 122 }}>
                                    <Group justify="space-between" align="center">
                                        <Title order={3}>Episodes</Title>
                                        {(episodesData?.present_types?.length ?? 0) > 1 && (
                                            <Group gap="xs">
                                                {episodesData!.present_types.map((type) => (
                                                    <Button
                                                        key={type}
                                                        size="xs"
                                                        variant={activeTypes.has(type) ? "filled" : "light"}
                                                        color="gray"
                                                        onClick={() => toggleType(type)}
                                                    >
                                                        {TYPE_LABELS[type]}
                                                    </Button>
                                                ))}
                                            </Group>
                                        )}
                                    </Group>
                                    <EpisodeTable
                                        episodes={episodesData?.episodes ?? []}
                                        monitoredItems={monitoredItems}
                                        sortField={sortField}
                                        sortDir={sortDir}
                                        onSort={handleSort}
                                        onMonitor={handleMonitorEpisode}
                                        onUnmonitor={handleUnmonitorEpisode}
                                    />
                                    {totalPages > 1 ? (
                                        <Group justify="space-between" align="center" mt="auto">
                                            <Text size="sm" c="dimmed">
                                                {(page - 1) * 10 + 1}–{Math.min(page * 10, episodesData?.total ?? 0)} of {episodesData?.total ?? 0}
                                            </Text>
                                            <Pagination value={page} onChange={setPage} total={totalPages} color="gray" />
                                        </Group>
                                    ) : null}
                                </Stack>
                            </Card>
                        ) : null}

                        <Group>
                            <Button
                                variant="light"
                                color="gray"
                                leftSection={<IconRefresh size={16} />}
                                loading={refreshing}
                                onClick={handleRefresh}
                            >
                                Refresh
                            </Button>

                            <Button
                                color="red"
                                variant="light"
                                leftSection={<IconTrash size={16} />}
                                onClick={() => {
                                    deleteFilesRef.current = false
                                    modals.openConfirmModal({
                                        title: "Delete media",
                                        centered: true,
                                        children: <DeleteConfirmBody title={media.title} onChange={(v) => { deleteFilesRef.current = v }} />,
                                        labels: { confirm: "Delete", cancel: "Cancel" },
                                        confirmProps: { color: "red" },
                                        onConfirm: () => deleteMedia(deleteFilesRef.current),
                                    })
                                }}
                            >
                                Delete media
                            </Button>
                        </Group>
                    </Stack>
                </Grid.Col>

                <Grid.Col span={{ base: 12, md: 4 }}>
                    <Stack gap="lg" pos="sticky" top={96}>
                        <Card withBorder radius="xl" p={0} style={{ overflow: "hidden" }}>
                            <Image src={resolvePosterUrl(media.poster_url)} alt={media.title} radius="xl" />
                        </Card>

                        <Card withBorder radius="xl">
                            <Stack gap={6}>
                                <Text size="sm" c="dimmed" tt="uppercase" fw={700}>Metadata</Text>
                                <SimpleKeyValue label="Created" value={media.CreatedAt ? new Date(media.CreatedAt).toLocaleString() : "Unknown"} />
                                <SimpleKeyValue label="Updated" value={media.UpdatedAt ? new Date(media.UpdatedAt).toLocaleString() : "Unknown"} />
                            </Stack>
                        </Card>
                    </Stack>
                </Grid.Col>
            </Grid>
        </Stack>
    )
}

// airingStatus describes a show with no end date: it has either started airing
// (Airing) or its start date is still in the future (Not yet aired). When there
// is no usable start date we can only say it has not ended.
function airingStatus(media: MediaDetails): string {
    if (!media.release_date) return "Ongoing"
    const start = new Date(media.release_date)
    if (isNaN(start.getTime())) return "Ongoing"
    return start.getTime() > Date.now() ? "Not yet aired" : "Airing"
}

function externalLinks(media: MediaDetails): { label: string; href: string }[] {
    const links: { label: string; href: string }[] = []
    if (media.source === "anidb" && media.source_id) {
        links.push({ label: "AniDB", href: `https://anidb.net/anime/${media.source_id}` })
    }
    if (media.anilist_id) links.push({ label: "AniList", href: `https://anilist.co/anime/${media.anilist_id}` })
    if (media.mal_id) links.push({ label: "MyAnimeList", href: `https://myanimelist.net/anime/${media.mal_id}` })
    if (media.kitsu_id) links.push({ label: "Kitsu", href: `https://kitsu.io/anime/${media.kitsu_id}` })
    if (media.animeplanet_id) links.push({ label: "Anime-Planet", href: `https://anime-planet.com/anime/${media.animeplanet_id}` })
    if (media.anisearch_id) links.push({ label: "aniSearch", href: `https://anisearch.com/anime/${media.anisearch_id}` })
    if (media.tvdb_id) links.push({ label: "TheTVDB", href: `https://thetvdb.com/dereferrer/series/${media.tvdb_id}` })
    if (media.tmdb_id) links.push({ label: "TMDB", href: `https://www.themoviedb.org/tv/${media.tmdb_id}` })
    if (media.imdb_id) links.push({ label: "IMDb", href: `https://www.imdb.com/title/${media.imdb_id}` })
    return links
}

function SimpleKeyValue({ label, value }: { label: string; value: string }) {
    return (
        <Group justify="space-between" align="start" wrap="nowrap">
            <Text c="dimmed" size="sm">
                {label}
            </Text>
            <Text fw={600} ta="right" style={{ wordBreak: "break-word" }}>
                {value}
            </Text>
        </Group>
    )
}

function episodeSourceUrl(episode: Episode): string | null {
    if (episode.source === "anidb") return `https://anidb.net/episode/${episode.external_id}`
    return null
}

function SortIndicator({ field, sortField, sortDir }: { field: SortField; sortField: SortField; sortDir: SortDir }) {
    if (sortField !== field) return <Text span c="dimmed" size="xs"> ↕</Text>
    return <Text span size="xs"> {sortDir === "asc" ? "↑" : "↓"}</Text>
}

function EpisodeTable({ episodes, monitoredItems, sortField, sortDir, onSort, onMonitor, onUnmonitor }: {
    episodes: Episode[]
    monitoredItems: MonitoredItem[]
    sortField: SortField
    sortDir: SortDir
    onSort: (field: SortField) => void
    onMonitor: (episode: Episode) => void
    onUnmonitor: (episode: Episode) => void
}) {
    const hasLinks = episodes.some((e) => episodeSourceUrl(e) !== null)
    const isMobile = useMediaQuery("(max-width: 768px)")

    if (isMobile) {
        return (
            <Stack gap="xs">
                {episodes.length === 0 ? (
                    <Text ta="center" py="md" c="dimmed">No episodes found.</Text>
                ) : (
                    episodes.map((episode) => {
                        const monitorEntry = monitoredItems.find((item) => item.external_id === episode.external_id && item.source === episode.source && item.is_episode)
                        const available = monitorEntry?.available === true
                        const monitored = monitorEntry?.monitored === true
                        const quality = monitorEntry?.quality?.toUpperCase() || ""
                        const subtitles = (monitorEntry?.subtitles || "").split(",").filter(Boolean).map((s) => s.toUpperCase()).join(", ")
                        const link = episodeSourceUrl(episode)
                        return (
                            <Card withBorder radius="md" p="sm" key={episode.ID}>
                                <Group justify="space-between" align="center" wrap="nowrap">
                                    <Group gap="xs" wrap="nowrap" style={{ flex: 1, minWidth: 0 }}>
                                        <Text fw={700} size="sm">{episode.ep_no}</Text>
                                        <Text size="sm" truncate>{episode.title}</Text>
                                    </Group>
                                    <StatusPill label={episodeTypeLabel(episode.type)} tone={episodeTypeTone(episode.type)} />
                                </Group>
                                <Group gap="xs" mt={4}>
                                    <StatusPill label={available ? "Available" : "Unavailable"} tone={available ? "green" : "gray"} />
                                    {quality && <Text size="xs" c="dimmed">{quality}</Text>}
                                    {subtitles && <Text size="xs" c="dimmed">{subtitles}</Text>}
                                    <div style={{ flex: 1 }} />
                                    <StatusPill
                                        label={monitored ? "Monitored" : "Not monitored"}
                                        tone={monitored ? "green" : "gray"}
                                        onClick={() => monitored ? onUnmonitor(episode) : onMonitor(episode)}
                                    />
                                    {link && (
                                        <Anchor href={link} target="_blank" rel="noreferrer" c="gray">
                                            <IconExternalLink size={14} />
                                        </Anchor>
                                    )}
                                </Group>
                            </Card>
                        )
                    })
                )}
            </Stack>
        )
    }

    return (
        <ScrollArea type="auto">
            <Table striped highlightOnHover withTableBorder withColumnBorders w="100%">
                <Table.Thead>
                    <Table.Tr>
                        <Table.Th style={{ cursor: "pointer", textAlign: "center", whiteSpace: "nowrap" }} onClick={() => onSort("ep_no")}>
                            No.<SortIndicator field="ep_no" sortField={sortField} sortDir={sortDir} />
                        </Table.Th>
                        <Table.Th style={{ cursor: "pointer" }} onClick={() => onSort("title")}>
                            Title<SortIndicator field="title" sortField={sortField} sortDir={sortDir} />
                        </Table.Th>
                        <Table.Th style={{ textAlign: "center", whiteSpace: "nowrap" }}>Type</Table.Th>
                        <Table.Th style={{ textAlign: "center", whiteSpace: "nowrap" }}>Availability</Table.Th>
                        <Table.Th style={{ textAlign: "center", whiteSpace: "nowrap" }}>Quality</Table.Th>
                        <Table.Th style={{ textAlign: "center", whiteSpace: "nowrap" }}>Subtitles</Table.Th>
                        <Table.Th style={{ textAlign: "center", whiteSpace: "nowrap" }}>Monitor</Table.Th>
                        {hasLinks && <Table.Th style={{ textAlign: "center", whiteSpace: "nowrap" }}>Link</Table.Th>}
                    </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                    {episodes.map((episode) => {
                        const monitorEntry = monitoredItems.find((item) => item.external_id === episode.external_id && item.source === episode.source && item.is_episode)
                        const available = monitorEntry?.available === true
                        const monitored = monitorEntry?.monitored === true
                        const quality = monitorEntry?.quality?.toUpperCase() || ""
                        const subtitles = (monitorEntry?.subtitles || "").split(",").filter(Boolean).map((s) => s.toUpperCase()).join(", ")
                        const link = episodeSourceUrl(episode)
                        return (
                            <Table.Tr key={episode.ID}>
                                <Table.Td fw={700} style={{ textAlign: "center" }}>{episode.ep_no}</Table.Td>
                                <Table.Td style={{ maxWidth: 420, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{episode.title}</Table.Td>
                                <Table.Td style={{ whiteSpace: "nowrap", textAlign: "center" }}>
                                    <StatusPill label={episodeTypeLabel(episode.type)} tone={episodeTypeTone(episode.type)} />
                                </Table.Td>
                                <Table.Td style={{ whiteSpace: "nowrap", textAlign: "center" }}>
                                    <StatusPill label={available ? "Available" : "Unavailable"} tone={available ? "green" : "gray"} />
                                </Table.Td>
                                <Table.Td style={{ whiteSpace: "nowrap", textAlign: "center" }}>
                                    <Text size="sm" c={quality ? undefined : "dimmed"}>{quality || "—"}</Text>
                                </Table.Td>
                                <Table.Td style={{ whiteSpace: "nowrap", textAlign: "center" }}>
                                    <Text size="sm" c={subtitles ? undefined : "dimmed"}>{subtitles || "—"}</Text>
                                </Table.Td>
                                <Table.Td style={{ whiteSpace: "nowrap", textAlign: "center" }}>
                                    <StatusPill
                                        label={monitored ? "Monitored" : "Not monitored"}
                                        tone={monitored ? "green" : "gray"}
                                        onClick={() => monitored ? onUnmonitor(episode) : onMonitor(episode)}
                                    />
                                </Table.Td>
                                {hasLinks && (
                                    <Table.Td style={{ textAlign: "center" }}>
                                        {link ? (
                                            <Anchor href={link} target="_blank" rel="noreferrer" c="gray">
                                                <IconExternalLink size={18} />
                                            </Anchor>
                                        ) : null}
                                    </Table.Td>
                                )}
                            </Table.Tr>
                        )
                    })}
                </Table.Tbody>
            </Table>
        </ScrollArea>
    )
}

function episodeTypeLabel(type: number) {
    switch (type) {
        case 1: return "Regular"
        case 2: return "Special"
        case 3: return "Credit"
        case 4: return "Trailer"
        case 5: return "Parody"
        default: return "Other"
    }
}

function episodeTypeTone(type: number) {
    switch (type) {
        case 1: return "blue"
        case 2: return "violet"
        case 3: return "green"
        case 4: return "gray"
        case 5: return "red"
        default: return "gray"
    }
}
