import { useEffect, useMemo, useState } from "react"
import { ActionIcon, Card, Group, Pagination, ScrollArea, Stack, Table, Text, Title, UnstyledButton } from "@mantine/core"
import { IconChevronDown, IconChevronUp, IconSelector, IconTrash } from "@tabler/icons-react"
import { Link } from "react-router-dom"
import { API_URL, apiFetch, showToast } from "@/utils"
import { StatusPill } from "@/components"

interface MonitorEntry {
    ID: number
    CreatedAt: string
    library_id: number
    title: string
    episode_title: string
    season: number
    episode_number: number
    is_episode: boolean
    is_season: boolean
    status: string
    quality?: string
    subtitles?: string
}

const STATUS_PRIORITY: Record<string, number> = {
    downloading: 0,
    queued:      1,
    searching:   2,
    pending:     3,
    monitored:   4,
    missing:     5,
    available:   6,
    downloaded:  7,
    unmonitored: 8,
}

type SortField = "title" | "status" | "type"
type SortDir = "asc" | "desc"

const ALL_STATUSES = ["monitored", "searching", "pending", "queued", "downloading", "missing", "available", "downloaded", "unmonitored"] as const
// "active" (the default) shows everything still in progress — i.e. every item
// that is not yet downloaded; "all" includes downloaded items too.
type StatusFilter = typeof ALL_STATUSES[number] | "all" | "active"

const STATUS_TONES: Record<string, "gray" | "blue" | "green" | "yellow" | "violet" | "red"> = {
    monitored: "blue",
    pending: "gray",
    searching: "yellow",
    queued: "violet",
    downloading: "blue",
    missing: "red",
    available: "green",
    downloaded: "green",
    unmonitored: "gray",
}

function SortIcon({ field, sortField, sortDir }: { field: SortField; sortField: SortField; sortDir: SortDir }) {
    if (field !== sortField) return <IconSelector size={14} style={{ opacity: 0.4 }} />
    return sortDir === "asc" ? <IconChevronUp size={14} /> : <IconChevronDown size={14} />
}

