import { Card, Checkbox, Group, Stack, Text, TextInput, Title } from "@mantine/core"

interface GeneralSettingsProps {
    autoMonitorOnAdd: string
    setAutoMonitorOnAdd: (value: string) => void
    monitorSyncInterval: string
    setMonitorSyncInterval: (value: string) => void
}

export function GeneralSettings({ autoMonitorOnAdd, setAutoMonitorOnAdd, monitorSyncInterval, setMonitorSyncInterval }: GeneralSettingsProps) {
    const isEnabled = autoMonitorOnAdd === "true"

    return (
        <Card withBorder radius="md" p="lg">
            <Stack gap="md">
                <div>
                    <Title order={3}>General settings</Title>
                    <Text size="sm" c="dimmed">Configure the application-wide behavior.</Text>
                </div>

                <Group justify="space-between" align="start">
                    <div>
                        <Text fw={700}>Auto-monitor on media add</Text>
                        <Text size="sm" c="dimmed">Automatically mark newly added media as monitored and trigger search.</Text>
                    </div>
                    <Checkbox checked={isEnabled} onChange={(event) => setAutoMonitorOnAdd(event.currentTarget.checked ? "true" : "false")} />
                </Group>

                <TextInput
                    label="Monitor sync interval"
                    description="Interval in minutes for adding monitored items to the search queue. Minimum 1 minute."
                    value={monitorSyncInterval}
                    onChange={(event) => {
                        const value = event.currentTarget.value
                        if (value === "" || /^[0-9]+$/.test(value)) {
                            setMonitorSyncInterval(value)
                        }
                    }}
                    rightSection={<Text size="xs" c="dimmed">min</Text>}
                />
            </Stack>
        </Card>
    )
}