import { useMemo, useState } from "react"
import { Badge, Box, Group, Paper, SimpleGrid, Stack, Table, Text, Title } from "@mantine/core"
import { useMediaQuery } from "@mantine/hooks"
import { API_URL, apiFetch } from "@/utils"
import { usePolling } from "@/hooks"
import { formatDuration, formatRelativeWords, formatTimeAgoWords, ringProgress, formatWallClock } from "@/lib/format"

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

const MONO = "'JetBrains Mono', var(--mantine-font-family-monospace)"
const MISSING_RETRY_MIN = 1440

function stateColor(state: "idle" | "running", offline: boolean): string {
    if (offline) return "var(--mantine-color-red-6)"
    return state === "running" ? "var(--mantine-color-yellow-6)" : "var(--mantine-color-blue-6)"
}

function timeCell(ts: string | null, running: boolean, now: Date, isFuture = false): React.ReactNode {
    if (!ts) return <Text c="dimmed" ff={MONO}>—</Text>
    const date = new Date(ts)
    const wall = formatWallClock(date)
    const dateStr = date.toISOString().slice(0, 10) // YYYY-MM-DD
    const relative = running
        ? "started " + formatRelativeWords(date, now)
        : isFuture
            ? formatRelativeWords(date, now)
            : formatTimeAgoWords(date, now)
    return (
        <Text ff={MONO} fw={500} fz="xs">
            <span>{dateStr} {wall}</span> <span style={{ opacity: 0.5, marginLeft: 4 }}>({relative})</span>
        </Text>
    )
}

interface CycleView {
    offline: boolean
    running: boolean
    lastTs: string | null
    progress: number | null
    stateColor: string
}

function cycleView(c: CycleStatus, offlineServices: Set<string>, now: Date): CycleView {
    const offline = offlineServices.has(c.service)
    const running = !offline && c.state === "running"
    const lastTs = running ? c.last_started_at : c.last_finished_at ?? c.last_started_at
    return {
        offline,
        running,
        lastTs,
        progress: running ? null : ringProgress(now, c.last_finished_at, c.next_run_at),
        stateColor: stateColor(c.state, offline),
    }
}

function CycleRing({ running, offline, progress }: { running: boolean; offline: boolean; progress: number | null }) {
    const size = 18
    const stroke = 2.5
    const r = (size - stroke) / 2
    const c = 2 * Math.PI * r
    return (
        <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} aria-hidden className={running && !offline ? "kbarr-ring-pulse" : undefined}>
            <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="var(--mantine-color-default-border)" strokeWidth={stroke} />
            {offline ? (
                <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="var(--mantine-color-red-6)" strokeWidth={stroke} />
            ) : running ? (
                <circle cx={size / 2} cy={size / 2} r={r * 0.4} fill="var(--mantine-color-yellow-6)" />
            ) : (
                <circle
                    cx={size / 2}
                    cy={size / 2}
                    r={r}
                    fill="none"
                    stroke="var(--mantine-color-yellow-6)"
                    strokeWidth={stroke}
                    strokeLinecap="round"
                    strokeDasharray={c}
                    strokeDashoffset={c * (1 - (progress ?? 0))}
                    transform={`rotate(-90 ${size / 2} ${size / 2})`}
                />
            )}
        </svg>
    )
}

function CycleTable({ cycles, offlineServices, now }: { cycles: CycleStatus[]; offlineServices: Set<string>; now: Date }) {
    return (
        <Table striped highlightOnHover>
            <Table.Thead>
                <Table.Tr>
                    <Table.Th style={{ width: "200px", minWidth: "160px" }}>Cycle</Table.Th>
                    <Table.Th style={{ width: "140px" }}>Last run</Table.Th>
                    <Table.Th style={{ width: "140px" }}>Next run</Table.Th>
                    <Table.Th style={{ width: "120px" }}>Duration</Table.Th>
                </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
                {cycles.map((c) => {
                    const v = cycleView(c, offlineServices, now)
                    return (
                        <Table.Tr
                            key={`${c.service}/${c.cycle}`}
                            style={{
                                borderLeft: `3px solid ${v.stateColor}`,
                                transition: "border-color 0.2s ease",
                            }}
                        >
                            <Table.Td style={{ paddingLeft: "md" }}>
                                <Group gap="xs">
                                    <CycleRing running={v.running} offline={v.offline} progress={v.progress} />
                                    <Text fw={500} fz="sm">{c.display_name}</Text>
                                    <Badge size="xs" variant="outline" color="gray">{c.service}</Badge>
                                </Group>
                            </Table.Td>
                            <Table.Td ff={MONO}>{timeCell(v.lastTs, v.running, now)}</Table.Td>
                            <Table.Td ff={MONO}>
                                {v.running ? (
                                    <Text c="dimmed">—</Text>
                                ) : (
                                    timeCell(c.next_run_at, false, now, true)
                                )}
                            </Table.Td>
                            <Table.Td ff={MONO} fw={500}>{formatDuration(c.last_duration_ms)}</Table.Td>
                        </Table.Tr>
                    )
                })}
                {cycles.length === 0 && (
                    <Table.Tr>
                        <Table.Td colSpan={4} style={{ textAlign: "center", padding: "xl" }}>
                            <Text c="dimmed">No cycles recorded yet</Text>
                        </Table.Td>
                    </Table.Tr>
                )}
            </Table.Tbody>
        </Table>
    )
}

