import { useEffect, useState } from "react"
import { Card, Group, SimpleGrid, Stack, Text, Title } from "@mantine/core"
import { API_URL, showToast } from "@/utils"
import { StatusPill } from "@/components"

interface ServiceHealth {
    name: string
    display_name: string
    address: string
    running: boolean
    error?: string
}

export function WorkersPage() {
    const [services, setServices] = useState<ServiceHealth[]>([])
    const [loading, setLoading] = useState(true)

    const fetchWorkers = async () => {
        try {
            const response = await fetch(`${API_URL}/api/workers`)
            if (!response.ok) throw new Error("Failed to fetch worker statuses")
            const data = await response.json()
            setServices(data || [])
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

    return (
        <Stack gap="lg">
            <div>
                <Title order={1}>Workers</Title>
                <Text c="dimmed">Status of background workers</Text>
            </div>

            <SimpleGrid cols={{ base: 1, sm: 2, lg: 3 }} spacing="md">
                {loading ? (
                    <Card withBorder radius="xl" style={{ minHeight: 140 }}>
                        <Text c="dimmed" ta="center" py="xl">Loading service health...</Text>
                    </Card>
                ) : services.length === 0 ? (
                    <Card withBorder radius="xl" style={{ minHeight: 140 }}>
                        <Text c="dimmed" ta="center" py="xl">No service health data available.</Text>
                    </Card>
                ) : (
                    services.map((service) => (
                        <Card key={service.name} withBorder radius="xl">
                            <Group justify="space-between" align="center">
                                <Title order={4}>{service.display_name}</Title>
                                <StatusPill label={service.running ? "Online" : "Offline"} tone={service.running ? "green" : "red"} />
                            </Group>
                            {service.error ? (
                                <Text size="xs" c="red">
                                    {service.error}
                                </Text>
                            ) : null}
                        </Card>
                    ))
                )}
            </SimpleGrid>
        </Stack>
    )
}
