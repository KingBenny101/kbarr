import { useEffect, useMemo, useState } from "react"
import { Badge, Group, Paper, Stack, Table, Text, Title, Tooltip } from "@mantine/core"
import { API_URL, apiFetch } from "@/utils"
import { usePolling } from "@/hooks"
import { formatDuration, formatRelative, formatWallClock } from "@/lib/format"

interface CycleStatus {
    service: string
    cycle: string
    display_name: string
    state: "idle" | "running"
    last_started_at: string | null
    last_finished_at: string | null
    last_duration_ms: number
    next_run_at: string | null
}

interface ServiceHealth {
    name: string
    running: boolean
}

function statePill(state: string, offline: boolean): React.ReactNode {
    if (offline) return <Badge color="red" variant="light">Offline</Badge>
    if (state === "running") return <Badge color="yellow" variant="light">Running now</Badge>
    return <Badge color="gray" variant="light">Idle</Badge>
}

function timeCell(ts: string | null, running: boolean, now: Date): React.ReactNode {
    if (!ts) return <Text c="dimmed">never</Text>
    const date = new Date(ts)
    return (
        <Tooltip label={formatWallClock(date)}>
            <Text>{running ? "started " + formatRelative(date, now) : formatRelative(date, now)}</Text>
        </Tooltip>
    )
}

export default function SystemPage() {
    const [cycles, setCycles] = useState<CycleStatus[]>([])
    const [workers, setWorkers] = useState<ServiceHealth[]>([])
    const [now, setNow] = useState(() => new Date())
    const [stale, setStale] = useState(false)

    useEffect(() => {
        const timer = setInterval(() => setNow(new Date()), 1000)
        return () => clearInterval(timer)
    }, [])

    const fetchCycles = async (): Promise<boolean> => {
        try {
            const res = await apiFetch(`${API_URL}/api/cycles`)
            if (!res.ok) throw new Error()
            const data: { cycles: CycleStatus[] } = await res.json()
            setCycles(data.cycles)
            setStale(false)
            return true
        } catch {
            setStale(true)
            return false
        }
    }

    const fetchWorkers = async (): Promise<boolean> => {
        try {
            const res = await apiFetch(`${API_URL}/api/workers`)
            if (!res.ok) throw new Error()
            const data: ServiceHealth[] = await res.json()
            setWorkers(data)
            return true
        } catch {
            return false
        }
    }

    usePolling(fetchCycles, { interval: 15_000 }, [])
    usePolling(fetchWorkers, { interval: 15_000 }, [])

    const offlineServices = useMemo(
        () => new Set(workers.filter((w) => !w.running).map((w) => w.name)),
        [workers],
    )

    return (
        <Stack gap="md">
            <Group justify="space-between">
                <Title order={2}>System</Title>
                {stale && <Text c="orange" size="sm">Status data is stale — retrying…</Text>}
            </Group>
            <Paper withBorder p="md">
                <Table striped highlightOnHover>
                    <Table.Thead>
                        <Table.Tr>
                            <Table.Th>Cycle</Table.Th>
                            <Table.Th>State</Table.Th>
                            <Table.Th>Last run</Table.Th>
                            <Table.Th>Next run</Table.Th>
                            <Table.Th>Duration</Table.Th>
                        </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                        {cycles.map((c) => {
                            const offline = offlineServices.has(c.service)
                            const running = !offline && c.state === "running"
                            const lastTs = running ? c.last_started_at : c.last_finished_at ?? c.last_started_at
                            return (
                                <Table.Tr key={`${c.service}/${c.cycle}`}>
                                    <Table.Td>
                                        <Group gap="xs">
                                            <Text fw={500}>{c.display_name}</Text>
                                            <Badge size="xs" variant="outline" color="gray">{c.service}</Badge>
                                        </Group>
                                    </Table.Td>
                                    <Table.Td>{statePill(c.state, offline)}</Table.Td>
                                    <Table.Td>{timeCell(lastTs, running, now)}</Table.Td>
                                    <Table.Td>
                                        {running ? (
                                            <Text c="dimmed">—</Text>
                                        ) : (
                                            timeCell(c.next_run_at, false, now)
                                        )}
                                    </Table.Td>
                                    <Table.Td>{formatDuration(c.last_duration_ms)}</Table.Td>
                                </Table.Tr>
                            )
                        })}
                        {cycles.length === 0 && (
                            <Table.Tr>
                                <Table.Td colSpan={5}><Text c="dimmed" ta="center">No cycles recorded yet</Text></Table.Td>
                            </Table.Tr>
                        )}
                    </Table.Tbody>
                </Table>
            </Paper>
        </Stack>
    )
}

export { SystemPage }