function CycleCards({ cycles, offlineServices, now }: { cycles: CycleStatus[]; offlineServices: Set<string>; now: Date }) {
    return (
        <>
            {cycles.map((c, i) => {
                const v = cycleView(c, offlineServices, now)
                return (
                    <Box
                        key={`${c.service}/${c.cycle}`}
                        px="md"
                        py="md"
                        style={{
                            borderTop: i > 0 ? "1px solid var(--mantine-color-default-border)" : undefined,
                            borderLeft: `3px solid ${v.stateColor}`,
                            background: "linear-gradient(90deg, transparent 99%, var(--mantine-color-default-border) 100%)",
                        }}
                    >
                        <Group justify="space-between" gap="sm" wrap="nowrap" mb="xs">
                            <Group gap="xs" wrap="nowrap" miw={0}>
                                <CycleRing running={v.running} offline={v.offline} progress={v.progress} />
                                <Text fw={500} fz="sm" truncate>{c.display_name}</Text>
                            </Group>
                            <Badge size="xs" variant="outline" color="gray" visibleFrom="sm">{c.service}</Badge>
                        </Group>
                        <SimpleGrid cols={3} spacing="xs">
                            <Box>
                                <Text size="xs" c="dimmed" mb="2">Last run</Text>
                                <Box ff={MONO} fz="sm" fw={500}>{timeCell(v.lastTs, v.running, now)}</Box>
                            </Box>
                            <Box>
                                <Text size="xs" c="dimmed" mb="2">Next run</Text>
                                <Box ff={MONO} fz="sm" fw={500}>
                                    {v.running ? (
                                        <Text c="dimmed">—</Text>
                                    ) : (
                                        timeCell(c.next_run_at, false, now, true)
                                    )}
                                </Box>
                            </Box>
                            <Box>
                                <Text size="xs" c="dimmed" mb="2">Duration</Text>
                                <Box ff={MONO} fz="sm" fw={500}>{formatDuration(c.last_duration_ms)}</Box>
                            </Box>
                        </SimpleGrid>
                    </Box>
                )
            })}
            {cycles.length === 0 && (
                <Text c="dimmed" ta="center" p="md">No cycles recorded yet</Text>
            )}
        </>
    )
}

export default function SystemPage() {
    const [cycles, setCycles] = useState<CycleStatus[]>([])
    const [workers, setWorkers] = useState<ServiceHealth[]>([])
    const now = useMemo(() => new Date(), [])
    const [stale, setStale] = useState(false)
    const isDesktop = useMediaQuery("(min-width: 62em)")

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

    // Override process_missing cycle's next_run_at with client-side calculation
    // based on missingRetryInterval (default 1440 min).
    const allCycles = useMemo(() => {
        return cycles.map((c) => {
            if (c.service === "indexer" && c.cycle === "process_missing" && c.last_finished_at) {
                const nextRun = new Date(new Date(c.last_finished_at).getTime() + MISSING_RETRY_MIN * 60_000)
                return { ...c, next_run_at: nextRun.toISOString() }
            }
            return c
        })
    }, [cycles])

    return (
        <>
            <style>{`@media (prefers-reduced-motion: no-preference) { @keyframes kbarr-ring-pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.35; } } .kbarr-ring-pulse { animation: kbarr-ring-pulse 1.6s ease-in-out infinite; } }`}</style>
            <Stack gap="md">
                <Group justify="space-between">
                    <Title order={2} fw={600}>System</Title>
                    {stale && <Text c="orange" size="sm">Status data is stale — retrying…</Text>}
                </Group>
                <Group justify="space-between" mb="xs">
                    <Text c="dimmed" size="sm" ff={MONO}>Current server time: {formatWallClock(now)}</Text>
                </Group>
                <Paper withBorder p={isDesktop ? "md" : 0} style={{ borderColor: "var(--mantine-color-default-border)" }}>
                    {isDesktop ? (
                        <CycleTable cycles={allCycles} offlineServices={offlineServices} now={now} />
                    ) : (
                        <CycleCards cycles={allCycles} offlineServices={offlineServices} now={now} />
                    )}
                </Paper>
            </Stack>
        </>
    )
}

export { SystemPage }