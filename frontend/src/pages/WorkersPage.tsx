import { useEffect, useRef, useState } from "react"
import { Badge, Card, Code, Group, Paper, ScrollArea, Stack, Text, ThemeIcon, Title, UnstyledButton } from "@mantine/core"
import { IconCircleCheck, IconCircleX } from "@tabler/icons-react"
import { API_URL, apiFetch, showToast } from "@/utils"

interface ServiceHealth {
    name: string
    display_name: string
    running: boolean
    error?: string
}

interface LogEntry {
    time: string
    level: string
    msg: string
    attrs?: string
}

function levelColor(level: string): string {
    if (level === "ERROR") return "red"
    if (level === "WARN") return "yellow"
    if (level === "DEBUG") return "gray"
    return "blue"
}

function LogViewer({ service }: { service: ServiceHealth }) {
    const [logs, setLogs] = useState<LogEntry[]>([])
    const [loading, setLoading] = useState(true)
    const viewport = useRef<HTMLDivElement>(null)
    const isAtBottom = useRef(true)

    useEffect(() => {
        setLogs([])
        setLoading(true)
        let cancelled = false

        const fetchLogs = async () => {
            try {
                const res = await apiFetch(`${API_URL}/api/workers/${service.name}/logs`)
                if (!res.ok) throw new Error()
                const data: LogEntry[] = await res.json()
                if (!cancelled) {
                    setLogs(data)
                    if (isAtBottom.current) {
                        setTimeout(() => {
                            if (viewport.current) viewport.current.scrollTop = viewport.current.scrollHeight
                        }, 30)
                    }
                }
            } catch {
                // ignore
            } finally {
                if (!cancelled) setLoading(false)
            }
        }

        fetchLogs()
        const interval = setInterval(fetchLogs, 3000)
        return () => { cancelled = true; clearInterval(interval) }
    }, [service.name])

    const handleScroll = () => {
        if (!viewport.current) return
        const { scrollTop, scrollHeight, clientHeight } = viewport.current
        isAtBottom.current = scrollHeight - scrollTop - clientHeight < 40
    }

    return (
        <Stack gap="xs" h="100%">
            <Group justify="space-between" align="center">
                <Group gap="xs">
                    <ThemeIcon size="sm" radius="xl" color={service.running ? "green" : "red"} variant="light">
                        {service.running ? <IconCircleCheck size={14} /> : <IconCircleX size={14} />}
                    </ThemeIcon>
                    <Title order={4}>{service.display_name}</Title>
                    <Badge size="xs" color={service.running ? "green" : "red"} variant="light">
                        {service.running ? "Online" : "Offline"}
                    </Badge>
                </Group>
                <Text size="xs" c="dimmed">{logs.length} entries</Text>
            </Group>

            {service.error && (
                <Text size="xs" c="red">{service.error}</Text>
            )}

            <Paper withBorder radius="md" p="xs" style={{ flex: 1, minHeight: 0, background: "var(--mantine-color-dark-8, #1a1b1e)", fontFamily: "monospace" }}>
                <ScrollArea h="100%" viewportRef={viewport} onScrollPositionChange={handleScroll} type="scroll">
                    {loading && logs.length === 0 ? (
                        <Text size="xs" c="dimmed" p="sm">Loading logs…</Text>
                    ) : logs.length === 0 ? (
                        <Text size="xs" c="dimmed" p="sm">No log entries yet.</Text>
                    ) : (
                        <Stack gap={1} p="xs">
                            {logs.map((entry, i) => (
                                <Group key={i} gap={8} wrap="nowrap" align="flex-start">
                                    <Text size="xs" c="dimmed" style={{ whiteSpace: "nowrap", flexShrink: 0, fontFamily: "monospace" }}>
                                        {entry.time.slice(11, 19)}
                                    </Text>
                                    <Badge size="xs" color={levelColor(entry.level)} variant="filled" style={{ flexShrink: 0, fontFamily: "monospace", minWidth: 44, textAlign: "center" }}>
                                        {entry.level.slice(0, 4)}
                                    </Badge>
                                    <Code style={{ fontSize: 12, background: "transparent", color: "var(--mantine-color-gray-3, #dee2e6)", whiteSpace: "pre-wrap", wordBreak: "break-all", padding: 0 }}>
                                        {entry.msg}{entry.attrs || ""}
                                    </Code>
                                </Group>
                            ))}
                        </Stack>
                    )}
                </ScrollArea>
            </Paper>
        </Stack>
    )
}

