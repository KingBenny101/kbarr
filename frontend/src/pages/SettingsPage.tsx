import { useEffect, useMemo, useState } from "react"
import { Navigate, useLocation } from "react-router-dom"
import { Alert, Button, Card, Checkbox, Divider, Group, Loader, Modal, PasswordInput, Select, Stack, Text, TextInput, Title } from "@mantine/core"
import { IconAlertTriangle } from "@tabler/icons-react"
import { API_URL, apiFetch, clearToken, showToast } from "@/utils"

interface Settings {
    anidbClient: string
    anidbVersion: string
    anidbSyncInterval: string
    prowlarrUrl: string
    prowlarrApiKey: string
    prowlarrInterval: string
    prowlarrCacheAge: string
    matchThreshold: string
    cacheFileLimit: string
    preferredQuality: string
    minSeeders: string
    autoMonitorOnAdd: string
    monitorSyncInterval: string
    qbittorrentUrl: string
    qbittorrentUsername: string
    qbittorrentPassword: string
    downloadPath: string
    mediaPath: string
    allowedVideoExtensions: string
    downloaderInterval: string
    stallTimeout: string
    devMode: string
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
    const [clearBlacklistModal, setClearBlacklistModal] = useState(false)
    const [clearingBlacklist, setClearingBlacklist] = useState(false)

