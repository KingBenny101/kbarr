import { useEffect, useState } from "react"
import { ActionIcon, Group, ScrollArea, Stack, Table, Text, Title } from "@mantine/core"
import { IconExternalLink, IconTrash } from "@tabler/icons-react"
import { Link } from "react-router-dom"
import { API_URL } from "@/lib/api"
import { showToast } from "@/lib/notifications"
import { SectionCard } from "@/components/SectionCard"
import { StatusPill } from "@/components/StatusPill"

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
    anidb_id: string
    status: string
}

export function MonitorPage() {
    const [monitors, setMonitors] = useState<MonitorEntry[]>([])
    const [loading, setLoading] = useState(true)
    const [currentPage, setCurrentPage] = useState(1)
    const itemsPerPage = 10

    const fetchMonitors = async () => {
        try {
            const response = await fetch(`${API_URL}/api/monitor`)
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

    useEffect(() => {
        fetchMonitors()
    }, [])

    const handleDelete = async (id: number) => {
        try {
            const response = await fetch(`${API_URL}/api/monitor/${id}`, { method: "DELETE" })
            if (!response.ok) throw new Error("Failed to delete monitor entry")
            setMonitors((items) => items.filter((item) => item.ID !== id))
            showToast("Removed from monitor", "success")
        } catch (error) {
            console.error(error)
            showToast("Failed to remove item", "error")
        }
    }

    const totalPages = Math.ceil(monitors.length / itemsPerPage)
    const startIndex = (currentPage - 1) * itemsPerPage
    const currentMonitors = monitors.slice(startIndex, startIndex + itemsPerPage)

    return (
        <Stack gap="lg">
            <Stack gap={4}>
                <Title order={1}>Monitored items</Title>
            </Stack>

            <SectionCard withBorder radius="xl">
                <Stack gap="md">
                    <Group justify="space-between" align="center">
                        <div>
                            <Title order={3}>Currently monitoring</Title>
                            <Text c="dimmed" size="sm">
                                You have {monitors.length} item{monitors.length !== 1 ? "s" : ""} being monitored.
                            </Text>
                        </div>
                        <StatusPill label={`${monitors.length} total`} tone="gray" />
                    </Group>

                    <ScrollArea type="auto">
                        <Table striped highlightOnHover withTableBorder withColumnBorders verticalSpacing="md">
                            <Table.Thead>
                                <Table.Tr>
                                    <Table.Th>Anime</Table.Th>
                                    <Table.Th>Type</Table.Th>
                                    <Table.Th>Details</Table.Th>
                                    <Table.Th>Status</Table.Th>
                                    <Table.Th>AniDB</Table.Th>
                                    <Table.Th w={72} />
                                </Table.Tr>
                            </Table.Thead>
                            <Table.Tbody>
                                {loading ? (
                                    <Table.Tr>
                                        <Table.Td colSpan={6}>
                                            <Text ta="center" py="md">Loading...</Text>
                                        </Table.Td>
                                    </Table.Tr>
                                ) : monitors.length === 0 ? (
                                    <Table.Tr>
                                        <Table.Td colSpan={6}>
                                            <Text ta="center" py="md" c="dimmed">No items monitored yet.</Text>
                                        </Table.Td>
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
                                                <StatusPill label={entry.status} tone="gray" />
                                            </Table.Td>
                                            <Table.Td>
                                                {entry.anidb_id ? (
                                                    <ActionIcon component="a" href={entry.is_season ? `https://anidb.net/anime/${entry.anidb_id}` : `https://anidb.net/episode/${entry.anidb_id}`} target="_blank" rel="noreferrer" variant="subtle" color="gray" aria-label="Open AniDB entry">
                                                        <IconExternalLink size={18} />
                                                    </ActionIcon>
                                                ) : null}
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

                    {totalPages > 1 ? (
                        <Group justify="space-between">
                            <Text size="sm" c="dimmed">
                                Showing {startIndex + 1} to {Math.min(startIndex + itemsPerPage, monitors.length)} of {monitors.length} items
                            </Text>
                            <Group gap="xs">
                                <ActionIcon variant="light" onClick={() => setCurrentPage((previous) => Math.max(1, previous - 1))} disabled={currentPage === 1} aria-label="Previous page">
                                    <IconExternalLink size={16} style={{ transform: "rotate(180deg)" }} />
                                </ActionIcon>
                                <Text size="sm" fw={700}>
                                    {currentPage} / {totalPages}
                                </Text>
                                <ActionIcon variant="light" onClick={() => setCurrentPage((previous) => Math.min(totalPages, previous + 1))} disabled={currentPage === totalPages} aria-label="Next page">
                                    <IconExternalLink size={16} />
                                </ActionIcon>
                            </Group>
                        </Group>
                    ) : null}
                </Stack>
            </SectionCard>
        </Stack>
    )
}