export function MonitorPage() {
    const [monitors, setMonitors] = useState<MonitorEntry[]>([])
    const [loading, setLoading] = useState(true)
    const [currentPage, setCurrentPage] = useState(1)
    const [sortField, setSortField] = useState<SortField>("status")
    const [sortDir, setSortDir] = useState<SortDir>("asc")
    const [activeStatus, setActiveStatus] = useState<StatusFilter>("active")
    const itemsPerPage = 9

    const fetchMonitors = async () => {
        try {
            const response = await apiFetch(`${API_URL}/api/monitor`)
            if (!response.ok) throw new Error("Failed to fetch monitored items")
            const data = await response.json()
            setMonitors(data || [])
        } catch (error) {
            console.error(error)
            showToast("Failed to load monitored items", "error")
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => { fetchMonitors() }, [])

    const handleDelete = async (id: number) => {
        try {
            const response = await apiFetch(`${API_URL}/api/monitor/${id}`, { method: "DELETE" })
            if (!response.ok) throw new Error("Failed to delete monitor entry")
            setMonitors((items) => items.filter((item) => item.ID !== id))
            showToast("Removed from monitor", "success")
        } catch (error) {
            console.error(error)
            showToast("Failed to remove item", "error")
        }
    }

    const toggleSort = (field: SortField) => {
        if (sortField === field) {
            setSortDir((d) => d === "asc" ? "desc" : "asc")
        } else {
            setSortField(field)
            setSortDir("asc")
        }
        setCurrentPage(1)
    }

    const setFilter = (s: StatusFilter) => {
        setActiveStatus(s)
        setCurrentPage(1)
    }

    const statusCounts = useMemo(() => {
        const counts: Record<string, number> = {}
        for (const m of monitors) counts[m.status] = (counts[m.status] ?? 0) + 1
        return counts
    }, [monitors])

    const activeCount = useMemo(() => monitors.filter((m) => m.status !== "downloaded").length, [monitors])

    const filtered = useMemo(() => {
        let list: MonitorEntry[]
        if (activeStatus === "all") list = monitors
        else if (activeStatus === "active") list = monitors.filter((m) => m.status !== "downloaded")
        else list = monitors.filter((m) => m.status === activeStatus)
        list = [...list].sort((a, b) => {
            let cmp = 0
            if (sortField === "title") cmp = a.title.localeCompare(b.title)
            else if (sortField === "status") cmp = (STATUS_PRIORITY[a.status] ?? 99) - (STATUS_PRIORITY[b.status] ?? 99)
            else if (sortField === "type") cmp = (a.is_season ? 0 : 1) - (b.is_season ? 0 : 1)
            return sortDir === "asc" ? cmp : -cmp
        })
        return list
    }, [monitors, activeStatus, sortField, sortDir])

    const totalPages = Math.ceil(filtered.length / itemsPerPage)
    const startIndex = (currentPage - 1) * itemsPerPage
    const currentMonitors = filtered.slice(startIndex, startIndex + itemsPerPage)

    return (
        <Stack gap="lg">
            <Title order={1}>Monitored items</Title>

            <Card withBorder radius="xl">
                <Stack gap="md">
                    <Group justify="space-between" align="flex-start">
                        <div>
                            <Title order={3}>Currently monitoring</Title>
                            <Text c="dimmed" size="sm">
                                {filtered.length !== monitors.length
                                    ? `${filtered.length} of ${monitors.length} items`
                                    : `${monitors.length} item${monitors.length !== 1 ? "s" : ""}`}
                            </Text>
                        </div>
                        <Group gap="xs" wrap="wrap" justify="flex-end">
                            <StatusPill
                                label={`Active (${activeCount})`}
                                tone={activeStatus === "active" ? "blue" : "gray"}
                                onClick={() => setFilter("active")}
                            />
                            <StatusPill
                                label={`All (${monitors.length})`}
                                tone={activeStatus === "all" ? "blue" : "gray"}
                                onClick={() => setFilter("all")}
                            />
                            {ALL_STATUSES.filter((s) => statusCounts[s]).map((s) => (
                                <StatusPill
                                    key={s}
                                    label={`${s} (${statusCounts[s]})`}
                                    tone={activeStatus === s ? STATUS_TONES[s] : "gray"}
                                    onClick={() => setFilter(s)}
                                />
                            ))}
                        </Group>
                    </Group>

                    <ScrollArea type="auto">
                        <Table striped highlightOnHover withTableBorder withColumnBorders verticalSpacing="md">
                            <Table.Thead>
                                <Table.Tr>
                                    <Table.Th>
                                        <UnstyledButton onClick={() => toggleSort("title")} style={{ display: "flex", alignItems: "center", gap: 4 }}>
                                            Anime <SortIcon field="title" sortField={sortField} sortDir={sortDir} />
                                        </UnstyledButton>
                                    </Table.Th>
                                    <Table.Th>
                                        <UnstyledButton onClick={() => toggleSort("type")} style={{ display: "flex", alignItems: "center", gap: 4 }}>
                                            Type <SortIcon field="type" sortField={sortField} sortDir={sortDir} />
                                        </UnstyledButton>
                                    </Table.Th>
                                    <Table.Th>Details</Table.Th>
                                    <Table.Th>
                                        <UnstyledButton onClick={() => toggleSort("status")} style={{ display: "flex", alignItems: "center", gap: 4 }}>
                                            Status <SortIcon field="status" sortField={sortField} sortDir={sortDir} />
                                        </UnstyledButton>
                                    </Table.Th>
                                    <Table.Th>Quality</Table.Th>
                                    <Table.Th>Subtitles</Table.Th>
                                    <Table.Th w={72} />
                                </Table.Tr>
                            </Table.Thead>
                            <Table.Tbody>
                                {loading ? (
                                    <Table.Tr>
                                        <Table.Td colSpan={7}><Text ta="center" py="md">Loading...</Text></Table.Td>
                                    </Table.Tr>
                                ) : filtered.length === 0 ? (
                                    <Table.Tr>
                                        <Table.Td colSpan={7}><Text ta="center" py="md" c="dimmed">No items match the current filter.</Text></Table.Td>
                                    </Table.Tr>
                                ) : (
                                    currentMonitors.map((entry) => (
                                        <Table.Tr key={entry.ID}>
                                            <Table.Td fw={700}>
                                                <Text component={Link} to={`/media/${entry.library_id}`} c="gray" fw={700}>
                                                    {entry.title}
                                                </Text>
                                            </Table.Td>
                                            <Table.Td>
                                                <StatusPill label={entry.is_season ? "Season" : "Episode"} tone={entry.is_season ? "violet" : "blue"} />
                                            </Table.Td>
                                            <Table.Td c="dimmed">
                                                {entry.is_season ? "" : `E${entry.episode_number}: ${entry.episode_title}`}
                                            </Table.Td>
                                            <Table.Td>
                                                <StatusPill label={entry.status} tone={STATUS_TONES[entry.status] ?? "gray"} />
                                            </Table.Td>
                                            <Table.Td c={entry.quality ? undefined : "dimmed"}>
                                                {entry.quality ? entry.quality.toUpperCase() : "—"}
                                            </Table.Td>
                                            <Table.Td c={entry.subtitles ? undefined : "dimmed"}>
                                                {entry.subtitles ? entry.subtitles.split(",").filter(Boolean).map((s) => s.toUpperCase()).join(", ") : "—"}
                                            </Table.Td>
                                            <Table.Td ta="right">
                                                <ActionIcon variant="subtle" color="red" onClick={() => handleDelete(entry.ID)} aria-label="Remove from monitor">
                                                    <IconTrash size={18} />
                                                </ActionIcon>
                                            </Table.Td>
                                        </Table.Tr>
                                    ))
                                )}
                            </Table.Tbody>
                        </Table>
                    </ScrollArea>

                    {totalPages > 1 && (
                        <Group justify="space-between" align="center">
                            <Text size="sm" c="dimmed">
                                Showing {startIndex + 1}–{Math.min(startIndex + itemsPerPage, filtered.length)} of {filtered.length}
                            </Text>
                            <Pagination value={currentPage} onChange={setCurrentPage} total={totalPages} color="gray" />
                        </Group>
                    )}
                </Stack>
            </Card>
        </Stack>
    )
}