    const handleClearBlacklist = async () => {
        setClearingBlacklist(true)
        try {
            const res = await apiFetch(`${API_URL}/api/downloads/blacklist`, { method: "DELETE" })
            if (!res.ok) throw new Error()
            showToast("Blacklist cleared", "success")
            setClearBlacklistModal(false)
        } catch {
            showToast("Failed to clear blacklist", "error")
        } finally {
            setClearingBlacklist(false)
        }
    }

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
                prowlarrUrl: data.prowlarrUrl || "http://host.docker.internal:9696",
                prowlarrApiKey: data.prowlarrApiKey || "",
                prowlarrInterval: data.prowlarrInterval || "1",
                prowlarrCacheAge: data.prowlarrCacheAge || "3600",
                matchThreshold: data.matchThreshold || "80",
                cacheFileLimit: data.cacheFileLimit || "10",
                preferredQuality: data.preferredQuality || "1080p",
                minSeeders: data.minSeeders || "1",
                autoMonitorOnAdd: data.autoMonitorOnAdd || "false",
                monitorSyncInterval: data.monitorSyncInterval || "1",
                qbittorrentUrl: data.qbittorrentUrl || "http://host.docker.internal:8080",
                qbittorrentUsername: data.qbittorrentUsername || "",
                qbittorrentPassword: data.qbittorrentPassword || "",
                downloadPath: data.downloadPath || "",
                mediaPath: data.mediaPath || "",
                allowedVideoExtensions: data.allowedVideoExtensions || ".mkv,.mp4,.avi,.mov,.wmv,.m4v",
                downloaderInterval: data.downloaderInterval || "1",
                stallTimeout: data.stallTimeout || "300",
                devMode: data.devMode || "false",
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
                    <Card withBorder radius="xl" p="lg" style={{ borderColor: "var(--mantine-color-orange-5)" }}>
                        <Stack gap="md">
                            <Group gap="xs">
                                <IconAlertTriangle size={18} color="var(--mantine-color-orange-5)" />
                                <Title order={3} c="orange">Developer options</Title>
                            </Group>
                            <Text size="sm" c="dimmed">These options are intended for development and testing only.</Text>
                            <Group justify="space-between" align="start">
                                <div>
                                    <Text fw={700}>Dev mode</Text>
                                    <Text size="sm" c="dimmed">Enables test torrent insertion on the Download Queue page.</Text>
                                </div>
                                <Checkbox
                                    checked={settings.devMode === "true"}
                                    onChange={(e) => update("devMode", e.currentTarget.checked ? "true" : "false")}
                                />
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
                <>
                <Card withBorder radius="xl" p="lg">
                    <Stack gap="md">
                        <div>
                            <Title order={3}>Indexer</Title>
                            <Text size="sm" c="dimmed">Configure the indexer service behavior.</Text>
                        </div>
                        <TextInput
                            label="Scan interval (sec)"
                            description="How often the monitor table is polled for new items to search."
                            value={settings.prowlarrInterval}
                            onChange={(e) => { if (e.currentTarget.value === "" || /^[0-9]+$/.test(e.currentTarget.value)) update("prowlarrInterval", e.currentTarget.value) }}
                            placeholder="1"
                            rightSection={<Text size="xs" c="dimmed">sec</Text>}
                        />
                        <TextInput
                            label="Title match threshold"
                            description="Minimum similarity (0–100) between the guessit-parsed torrent title and the anime title. Lower = more permissive."
                            value={settings.matchThreshold}
                            onChange={(e) => { if (e.currentTarget.value === "" || /^[0-9]+$/.test(e.currentTarget.value)) update("matchThreshold", e.currentTarget.value) }}
                            placeholder="80"
                            rightSection={<Text size="xs" c="dimmed">%</Text>}
                        />
                        <TextInput
                            label="Cache file limit"
                            description="Maximum files kept in each cache/debug folder (prowlarr-cache, guessit-debug, matching-debug). Oldest deleted first."
                            value={settings.cacheFileLimit}
                            onChange={(e) => { if (e.currentTarget.value === "" || /^[0-9]+$/.test(e.currentTarget.value)) update("cacheFileLimit", e.currentTarget.value) }}
                            placeholder="10"
                        />
                    </Stack>
                </Card>
                <Card withBorder radius="xl" p="lg">
                    <Stack gap="md">
                        <div>
                            <Title order={3}>Prowlarr</Title>
                            <Text size="sm" c="dimmed">Configure the Prowlarr indexer used for monitoring and searches.</Text>
                        </div>
                        <TextInput label="URL" value={settings.prowlarrUrl} onChange={(e) => update("prowlarrUrl", e.currentTarget.value)} placeholder="http://host.docker.internal:9696" />
                        <PasswordInput
                            label="API key"
                            value={settings.prowlarrApiKey}
                            onChange={(e) => update("prowlarrApiKey", e.currentTarget.value)}
                            onFocus={() => { if (settings.prowlarrApiKey === initialSettings?.prowlarrApiKey) update("prowlarrApiKey", "") }}
                            placeholder="api_key_..."
                        />
                        <TextInput
                            label="Result cache age (sec)"
                            description="How long Prowlarr search results are cached on disk before re-querying. Set to 0 to disable."
                            value={settings.prowlarrCacheAge}
                            onChange={(e) => { if (e.currentTarget.value === "" || /^[0-9]+$/.test(e.currentTarget.value)) update("prowlarrCacheAge", e.currentTarget.value) }}
                            placeholder="3600"
                            rightSection={<Text size="xs" c="dimmed">sec</Text>}
                        />
                        <Group justify="flex-end">
                            <Button variant="light" color="gray" loading={testingIndexer} onClick={handleTestIndexer}>
                                Test connection
                            </Button>
                        </Group>
                    </Stack>
                </Card>
                </>
            ) : null}

            {isDownloader ? (
                <>
                <Card withBorder radius="xl" p="lg">
                    <Stack gap="md">
                        <div>
                            <Title order={3}>Downloader</Title>
                            <Text size="sm" c="dimmed">Configure the downloader service behavior.</Text>
                        </div>
                        <TextInput
                            label="Poll interval (sec)"
                            description="How often the downloader checks for pending items and updates progress."
                            value={settings.downloaderInterval}
                            onChange={(e) => { if (e.currentTarget.value === "" || /^[0-9]+$/.test(e.currentTarget.value)) update("downloaderInterval", e.currentTarget.value) }}
                            placeholder="1"
                            rightSection={<Text size="xs" c="dimmed">sec</Text>}
                        />
                        <TextInput
                            label="Download path"
                            description="Absolute path where qBittorrent saves files. Must match DOWNLOAD_DIR_HOST in your .env."
                            value={settings.downloadPath}
                            onChange={(e) => update("downloadPath", e.currentTarget.value)}
                            placeholder="/home/user/Downloads"
                        />
                        <TextInput
                            label="Media path"
                            description="Absolute path where named symlinks are created. Must match MEDIA_DIR_HOST in your .env."
                            value={settings.mediaPath}
                            onChange={(e) => update("mediaPath", e.currentTarget.value)}
                            placeholder="/home/user/Media/Anime"
                        />
                        <TextInput
                            label="Allowed video extensions"
                            description="Comma-separated list of file extensions to symlink on completion."
                            value={settings.allowedVideoExtensions}
                            onChange={(e) => update("allowedVideoExtensions", e.currentTarget.value)}
                            placeholder=".mkv,.mp4,.avi,.mov,.wmv,.m4v"
                        />
                        <TextInput
                            label="Stall timeout (sec)"
                            description="Remove torrents with no progress for this many seconds and re-queue. Set to 0 to disable."
                            value={settings.stallTimeout}
                            onChange={(e) => { if (e.currentTarget.value === "" || /^[0-9]+$/.test(e.currentTarget.value)) update("stallTimeout", e.currentTarget.value) }}
                            placeholder="300"
                            rightSection={<Text size="xs" c="dimmed">sec</Text>}
                        />
                    </Stack>
                </Card>
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
                            placeholder="http://host.docker.internal:8080"
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

                <Modal opened={clearBlacklistModal} onClose={() => setClearBlacklistModal(false)} title="Clear blacklist" centered size="sm">
                    <Stack gap="md">
                        <Group gap="xs">
                            <IconAlertTriangle size={18} color="var(--mantine-color-orange-5)" />
                            <Text fw={600}>This will remove all blacklisted torrents.</Text>
                        </Group>
                        <Text size="sm" c="dimmed">The indexer will be able to queue these torrents again on the next search cycle.</Text>
                        <Group justify="flex-end" gap="xs">
                            <Button variant="subtle" color="gray" onClick={() => setClearBlacklistModal(false)}>Cancel</Button>
                            <Button color="red" loading={clearingBlacklist} onClick={handleClearBlacklist}>Clear blacklist</Button>
                        </Group>
                    </Stack>
                </Modal>

                <Card withBorder radius="xl" p="lg">
                    <Stack gap="md">
                        <div>
                            <Title order={3}>Blacklist</Title>
                            <Text size="sm" c="dimmed">Torrents are blacklisted automatically when stalled or manually when removed from the queue.</Text>
                        </div>
                        <Group justify="space-between" align="center">
                            <Text size="sm">Clear all blacklisted torrents to allow them to be queued again.</Text>
                            <Button variant="light" color="red" onClick={() => setClearBlacklistModal(true)}>
                                Clear blacklist
                            </Button>
                        </Group>
                    </Stack>
                </Card>
                </>
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
