import { useEffect, useMemo, useState } from "react"
import { Navigate, useLocation } from "react-router-dom"
import { Alert, Button, Card, Checkbox, Group, Loader, PasswordInput, Stack, Text, TextInput, Title } from "@mantine/core"
import { API_URL, showToast } from "@/utils"

interface Settings {
    anidbClient: string
    anidbVersion: string
    anidbSyncInterval: string
    prowlarrUrl: string
    prowlarrApiKey: string
    prowlarrInterval: string
    autoMonitorOnAdd: string
    monitorSyncInterval: string
}

export function SettingsPage() {
    const location = useLocation()
    const path = location.pathname.toLowerCase()

    const [settings, setSettings] = useState<Settings | null>(null)
    const [initialSettings, setInitialSettings] = useState<Settings | null>(null)
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)

    useEffect(() => {
        fetchSettings()
    }, [])

    const fetchSettings = async () => {
        setLoading(true)
        try {
            const response = await fetch(`${API_URL}/api/settings`)
            const data = await response.json()
            const nextSettings: Settings = {
                anidbClient: data.anidbClient || "",
                anidbVersion: data.anidbVersion || "",
                anidbSyncInterval: data.anidbSyncInterval || "1440",
                prowlarrUrl: data.prowlarrUrl || "http://localhost:9696",
                prowlarrApiKey: data.prowlarrApiKey || "",
                prowlarrInterval: data.prowlarrInterval || "60",
                autoMonitorOnAdd: data.autoMonitorOnAdd || "false",
                monitorSyncInterval: data.monitorSyncInterval || "1",
            }
            setSettings(nextSettings)
            setInitialSettings(nextSettings)
        } catch (error) {
            console.error("Fetch settings error:", error)
        } finally {
            setLoading(false)
        }
    }

    const isDirty = useMemo(
        () => Boolean(initialSettings && settings && JSON.stringify(initialSettings) !== JSON.stringify(settings)),
        [initialSettings, settings],
    )

    const update = (key: keyof Settings, value: string) => {
        setSettings((prev) => (prev ? { ...prev, [key]: value } : prev))
    }

    const handleSave = async () => {
        if (!settings) return
        setSaving(true)
        try {
            const response = await fetch(`${API_URL}/api/settings`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    ...settings,
                    prowlarrApiKey: settings.prowlarrApiKey.trim() || "error",
                }),
            })
            if (response.ok) {
                showToast("Settings saved", "success")
                await fetchSettings()
            } else {
                showToast("Save failed", "error")
            }
        } catch (error) {
            console.error(error)
            showToast("Error saving", "error")
        } finally {
            setSaving(false)
        }
    }

    if (path === "/settings" || path === "/settings/") {
        return <Navigate to="/settings/general" replace />
    }

    if (loading || !settings) {
        return <Group justify="center" py="xl"><Loader color="gray" /></Group>
    }

    return (
        <Stack gap="lg" pb={80}>
            <Title order={1}>
                {path.includes("metadata") ? "Metadata" : path.includes("indexer") ? "Indexer" : "General"}
            </Title>

            {path.includes("general") ? (
                <Card withBorder radius="xl" p="lg">
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
                            <Checkbox
                                checked={settings.autoMonitorOnAdd === "true"}
                                onChange={(e) => update("autoMonitorOnAdd", e.currentTarget.checked ? "true" : "false")}
                            />
                        </Group>
                        <TextInput
                            label="Monitor sync interval"
                            description="Interval in minutes for adding monitored items to the search queue. Minimum 1 minute."
                            value={settings.monitorSyncInterval}
                            onChange={(e) => { if (e.currentTarget.value === "" || /^[0-9]+$/.test(e.currentTarget.value)) update("monitorSyncInterval", e.currentTarget.value) }}
                            rightSection={<Text size="xs" c="dimmed">min</Text>}
                        />
                    </Stack>
                </Card>
            ) : null}

            {path.includes("metadata") ? (
                <Card withBorder radius="xl" p="lg">
                    <Stack gap="md">
                        <div>
                            <Title order={3}>AniDB</Title>
                            <Text size="sm" c="dimmed">Configure how the service syncs metadata from AniDB.</Text>
                        </div>
                        <TextInput label="Client name" value={settings.anidbClient} onChange={(e) => update("anidbClient", e.currentTarget.value)} placeholder="kbarr" />
                        <TextInput label="Client version" value={settings.anidbVersion} onChange={(e) => update("anidbVersion", e.currentTarget.value)} placeholder="1" />
                        <TextInput
                            label="Sync interval (m)"
                            value={settings.anidbSyncInterval}
                            onChange={(e) => { if (e.currentTarget.value === "" || /^[0-9]+$/.test(e.currentTarget.value)) update("anidbSyncInterval", e.currentTarget.value) }}
                            placeholder="1440"
                        />
                    </Stack>
                </Card>
            ) : null}

            {path.includes("indexer") ? (
                <Card withBorder radius="xl" p="lg">
                    <Stack gap="md">
                        <div>
                            <Title order={3}>Prowlarr</Title>
                            <Text size="sm" c="dimmed">Configure the Prowlarr indexer used for monitoring and searches.</Text>
                        </div>
                        <TextInput label="URL" value={settings.prowlarrUrl} onChange={(e) => update("prowlarrUrl", e.currentTarget.value)} placeholder="http://localhost:9696" />
                        <PasswordInput
                            label="API key"
                            value={settings.prowlarrApiKey}
                            onChange={(e) => update("prowlarrApiKey", e.currentTarget.value)}
                            onFocus={() => { if (settings.prowlarrApiKey === initialSettings?.prowlarrApiKey) update("prowlarrApiKey", "") }}
                            placeholder="api_key_..."
                        />
                        <TextInput
                            label="Scan interval (min)"
                            value={settings.prowlarrInterval}
                            onChange={(e) => { if (e.currentTarget.value === "" || /^[0-9]+$/.test(e.currentTarget.value)) update("prowlarrInterval", e.currentTarget.value) }}
                            placeholder="60"
                        />
                    </Stack>
                </Card>
            ) : null}

            <Group justify="flex-end" style={{ position: "sticky", bottom: 16 }}>
                {isDirty ? (
                    <Alert color="gray" variant="light" style={{ flex: 1, maxWidth: 480 }}>
                        You have unsaved changes.
                    </Alert>
                ) : null}
                <Button color="gray" onClick={handleSave} disabled={!isDirty || saving} loading={saving}>
                    Save changes
                </Button>
            </Group>
        </Stack>
    )
}
