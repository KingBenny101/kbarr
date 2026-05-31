import { useEffect, useMemo, useState } from "react"
import { useNavigate, useParams } from "react-router-dom"
import { ActionIcon, Anchor, Button, Card, Checkbox, Grid, Group, Image, Pagination, ScrollArea, Stack, Table, Text, TextInput, Title } from "@mantine/core"
import { modals } from "@mantine/modals"
import { IconArrowLeft, IconBell, IconExternalLink, IconInfoCircle, IconTrash } from "@tabler/icons-react"
import { API_URL, resolvePosterUrl, showToast } from "@/utils"
import type { Episode, MediaDetails } from "@/types"
import { StatusPill } from "@/components"

interface MonitoredItem {
    anidb_id: string
    is_episode: boolean
    is_season: boolean
    season: number
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
    const itemsPerPage = 10

    useEffect(() => {
        if (!id) return
        setLoading(true)
        fetch(`${API_URL}/api/library/${id}`)
            .then(async (response) => {
                if (!response.ok) throw new Error("Failed to fetch media details")
                return response.json()
            })
            .then((data: MediaDetails) => setMedia(data))
            .catch((error) => {
                console.error(error)
                showToast("Error loading media details", "error")
            })
            .finally(() => setLoading(false))
    }, [id])

    useEffect(() => {
        if (!media || !id) return
        fetch(`${API_URL}/api/library/${id}/monitored`)
            .then((response) => response.json())
            .then((data: MonitoredItem[]) => {
                setMonitoredItems(data || [])
                setMonitorEntireSeason(data?.some((item) => item.is_season && item.season === 1) ?? false)
            })
            .catch((error) => console.error("Failed to fetch monitored items", error))
    }, [id, media])

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

        const buildPayload = (episodeNumbers: number[]) => [
            ...episodeNumbers.map((episodeNumber) => {
                const episode = media.episodes?.find((entry) => Number.parseInt(entry.ep_no, 10) === episodeNumber)
                return {
                    library_id: Number(id),
                    title: media.title,
                    episode_title: episode?.title || `Episode ${episodeNumber}`,
                    season: 1,
                    episode_number: episodeNumber,
                    is_episode: true,
                    anidb_id: episode?.anidb_id || "",
                }
            }),
            {
                library_id: Number(id),
                title: media.title,
                episode_title: "",
                season: 1,
                episode_number: 0,
                is_episode: false,
                is_season: true,
                anidb_id: String(media.aid || ""),
            },
        ]

