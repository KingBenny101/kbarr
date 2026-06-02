import { useEffect, useMemo, useState } from "react"
import { Navigate, useLocation } from "react-router-dom"
import { Alert, Button, Card, Checkbox, Divider, Group, Loader, PasswordInput, Select, Stack, Text, TextInput, Title } from "@mantine/core"
import { API_URL, apiFetch, clearToken, showToast } from "@/utils"

interface Settings {
    anidbClient: string
    anidbVersion: string
    anidbSyncInterval: string
    prowlarrUrl: string
    prowlarrApiKey: string
    prowlarrInterval: string
    preferredQuality: string
    minSeeders: string
    autoMonitorOnAdd: string
    monitorSyncInterval: string
    qbittorrentUrl: string
    qbittorrentUsername: string
    qbittorrentPassword: string
}

interface CredentialForm {
    currentPassword: string
    newUsername: string
    newPassword: string
    confirmPassword: string
}

export function SettingsPage() {
    const location = useLocation()
    const path = location.pathname.toLowerCase()

    const [settings, setSettings] = useState<Settings | null>(null)
    const [initialSettings, setInitialSettings] = useState<Settings | null>(null)
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [testingIndexer, setTestingIndexer] = useState(false)
    const [testingDownloader, setTestingDownloader] = useState(false)

    const [creds, setCreds] = useState<CredentialForm>({ currentPassword: "", newUsername: "", newPassword: "", confirmPassword: "" })
    const [savingCreds, setSavingCreds] = useState(false)

    useEffect(() => { fetchSettings() }, [])

    const fetchSettings = async () => {
        setLoading(true)
        try {
            const response = await apiFetch(`${API_URL}/api/settings`)
            const data = await response.json()
            const nextSettings: Settings = {
                anidbClient: data.anidbClient || "",
                anidbVersion: data.anidbVersion || "",
                anidbSyncInterval: data.anidbSyncInterval || "1440",
                prowlarrUrl: data.prowlarrUrl || "http://localhost:9696",
                prowlarrApiKey: data.prowlarrApiKey || "",
                prowlarrInterval: data.prowlarrInterval || "60",
                preferredQuality: data.preferredQuality || "1080p",
                minSeeders: data.minSeeders || "1",
                autoMonitorOnAdd: data.autoMonitorOnAdd || "false",
                monitorSyncInterval: data.monitorSyncInterval || "1",
                qbittorrentUrl: data.qbittorrentUrl || "http://localhost:8080",
                qbittorrentUsername: data.qbittorrentUsername || "",
                qbittorrentPassword: data.qbittorrentPassword || "",
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
            const response = await apiFetch(`${API_URL}/api/settings`, {
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

    const handleTestIndexer = async () => {
        setTestingIndexer(true)
        try {
            const res = await apiFetch(`${API_URL}/api/settings/test/indexer`, { method: "POST" })
            const data = await res.json()
            showToast(data.message, data.ok === "true" ? "success" : "error")
        } catch { showToast("Test failed", "error") }
        finally { setTestingIndexer(false) }
    }

    const handleTestDownloader = async () => {
        setTestingDownloader(true)
        try {
            const res = await apiFetch(`${API_URL}/api/settings/test/downloader`, { method: "POST" })
            const data = await res.json()
            showToast(data.message, data.ok === "true" ? "success" : "error")
        } catch { showToast("Test failed", "error") }
        finally { setTestingDownloader(false) }
    }

    const handleSaveCreds = async () => {
        if (!creds.currentPassword) {
            showToast("Current password is required", "error")
            return
        }
        if (creds.newPassword && creds.newPassword !== creds.confirmPassword) {
            showToast("New passwords do not match", "error")
            return
        }
        setSavingCreds(true)
        try {
            const res = await apiFetch(`${API_URL}/api/auth/credentials`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    currentPassword: creds.currentPassword,
                    newUsername: creds.newUsername,
                    newPassword: creds.newPassword,
                }),
            })
            if (res.ok) {
                showToast("Credentials updated, signing out…", "success")
                await apiFetch(`${API_URL}/api/auth/logout`, { method: "POST" })
                clearToken()
                window.location.href = "/login"
            } else if (res.status === 401) {
                showToast("Current password is incorrect", "error")
            } else {
                showToast("Failed to update credentials", "error")
            }
        } catch {
            showToast("Error updating credentials", "error")
        } finally {
            setSavingCreds(false)
        }
    }

    if (path === "/settings" || path === "/settings/") {
        return <Navigate to="/settings/general" replace />
    }

    const isDownloader = path.includes("downloader")

    if (loading || !settings) {
        return <Group justify="center" py="xl"><Loader color="gray" /></Group>
    }

    return (
        <Stack gap="lg" pb={80}>
            <Title order={1}>
                {path.includes("metadata") ? "Metadata" : path.includes("indexer") ? "Indexer" : isDownloader ? "Downloader" : "General"}
            </Title>

            {path.includes("general") ? (
                <>
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
                            <Select
                                label="Preferred quality"
                                description="Torrent results matching this quality score higher."
                                value={settings.preferredQuality}
                                onChange={(v) => update("preferredQuality", v ?? "1080p")}
                                data={[
                                    { value: "4K", label: "4K" },
                                    { value: "1080p", label: "1080p" },
                                    { value: "720p", label: "720p" },
                                    { value: "480p", label: "480p" },
                                    { value: "any", label: "Any" },
                                ]}
                            />
                            <TextInput
                                label="Minimum seeders"
                                description="Torrents with fewer seeders than this are ignored."
                                value={settings.minSeeders}
                                onChange={(e) => { if (e.currentTarget.value === "" || /^[0-9]+$/.test(e.currentTarget.value)) update("minSeeders", e.currentTarget.value) }}
                                placeholder="1"
                            />
                        </Stack>
                    </Card>

                    <Card withBorder radius="xl" p="lg">
                        <Stack gap="md">
                            <div>
                                <Title order={3}>Account</Title>
                                <Text size="sm" c="dimmed">Change your login username or password. Current password is always required.</Text>
                            </div>
                            <Divider />
                            <TextInput
                                label="New username"
                                placeholder="Leave blank to keep current"
                                value={creds.newUsername}
                                onChange={(e) => { const v = e.currentTarget.value; setCreds((c) => ({ ...c, newUsername: v })) }}
                            />
                            <PasswordInput
                                label="New password"
                                placeholder="Leave blank to keep current"
                                value={creds.newPassword}
                                onChange={(e) => { const v = e.currentTarget.value; setCreds((c) => ({ ...c, newPassword: v })) }}
                            />
                            <PasswordInput
                                label="Confirm new password"
                                placeholder="••••••••"
                                value={creds.confirmPassword}
                                onChange={(e) => { const v = e.currentTarget.value; setCreds((c) => ({ ...c, confirmPassword: v })) }}
                            />
                            <Divider />
                            <PasswordInput
                                label="Current password"
                                placeholder="Required to save changes"
                                value={creds.currentPassword}
                                onChange={(e) => { const v = e.currentTarget.value; setCreds((c) => ({ ...c, currentPassword: v })) }}
                            />
                            <Group justify="flex-end">
                                <Button
                                    color="gray"
                                    onClick={handleSaveCreds}
                                    loading={savingCreds}
                                    disabled={!creds.currentPassword || (!creds.newUsername && !creds.newPassword)}
                                >
                                    Update credentials
                                </Button>
                            </Group>
                        </Stack>
                    </Card>
                </>
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
                        <Group justify="flex-end">
                            <Button variant="light" color="gray" loading={testingIndexer} onClick={handleTestIndexer}>
                                Test connection
                            </Button>
                        </Group>
                    </Stack>
                </Card>
            ) : null}

            {isDownloader ? (
                <Card withBorder radius="xl" p="lg">
                    <Stack gap="md">
                        <div>
                            <Title order={3}>qBittorrent</Title>
                            <Text size="sm" c="dimmed">Configure the qBittorrent client used for downloading torrents.</Text>
                        </div>
                        <TextInput
                            label="URL"
                            value={settings.qbittorrentUrl}
                            onChange={(e) => update("qbittorrentUrl", e.currentTarget.value)}
                            placeholder="http://localhost:8080"
                        />
                        <TextInput
                            label="Username"
                            value={settings.qbittorrentUsername}
                            onChange={(e) => update("qbittorrentUsername", e.currentTarget.value)}
                            placeholder="admin"
                        />
                        <PasswordInput
                            label="Password"
                            value={settings.qbittorrentPassword}
                            onChange={(e) => update("qbittorrentPassword", e.currentTarget.value)}
                            onFocus={() => { if (settings.qbittorrentPassword === initialSettings?.qbittorrentPassword) update("qbittorrentPassword", "") }}
                            placeholder="••••••••"
                        />
                        <Group justify="flex-end">
                            <Button variant="light" color="gray" loading={testingDownloader} onClick={handleTestDownloader}>
                                Test connection
                            </Button>
                        </Group>
                    </Stack>
                </Card>
            ) : null}

            {!isDownloader && !path.includes("metadata") && !path.includes("indexer") ? null : (
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
            )}

            {path.includes("general") && isDirty ? (
                <Group justify="flex-end" style={{ position: "sticky", bottom: 16 }}>
                    <Alert color="gray" variant="light" style={{ flex: 1, maxWidth: 480 }}>
                        You have unsaved changes.
                    </Alert>
                    <Button color="gray" onClick={handleSave} disabled={saving} loading={saving}>
                        Save changes
                    </Button>
                </Group>
            ) : null}
        </Stack>
    )
}
