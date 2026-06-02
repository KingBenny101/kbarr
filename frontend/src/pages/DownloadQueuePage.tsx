import { useEffect, useState } from "react"
import { ActionIcon, Anchor, Card, Group, ScrollArea, Stack, Table, Text, Title } from "@mantine/core"
import { IconExternalLink, IconTrash } from "@tabler/icons-react"
import { API_URL, apiFetch, showToast } from "@/utils"
import { StatusPill } from "@/components"

interface DownloadItem {
    id: number
    created_at: string
    monitor_id: number | null
    title: string | null
    torrent_name: string | null
    torrent_url: string | null
    save_path: string | null
    torrent_hash: string | null
    indexer: string | null
    size: number | null
    seeders: number | null
    status: string | null
}

function formatBytes(bytes: number): string {
    if (bytes >= 1024 ** 3) return (bytes / 1024 ** 3).toFixed(1) + " GB"
    if (bytes >= 1024 ** 2) return (bytes / 1024 ** 2).toFixed(1) + " MB"
    return (bytes / 1024).toFixed(0) + " KB"
}

function statusTone(status: string | null): "gray" | "blue" | "green" | "red" | "yellow" {
    switch (status) {
        case "pending": return "yellow"
        case "downloading": return "blue"
        case "completed": return "green"
        case "failed": return "red"
        default: return "gray"
    }
}

export function DownloadQueuePage() {
    const [queue, setQueue] = useState<DownloadItem[]>([])
    const [loading, setLoading] = useState(true)

    const fetchQueue = async () => {
        try {
            const response = await apiFetch(`${API_URL}/api/downloads`)
            if (!response.ok) throw new Error("Failed to fetch download queue")
            const data = await response.json()
            setQueue(data || [])
        } catch (error) {
            console.error(error)
            showToast("Failed to load download queue", "error")
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
            const response = await apiFetch(`${API_URL}/api/downloads/${id}`, { method: "DELETE" })
            if (!response.ok) throw new Error("Failed to delete download entry")
            setQueue((items) => items.filter((item) => item.id !== id))
            showToast("Removed from queue", "success")
        } catch (error) {
            console.error(error)
            showToast("Failed to remove item", "error")
        }
    }

    const pending = queue.filter((i) => i.status === "pending").length
    const downloading = queue.filter((i) => i.status === "downloading").length
    const completed = queue.filter((i) => i.status === "completed").length

    return (
        <Stack gap="lg">
            <Title order={1}>Download queue</Title>

            <Card withBorder radius="xl">
                <Stack gap="md">
                    <Group justify="space-between">
                        <Title order={3}>Queue status</Title>
                        <Group gap="xs">
                            {downloading > 0 && <StatusPill label={`${downloading} downloading`} tone="blue" />}
                            {pending > 0 && <StatusPill label={`${pending} pending`} tone="yellow" />}
                            {completed > 0 && <StatusPill label={`${completed} completed`} tone="green" />}
                            {queue.length === 0 && <StatusPill label="Empty" tone="gray" />}
                        </Group>
                    </Group>
                    <ScrollArea type="auto">
                        <Table striped highlightOnHover withTableBorder withColumnBorders verticalSpacing="md">
                            <Table.Thead>
                                <Table.Tr>
                                    <Table.Th>Media</Table.Th>
                                    <Table.Th>Torrent name</Table.Th>
                                    <Table.Th>Indexer</Table.Th>
                                    <Table.Th>Size</Table.Th>
                                    <Table.Th>Seeders</Table.Th>
                                    <Table.Th>Hash</Table.Th>
                                    <Table.Th>Added</Table.Th>
                                    <Table.Th>Status</Table.Th>
                                    <Table.Th w={72} />
                                </Table.Tr>
                            </Table.Thead>
                            <Table.Tbody>
                                {loading ? (
                                    <Table.Tr>
                                        <Table.Td colSpan={9}>
                                            <Text ta="center" py="md">Loading queue...</Text>
                                        </Table.Td>
                                    </Table.Tr>
                                ) : queue.length === 0 ? (
                                    <Table.Tr>
                                        <Table.Td colSpan={9}>
                                            <Text ta="center" py="md" c="dimmed">Download queue is empty.</Text>
                                        </Table.Td>
                                    </Table.Tr>
                                ) : (
                                    queue.map((item) => (
                                        <Table.Tr key={item.id}>
                                            <Table.Td fw={700} style={{ whiteSpace: "nowrap" }}>{item.title ?? "—"}</Table.Td>
                                            <Table.Td>
                                                {item.torrent_url ? (
                                                    <Anchor href={item.torrent_url} target="_blank" rel="noreferrer" c="gray">
                                                        <Group gap={4} wrap="nowrap">
                                                            <IconExternalLink size={14} style={{ flexShrink: 0 }} />
                                                            <Text size="sm" style={{ wordBreak: "break-all" }}>{item.torrent_name ?? "Link"}</Text>
                                                        </Group>
                                                    </Anchor>
                                                ) : <Text c="dimmed" size="sm">{item.torrent_name ?? "—"}</Text>}
                                            </Table.Td>
                                            <Table.Td style={{ whiteSpace: "nowrap" }}>
                                                <Text size="sm">{item.indexer ?? "—"}</Text>
                                            </Table.Td>
                                            <Table.Td style={{ whiteSpace: "nowrap" }}>
                                                <Text size="sm" c="dimmed">{item.size ? formatBytes(item.size) : "—"}</Text>
                                            </Table.Td>
                                            <Table.Td ta="center">
                                                <Text size="sm">{item.seeders ?? "—"}</Text>
                                            </Table.Td>
                                            <Table.Td>
                                                <Text size="sm" c="dimmed" style={{ fontFamily: "monospace" }}>
                                                    {item.torrent_hash ? item.torrent_hash.slice(0, 12) + "…" : "—"}
                                                </Text>
                                            </Table.Td>
                                            <Table.Td c="dimmed" style={{ whiteSpace: "nowrap" }}>
                                                {new Date(item.created_at).toLocaleString()}
                                            </Table.Td>
                                            <Table.Td>
                                                <StatusPill label={item.status ?? "unknown"} tone={statusTone(item.status)} />
                                            </Table.Td>
                                            <Table.Td ta="right">
                                                <ActionIcon variant="subtle" color="red" onClick={() => handleDelete(item.id)} aria-label="Remove from queue">
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
