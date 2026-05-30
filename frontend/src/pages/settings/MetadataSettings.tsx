import { Card, Stack, Text, TextInput, Title } from "@mantine/core"

interface MetadataSettingsProps {
    client: string
    setClient: (val: string) => void
    version: string
    setVersion: (val: string) => void
    interval: string
    setInterval: (val: string) => void
}

export function MetadataSettings({ client, setClient, version, setVersion, interval, setInterval }: MetadataSettingsProps) {
    return (
        <Card withBorder radius="md" p="lg">
            <Stack gap="md">
                <div>
                    <Title order={3}>Metadata settings</Title>
                    <Text size="sm" c="dimmed">Configure how the service syncs metadata (AniDB).</Text>
                </div>

                <TextInput label="Client name" value={client} onChange={(event) => setClient(event.currentTarget.value)} placeholder="kbarr" />
                <TextInput label="Client version" value={version} onChange={(event) => setVersion(event.currentTarget.value)} placeholder="1" />
                <TextInput
                    label="Sync interval (m)"
                    value={interval}
                    onChange={(event) => {
                        const value = event.currentTarget.value
                        if (value === "" || /^[0-9]+$/.test(value)) {
                            setInterval(value)
                        }
                    }}
                    placeholder="1440"
                />
            </Stack>
        </Card>
    )
}
