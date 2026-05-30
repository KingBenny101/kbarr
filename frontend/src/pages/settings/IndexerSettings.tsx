import { Card, PasswordInput, Stack, Text, TextInput, Title } from "@mantine/core"

interface IndexerSettingsProps {
    url: string
    setUrl: (val: string) => void
    apiKey: string
    setApiKey: (val: string) => void
    interval: string
    setInterval: (val: string) => void
    initialApiKey?: string
}

export function IndexerSettings({ url, setUrl, apiKey, setApiKey, interval, setInterval, initialApiKey }: IndexerSettingsProps) {
    return (
        <Card withBorder radius="md" p="lg">
            <Stack gap="md">
                <div>
                    <Title order={3}>Indexer settings</Title>
                    <Text size="sm" c="dimmed">Configure the indexer used for monitoring and searches.</Text>
                </div>

                <TextInput label="URL" value={url} onChange={(event) => setUrl(event.currentTarget.value)} placeholder="http://localhost:9696" />
                <PasswordInput label="API key" value={apiKey} onChange={(event) => setApiKey(event.currentTarget.value)} onFocus={() => { if (apiKey === initialApiKey) setApiKey("") }} placeholder="api_key_..." />
                <TextInput
                    label="Scan interval (min)"
                    value={interval}
                    onChange={(event) => {
                        const value = event.currentTarget.value
                        if (value === "" || /^[0-9]+$/.test(value)) {
                            setInterval(value)
                        }
                    }}
                    placeholder="60"
                />
            </Stack>
        </Card>
    )
}
