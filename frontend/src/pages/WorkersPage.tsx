import { useEffect, useState } from "react"
import { Badge, Group, SimpleGrid, Stack, Text, Title } from "@mantine/core"
import { IconCalendar, IconClock } from "@tabler/icons-react"
import { API_URL } from "@/lib/api"
import { showToast } from "@/lib/notifications"
import { SectionCard } from "@/components/SectionCard"

interface WorkerStatus {
    name: string
    last_run: string
    next_run: string
    running: boolean
}

export function WorkersPage() {
    const [workers, setWorkers] = useState<WorkerStatus[]>([])
    const [loading, setLoading] = useState(true)

    const fetchWorkers = async () => {
        try {
            const response = await fetch(`${API_URL}/api/workers`)
            if (!response.ok) throw new Error("Failed to fetch worker statuses")
            const data = await response.json()
            setWorkers(data || [])
        } catch (error) {
            console.error(error)
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

    const formatDate = (dateStr: string) => {
        if (!dateStr || dateStr.startsWith("0001-01-01")) return "Never"
        return new Date(dateStr).toLocaleString()
    }

    const getWorkerDisplayName = (name: string) => {
        switch (name) {
            case "anidb":
                return "AniDB Sync"
            default:
                return name
        }
    }

    return (
        <Stack gap="lg">
            <Stack gap={4}>
                <Text size="sm" c="dimmed" tt="uppercase" fw={700}>
                    System
                </Text>
                <Title order={1}>Background workers</Title>
                <Text c="dimmed" maw={720}>
                    Live status for recurring jobs that keep AniDB sync and monitoring moving.
                </Text>
            </Stack>

            <SimpleGrid cols={{ base: 1, sm: 2, lg: 3 }} spacing="md">
                {loading ? (
                    <SectionCard withBorder radius="xl" style={{ minHeight: 140 }}>
                        <Text c="dimmed" ta="center" py="xl">Loading worker status...</Text>
                    </SectionCard>
                ) : workers.length === 0 ? (
                    <SectionCard withBorder radius="xl" style={{ minHeight: 140 }}>
                        <Text c="dimmed" ta="center" py="xl">No workers active.</Text>
                    </SectionCard>
                ) : (
                    workers.map((worker) => (
                        <SectionCard key={worker.name} withBorder radius="xl">
                            <Group justify="space-between" align="start" mb="md">
                                <Title order={4}>{getWorkerDisplayName(worker.name)}</Title>
                                <Badge color={worker.running ? "green" : "gray"} variant="light">
                                    {worker.running ? "Active" : "Stopped"}
                                </Badge>
                            </Group>

                            <Stack gap="sm">
                                <Group gap="sm" align="start">
                                    <IconClock size={18} style={{ marginTop: 3, color: "var(--mantine-color-dimmed)" }} />
                                    <div>
                                        <Text size="xs" c="dimmed">
                                            Last Run
                                        </Text>
                                        <Text fw={600}>{formatDate(worker.last_run)}</Text>
                                    </div>
                                </Group>

                                <Group gap="sm" align="start">
                                    <IconCalendar size={18} style={{ marginTop: 3, color: "var(--mantine-color-dimmed)" }} />
                                    <div>
                                        <Text size="xs" c="dimmed">
                                            Next Run
                                        </Text>
                                        <Text fw={600}>{formatDate(worker.next_run)}</Text>
                                    </div>
                                </Group>
                            </Stack>
                        </SectionCard>
                    ))
                )}
            </SimpleGrid>
        </Stack>
    )
}