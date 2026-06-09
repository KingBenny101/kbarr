import { useEffect, useMemo, useState } from "react"
import { ActionIcon, Anchor, Button, Card, Checkbox, Group, Modal, Pagination, Progress, ScrollArea, Stack, Table, Text, TextInput, Title, Tooltip, UnstyledButton } from "@mantine/core"
import { IconChevronDown, IconChevronUp, IconExternalLink, IconFlask, IconLink, IconSearch, IconSelector, IconTrash } from "@tabler/icons-react"
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
    progress?: number | null
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

const ALL_STATUSES = ["pending", "downloading", "completed", "failed"] as const
type StatusFilter = typeof ALL_STATUSES[number] | "all"

type SortField = "title" | "size" | "seeders" | "added" | "status"
type SortDir = "asc" | "desc"

const STATUS_PRIORITY: Record<string, number> = {
    downloading: 0,
    pending: 1,
    failed: 2,
    completed: 3,
}

function SortIcon({ field, sortField, sortDir }: { field: SortField; sortField: SortField; sortDir: SortDir }) {
    if (field !== sortField) return <IconSelector size={14} style={{ opacity: 0.4 }} />
    return sortDir === "asc" ? <IconChevronUp size={14} /> : <IconChevronDown size={14} />
}

export function DownloadsPage() {
    const [queue, setQueue] = useState<DownloadItem[]>([])
    const [loading, setLoading] = useState(true)
    const [triggering, setTriggering] = useState(false)
    const [devMode, setDevMode] = useState(false)
    const [deleteTarget, setDeleteTarget] = useState<DownloadItem | null>(null)
    const [blacklist, setBlacklist] = useState(false)
    const [unmonitor, setUnmonitor] = useState(false)
    const [deleteFiles, setDeleteFiles] = useState(false)
    const [deleting, setDeleting] = useState(false)
    const [testModalOpen, setTestModalOpen] = useState(false)
    const [testURL, setTestURL] = useState("")
    const [testTitle, setTestTitle] = useState("")
    const [submitting, setSubmitting] = useState(false)

    const [search, setSearch] = useState("")
    const [activeStatus, setActiveStatus] = useState<StatusFilter>("all")
    const [sortField, setSortField] = useState<SortField>("added")
    const [sortDir, setSortDir] = useState<SortDir>("desc")
    const [currentPage, setCurrentPage] = useState(1)
    const itemsPerPage = 15

    useEffect(() => {
        apiFetch(`${API_URL}/api/settings`)
            .then((r) => r.ok ? r.json() : null)
            .then((data) => { if (data?.devMode === "true") setDevMode(true) })
            .catch(() => {})
    }, [])

    const handleTrigger = async () => {
        setTriggering(true)
        try {
            await apiFetch(`${API_URL}/api/downloads/trigger`, { method: "POST" })
            setTimeout(fetchQueue, 1000)
        } catch {
            showToast("Failed to trigger downloader", "error")
        } finally {
            setTriggering(false)
        }
    }

    const handleAddTest = async () => {
        if (!testURL.trim()) return
        setSubmitting(true)
        try {
            const res = await apiFetch(`${API_URL}/api/downloads/test`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ torrent_url: testURL.trim(), title: testTitle.trim() }),
            })
            if (!res.ok) throw new Error()
            showToast("Test torrent added", "success")
            setTestModalOpen(false)
            setTestURL("")
            setTestTitle("")
            fetchQueue()
        } catch {
            showToast("Failed to add test torrent", "error")
        } finally {
            setSubmitting(false)
        }
    }

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

    const handleRetryHardlinks = async (item: DownloadItem) => {
        try {
            const res = await apiFetch(`${API_URL}/api/downloads/${item.id}/symlink`, { method: "POST" })
            if (!res.ok) throw new Error()
            showToast("Hardlink creation triggered — check logs", "success")
        } catch {
            showToast("Failed to trigger hardlink creation", "error")
        }
    }

    const openDeleteModal = (item: DownloadItem) => {
        setDeleteTarget(item)
        setBlacklist(false)
        setUnmonitor(false)
        setDeleteFiles(item.status === "completed")
    }

    const confirmDelete = async () => {
        if (!deleteTarget) return
        setDeleting(true)
        try {
            const response = await apiFetch(`${API_URL}/api/downloads/${deleteTarget.id}`, {
                method: "DELETE",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ blacklist, unmonitor, deleteFiles }),
            })
            if (!response.ok) throw new Error()
            setQueue((items) => items.filter((item) => item.id !== deleteTarget.id))
            showToast("Removed from queue", "success")
            setDeleteTarget(null)
        } catch {
            showToast("Failed to remove item", "error")
        } finally {
            setDeleting(false)
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
        for (const item of queue) counts[item.status ?? "unknown"] = (counts[item.status ?? "unknown"] ?? 0) + 1
        return counts
    }, [queue])

    const filtered = useMemo(() => {
        const q = search.trim().toLowerCase()
        let list = queue.filter((item) => {
            if (activeStatus !== "all" && item.status !== activeStatus) return false
            if (q) {
                const haystack = `${item.title ?? ""} ${item.torrent_name ?? ""}`.toLowerCase()
                if (!haystack.includes(q)) return false
            }
            return true
        })
        list = [...list].sort((a, b) => {
            let cmp = 0
            if (sortField === "title") cmp = (a.title ?? "").localeCompare(b.title ?? "")
            else if (sortField === "size") cmp = (a.size ?? 0) - (b.size ?? 0)
            else if (sortField === "seeders") cmp = (a.seeders ?? 0) - (b.seeders ?? 0)
            else if (sortField === "added") cmp = new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
            else if (sortField === "status") cmp = (STATUS_PRIORITY[a.status ?? ""] ?? 99) - (STATUS_PRIORITY[b.status ?? ""] ?? 99)
            return sortDir === "asc" ? cmp : -cmp
        })
        return list
    }, [queue, search, activeStatus, sortField, sortDir])

    const totalPages = Math.ceil(filtered.length / itemsPerPage)
    const startIndex = (currentPage - 1) * itemsPerPage
    const visible = filtered.slice(startIndex, startIndex + itemsPerPage)

    return (
        <Stack gap="lg">
            <Modal opened={!!deleteTarget} onClose={() => setDeleteTarget(null)} title="Remove from queue" centered size="sm">
                <Stack gap="md">
                    <Text size="sm">Remove <strong>{deleteTarget?.title ?? "this torrent"}</strong> from the queue?</Text>
                    <Checkbox
                        label="Blacklist this torrent (won't be queued again)"
                        checked={blacklist}
                        onChange={(e) => { setBlacklist(e.currentTarget.checked); if (!e.currentTarget.checked) setUnmonitor(false) }}
                    />
                    {blacklist && (
                        <Checkbox
                            label="Unmonitor (stop looking for this)"
                            checked={unmonitor}
                            onChange={(e) => setUnmonitor(e.currentTarget.checked)}
                        />
                    )}
                    <Checkbox
                        label="Delete downloaded files and hardlinks"
                        checked={deleteFiles}
                        onChange={(e) => setDeleteFiles(e.currentTarget.checked)}
                    />
                    <Group justify="flex-end" gap="xs">
                        <Button variant="subtle" color="gray" onClick={() => setDeleteTarget(null)}>Cancel</Button>
                        <Button color="red" loading={deleting} onClick={confirmDelete}>Remove</Button>
                    </Group>
                </Stack>
            </Modal>

            <Modal opened={testModalOpen} onClose={() => setTestModalOpen(false)} title="Add test torrent" centered>
                <Stack gap="md">
                    <TextInput
                        label="Torrent URL"
                        description="Magnet link or .torrent file URL"
                        placeholder="magnet:?xt=urn:btih:..."
                        value={testURL}
                        onChange={(e) => setTestURL(e.currentTarget.value)}
                        required
                    />
                    <TextInput
                        label="Title"
                        description="Optional label (defaults to URL)"
                        placeholder="My test torrent"
                        value={testTitle}
                        onChange={(e) => setTestTitle(e.currentTarget.value)}
                    />
                    <Button color="gray" fullWidth loading={submitting} disabled={!testURL.trim()} onClick={handleAddTest}>
                        Add to queue
                    </Button>
                </Stack>
            </Modal>

            <Title order={1}>Downloads</Title>

            <Card withBorder radius="xl">
                <Stack gap="md">
                    <Group justify="space-between" align="flex-start">
                        <div>
                            <Title order={3}>Queue status</Title>
                            <Text c="dimmed" size="sm">
                                {filtered.length !== queue.length
                                    ? `${filtered.length} of ${queue.length} items`
                                    : `${queue.length} item${queue.length !== 1 ? "s" : ""}`}
                            </Text>
                        </div>
                        <Group gap="xs" wrap="wrap" justify="flex-end">
                            {devMode && (
                                <Button variant="light" color="orange" size="xs" leftSection={<IconFlask size={14} />} onClick={() => setTestModalOpen(true)}>
                                    Add test torrent
                                </Button>
                            )}
                            <Button variant="light" color="gray" size="xs" loading={triggering} onClick={handleTrigger}>
                                Process now
                            </Button>
                        </Group>
                    </Group>

                    <Group gap="xs" wrap="wrap">
                        <StatusPill
                            label={`All (${queue.length})`}
                            tone={activeStatus === "all" ? "blue" : "gray"}
                            onClick={() => setFilter("all")}
                        />
                        {ALL_STATUSES.filter((s) => statusCounts[s]).map((s) => (
                            <StatusPill
                                key={s}
                                label={`${s} (${statusCounts[s]})`}
                                tone={activeStatus === s ? statusTone(s) : "gray"}
                                onClick={() => setFilter(s)}
                            />
                        ))}
                    </Group>

                    <TextInput
                        placeholder="Search by title or torrent name…"
                        leftSection={<IconSearch size={16} />}
                        value={search}
                        onChange={(e) => { setSearch(e.currentTarget.value); setCurrentPage(1) }}
                    />

                    <ScrollArea type="auto">
                        <Table striped highlightOnHover withTableBorder withColumnBorders verticalSpacing="md">
                            <Table.Thead>
                                <Table.Tr>
                                    <Table.Th>
                                        <UnstyledButton onClick={() => toggleSort("title")} style={{ display: "flex", alignItems: "center", gap: 4 }}>
                                            Media <SortIcon field="title" sortField={sortField} sortDir={sortDir} />
                                        </UnstyledButton>
                                    </Table.Th>
                                    <Table.Th>Torrent name</Table.Th>
                                    <Table.Th>Indexer</Table.Th>
                                    <Table.Th>
                                        <UnstyledButton onClick={() => toggleSort("size")} style={{ display: "flex", alignItems: "center", gap: 4 }}>
                                            Size <SortIcon field="size" sortField={sortField} sortDir={sortDir} />
                                        </UnstyledButton>
                                    </Table.Th>
                                    <Table.Th>
                                        <UnstyledButton onClick={() => toggleSort("seeders")} style={{ display: "flex", alignItems: "center", gap: 4 }}>
                                            Seeders <SortIcon field="seeders" sortField={sortField} sortDir={sortDir} />
                                        </UnstyledButton>
                                    </Table.Th>
                                    <Table.Th>Hash</Table.Th>
                                    <Table.Th>
                                        <UnstyledButton onClick={() => toggleSort("added")} style={{ display: "flex", alignItems: "center", gap: 4 }}>
                                            Added <SortIcon field="added" sortField={sortField} sortDir={sortDir} />
                                        </UnstyledButton>
                                    </Table.Th>
                                    <Table.Th>
                                        <UnstyledButton onClick={() => toggleSort("status")} style={{ display: "flex", alignItems: "center", gap: 4 }}>
                                            Status <SortIcon field="status" sortField={sortField} sortDir={sortDir} />
                                        </UnstyledButton>
                                    </Table.Th>
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
                                ) : filtered.length === 0 ? (
                                    <Table.Tr>
                                        <Table.Td colSpan={9}>
                                            <Text ta="center" py="md" c="dimmed">
                                                {queue.length === 0 ? "Download queue is empty." : "No items match the current filter."}
                                            </Text>
                                        </Table.Td>
                                    </Table.Tr>
                                ) : (
                                    visible.map((item) => (
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
                                                <Stack gap={4}>
                                                    <StatusPill label={item.status ?? "unknown"} tone={statusTone(item.status)} />
                                                    {item.status === "downloading" && (
                                                        <Progress value={(item.progress ?? 0) * 100} size="xs" color="blue" />
                                                    )}
                                                </Stack>
                                            </Table.Td>
                                            <Table.Td ta="right">
                                                <Group gap={4} justify="flex-end" wrap="nowrap">
                                                    {item.status === "completed" && (
                                                        <Tooltip label="Retry hardlink creation">
                                                            <ActionIcon variant="subtle" color="blue" onClick={() => handleRetryHardlinks(item)} aria-label="Retry hardlinks">
                                                                <IconLink size={18} />
                                                            </ActionIcon>
                                                        </Tooltip>
                                                    )}
                                                    <ActionIcon variant="subtle" color="red" onClick={() => openDeleteModal(item)} aria-label="Remove from queue">
                                                        <IconTrash size={18} />
                                                    </ActionIcon>
                                                </Group>
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
