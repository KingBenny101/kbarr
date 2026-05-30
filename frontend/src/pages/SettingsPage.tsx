import { useEffect, useMemo, useState } from "react"
import { Navigate, useLocation } from "react-router-dom"
import { Alert, Button, Group, Loader, Stack, Title } from "@mantine/core"
import { API_URL } from "@/lib/api"
import { showToast } from "@/lib/notifications"
import { MetadataSettings } from "./settings/MetadataSettings"
import { GeneralSettings } from "./settings/GeneralSettings"
import { IndexerSettings } from "./settings/IndexerSettings"

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

    const updateSetting = (key: keyof Settings, value: string) => {
        setSettings((previous) => (previous ? { ...previous, [key]: value } : previous))
    }

    const handleSave = async () => {
        if (!settings) return
        setSaving(true)
        try {
            const response = await fetch(`${API_URL}/api/settings`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    anidbClient: settings.anidbClient,
                    anidbVersion: settings.anidbVersion,
                    anidbSyncInterval: settings.anidbSyncInterval,
                    prowlarrUrl: settings.prowlarrUrl,
                    prowlarrApiKey: settings.prowlarrApiKey.trim() || "error",
                    prowlarrInterval: settings.prowlarrInterval,
                    autoMonitorOnAdd: settings.autoMonitorOnAdd,
                    monitorSyncInterval: settings.monitorSyncInterval,
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
            <Stack gap={4}>
                <Title order={1}>Settings</Title>
            </Stack>

            <Stack gap="lg">
                {path.includes("general") ? (
                    <GeneralSettings
                        autoMonitorOnAdd={settings.autoMonitorOnAdd}
                        setAutoMonitorOnAdd={(value) => updateSetting("autoMonitorOnAdd", value)}
                        monitorSyncInterval={settings.monitorSyncInterval}
                        setMonitorSyncInterval={(value) => updateSetting("monitorSyncInterval", value)}
                    />
                ) : null}
                {path.includes("metadata") ? (
                    <MetadataSettings
                        client={settings.anidbClient}
                        setClient={(value) => updateSetting("anidbClient", value)}
                        version={settings.anidbVersion}
                        setVersion={(value) => updateSetting("anidbVersion", value)}
                        interval={settings.anidbSyncInterval}
                        setInterval={(value) => updateSetting("anidbSyncInterval", value)}
                    />
                ) : null}
                {path.includes("indexer") ? (
                    <IndexerSettings
                        url={settings.prowlarrUrl}
                        setUrl={(value) => updateSetting("prowlarrUrl", value)}
                        apiKey={settings.prowlarrApiKey}
                        setApiKey={(value) => updateSetting("prowlarrApiKey", value)}
                        interval={settings.prowlarrInterval}
                        setInterval={(value) => updateSetting("prowlarrInterval", value)}
                        initialApiKey={initialSettings?.prowlarrApiKey}
                    />
                ) : null}
            </Stack>

            <Group justify="flex-end" style={{ position: "sticky", bottom: 16 }}>
                <Alert color={isDirty ? "gray" : "gray"} variant="light" style={{ flex: 1, maxWidth: 480 }}>
                    {isDirty ? "You have unsaved changes." : "No changes pending."}
                </Alert>
                <Button color="gray" onClick={handleSave} disabled={!isDirty || saving} loading={saving}>
                    Save changes
                </Button>
            </Group>
        </Stack>
    )
}