function ServiceListItem({ service, selected, onClick }: { service: ServiceHealth; selected: boolean; onClick: () => void }) {
    return (
        <UnstyledButton onClick={onClick} style={{ width: "100%", borderRadius: "var(--mantine-radius-lg)" }}>
            <Paper
                withBorder={selected}
                radius="lg"
                p="sm"
                style={{
                    background: selected ? "var(--mantine-color-default-hover)" : "transparent",
                    transition: "background 120ms",
                }}
            >
                <Group gap="sm" wrap="nowrap">
                    <ThemeIcon size="sm" radius="xl" color={service.running ? "green" : "red"} variant="light">
                        {service.running ? <IconCircleCheck size={14} /> : <IconCircleX size={14} />}
                    </ThemeIcon>
                    <div style={{ flex: 1, minWidth: 0 }}>
                        <Text size="sm" fw={selected ? 600 : 400} truncate>
                            {service.display_name}
                        </Text>
                        <Text size="xs" c={service.running ? "green" : "red"}>
                            {service.running ? "Online" : "Offline"}
                        </Text>
                    </div>
                </Group>
            </Paper>
        </UnstyledButton>
    )
}

export function WorkersPage() {
    const [services, setServices] = useState<ServiceHealth[]>([])
    const [selected, setSelected] = useState<string>("core")
    const [loading, setLoading] = useState(true)

    const fetchWorkers = async () => {
        try {
            const res = await apiFetch(`${API_URL}/api/workers`)
            if (!res.ok) throw new Error()
            const data: ServiceHealth[] = await res.json()
            setServices(data || [])
        } catch {
            showToast("Failed to load worker statuses", "error")
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        fetchWorkers()
        const interval = setInterval(fetchWorkers, 10000)
        return () => clearInterval(interval)
    }, [])

    const selectedService = services.find((s) => s.name === selected) ?? null

    return (
        <Stack gap="lg" h="calc(100vh - 56px - 32px)">
            <div>
                <Title order={1}>Workers</Title>
                <Text c="dimmed">Status and logs of background services</Text>
            </div>

            <Group align="flex-start" gap="md" style={{ flex: 1, minHeight: 0 }}>
                {/* Service list */}
                <Card withBorder radius="xl" p="sm" style={{ width: 200, flexShrink: 0 }}>
                    <Stack gap={4}>
                        <Text size="xs" tt="uppercase" fw={700} c="dimmed" px="xs" mb={4}>Services</Text>
                        {loading ? (
                            <Text size="xs" c="dimmed" px="xs">Loading…</Text>
                        ) : (
                            services.map((svc) => (
                                <ServiceListItem
                                    key={svc.name}
                                    service={svc}
                                    selected={selected === svc.name}
                                    onClick={() => setSelected(svc.name)}
                                />
                            ))
                        )}
                    </Stack>
                </Card>

                {/* Log panel */}
                <Card withBorder radius="xl" p="md" style={{ flex: 1, minHeight: 0, height: "100%", display: "flex", flexDirection: "column" }}>
                    {selectedService ? (
                        <LogViewer service={selectedService} />
                    ) : (
                        <Text c="dimmed" ta="center" py="xl">Select a service to view logs.</Text>
                    )}
                </Card>
            </Group>
        </Stack>
    )
}