        try {
            if (monitorEntireSeason) {
                const episodeNumbers = media.episodes.map((episode) => Number.parseInt(episode.ep_no, 10)).filter((value) => !Number.isNaN(value))
                const response = await fetch(`${API_URL}/api/monitor/bulk`, {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify(buildPayload(episodeNumbers)),
                })
                if (response.ok) {
                    showToast("Season monitoring applied", "success")
                    setRangeInput("")
                }
            } else if (rangeInput.trim()) {
                const episodeNumbers = parseRange(rangeInput)
                const response = await fetch(`${API_URL}/api/monitor/bulk`, {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify(buildPayload(episodeNumbers)),
                })
                if (response.ok) {
                    showToast("Episode monitoring applied", "success")
                    setRangeInput("")
                }
            } else if (monitoredItems.some((item) => item.is_season && item.season === 1)) {
                const response = await fetch(`${API_URL}/api/unmonitor/season`, {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ library_id: Number(id), season: 1 }),
                })
                if (response.ok) {
                    showToast("Stopped monitoring entire season", "success")
                }
            } else {
                showToast("Enter a range or select entire season", "error")
            }
        } catch (error) {
            console.error(error)
            showToast("Failed to apply monitor settings", "error")
        }
    }

    const totalPages = Math.ceil((media?.episodes?.length || 0) / itemsPerPage)
    const visibleEpisodes = useMemo(
        () => media?.episodes.slice((page - 1) * itemsPerPage, page * itemsPerPage) ?? [],
        [media?.episodes, page],
    )

    const deleteMedia = async () => {
        if (!media || !id) return
        try {
            const response = await fetch(`${API_URL}/api/library/${id}`, { method: "DELETE" })
            if (response.ok) {
                showToast("Media deleted", "success")
                navigate("/")
            }
        } catch (error) {
            console.error(error)
            showToast("Failed to delete media", "error")
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
                    <ActionIcon variant="light" size="lg" onClick={() => navigate(-1)} aria-label="Go back">
                        <IconArrowLeft size={18} />
                    </ActionIcon>
                    <Stack gap={4}>
                        <Title order={1}>{media.title}</Title>
                        {media.alternate_titles !== media.title ? <Text c="dimmed">{media.alternate_titles}</Text> : null}
                    </Stack>
                </Group>

                <Button component="a" href={`https://anidb.net/anime/${media.aid}`} target="_blank" rel="noreferrer" variant="light" color="gray" leftSection={<IconExternalLink size={16} />}>
                    View on AniDB
                </Button>
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
                                <SimpleKeyValue label="AniDB ID" value={String(media.aid)} />
                                {media.total_episodes > 0 ? <SimpleKeyValue label="Episodes" value={String(media.total_episodes)} /> : null}
                                {media.total_seasons > 0 ? <SimpleKeyValue label="Seasons" value={String(media.total_seasons)} /> : null}
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

                        {media.episodes?.length ? (
                            <Card withBorder radius="xl">
                                <Stack gap="md">
                                    <Title order={3}>Episodes</Title>
                                    <EpisodeTable episodes={visibleEpisodes} monitoredItems={monitoredItems} />
                                    {totalPages > 1 ? (
                                        <Group justify="space-between" align="center">
                                            <Text size="sm" c="dimmed">
                                                Showing {(page - 1) * itemsPerPage + 1} to {Math.min(page * itemsPerPage, media.episodes.length)} of {media.episodes.length} episodes
                                            </Text>
                                            <Pagination value={page} onChange={setPage} total={totalPages} color="gray" />
                                        </Group>
                                    ) : null}
                                </Stack>
                            </Card>
                        ) : null}

                        <Group>
                        

                            <Button
                                color="red"
                                variant="light"
                                leftSection={<IconTrash size={16} />}
                                onClick={() =>
                                    modals.openConfirmModal({
                                        title: "Delete media",
                                        centered: true,
                                        children: <Text size="sm">This removes <strong>{media.title}</strong> from your library and cannot be undone.</Text>,
                                        labels: { confirm: "Delete", cancel: "Cancel" },
                                        confirmProps: { color: "red" },
                                        onConfirm: deleteMedia,
                                    })
                                }
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
                                <SimpleKeyValue label="Genres" value={media.genres || "Unknown"} />
                                <SimpleKeyValue label="Release" value={media.release_date || "Unknown"} />
                            </Stack>
                        </Card>
                    </Stack>
                </Grid.Col>
            </Grid>
        </Stack>
    )
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

function EpisodeTable({ episodes, monitoredItems }: { episodes: Episode[]; monitoredItems: MonitoredItem[] }) {
    return (
        <ScrollArea type="auto">
            <Table striped highlightOnHover withTableBorder withColumnBorders verticalSpacing="md">
                <Table.Thead>
                    <Table.Tr>
                        <Table.Th w={80}>No.</Table.Th>
                        <Table.Th>Title</Table.Th>
                        <Table.Th w={120}>Type</Table.Th>
                        <Table.Th w={160}>Availability</Table.Th>
                        <Table.Th w={160}>Status</Table.Th>
                        <Table.Th w={80}>AniDB</Table.Th>
                    </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                    {episodes.map((episode) => {
                        const monitored = monitoredItems.some((item) => item.anidb_id === episode.anidb_id && item.is_episode)
                        return (
                            <Table.Tr key={episode.ID}>
                                <Table.Td fw={700}>{episode.ep_no}</Table.Td>
                                <Table.Td>{episode.title}</Table.Td>
                                <Table.Td>
                                    <StatusPill label={episodeTypeLabel(episode.type)} tone={episodeTypeTone(episode.type)} />
                                </Table.Td>
                                <Table.Td>
                                    <StatusPill label="Unavailable" tone="gray" />
                                </Table.Td>
                                <Table.Td>
                                    <StatusPill label={monitored ? "Monitored" : "Not monitored"} tone={monitored ? "green" : "gray"} />
                                </Table.Td>
                                <Table.Td>
                                    <Anchor href={`https://anidb.net/episode/${episode.anidb_id}`} target="_blank" rel="noreferrer" c="gray">
                                        <IconExternalLink size={18} />
                                    </Anchor>
                                </Table.Td>
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
