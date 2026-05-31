import { useEffect, useState } from "react"
import { ActionIcon, Card, Group, ScrollArea, Stack, Table, Text, Title } from "@mantine/core"
import { IconInfoCircle, IconTrash } from "@tabler/icons-react"
import { Link } from "react-router-dom"
import { API_URL, showToast } from "@/utils"
import { StatusPill } from "@/components"

interface QueueItem {
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
}

export function SearchQueuePage() {
    const [queue, setQueue] = useState<QueueItem[]>([])
    const [loading, setLoading] = useState(true)

    const fetchQueue = async () => {
        try {
            const response = await fetch(`${API_URL}/api/search-queue`)
            if (!response.ok) throw new Error("Failed to fetch search queue")
            const data = await response.json()
            setQueue(data || [])
        } catch (error) {
            console.error(error)
            showToast("Failed to load search queue", "error")
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        fetchQueue()
        const interval = setInterval(fetchQueue, 5000)
        return () => clearInterval(interval)
    }, [])

    const handleDelete = async (id: number) => {
        try {
            const response = await fetch(`${API_URL}/api/search-queue/${id}`, { method: "DELETE" })
            if (!response.ok) throw new Error("Failed to delete queue entry")
            setQueue((items) => items.filter((item) => item.ID !== id))
            showToast("Removed from queue", "success")
        } catch (error) {
            console.error(error)
            showToast("Failed to remove item", "error")
        }
    }

    return (
        <Stack gap="lg">
            <Title order={1}>Search queue</Title>

            <Card withBorder radius="xl">
                <Stack gap="md">
                    <Group justify="space-between">
                        <Title order={3}>Queue status</Title>
                        <StatusPill label={`${queue.length} waiting`} tone="gray" />
                    </Group>
                    <ScrollArea type="auto">
                        <Table striped highlightOnHover withTableBorder withColumnBorders verticalSpacing="md">
                            <Table.Thead>
                                <Table.Tr>
                                    <Table.Th>Anime</Table.Th>
                                    <Table.Th>Type</Table.Th>
                                    <Table.Th>Details</Table.Th>
                                    <Table.Th>Status</Table.Th>
                                    <Table.Th w={72} />
                                </Table.Tr>
                            </Table.Thead>
                            <Table.Tbody>
                                {loading ? (
                                    <Table.Tr>
                                        <Table.Td colSpan={5}>
                                            <Text ta="center" py="md">Loading queue...</Text>
                                        </Table.Td>
                                    </Table.Tr>
                                ) : queue.length === 0 ? (
                                    <Table.Tr>
                                        <Table.Td colSpan={5}>
                                            <Text ta="center" py="md" c="dimmed">Search queue is empty.</Text>
                                        </Table.Td>
                                    </Table.Tr>
                                ) : (
                                    queue.map((item) => (
                                        <Table.Tr key={item.ID}>
                                            <Table.Td fw={700}>
                                                <Text component={Link} to={`/media/${item.library_id}`} c="gray" fw={700}>
                                                    {item.title}
                                                </Text>
                                            </Table.Td>
                                            <Table.Td>
                                                <StatusPill label={item.is_season ? "Season" : "Episode"} tone={item.is_season ? "violet" : "blue"} />
                                            </Table.Td>
                                            <Table.Td c="dimmed">
                                                {item.is_season ? "" : `E${item.episode_number}: ${item.episode_title}`}
                                            </Table.Td>
                                            <Table.Td>
                                                <StatusPill label={item.status} tone="gray" />
                                            </Table.Td>
                                            <Table.Td ta="right">
                                                <ActionIcon variant="subtle" color="red" onClick={() => handleDelete(item.ID)} aria-label="Remove from queue">
                                                    <IconTrash size={18} />
                                                </ActionIcon>
                                            </Table.Td>
                                        </Table.Tr>
                                    ))
                                )}
                            </Table.Tbody>
                        </Table>
                    </ScrollArea>
                </Stack>
            </Card>

       
        </Stack>
    )
}
