import { useCallback, useEffect, useMemo, useState, type DragEvent } from "react"
import type { ReactNode } from "react"
import { Navigate, useLocation } from "react-router"
import { Alert, Badge, Button, Card, Checkbox, Divider, Group, Loader, Modal, PasswordInput, SegmentedControl, Select, Stack, Switch, Text, TextInput, Title } from "@mantine/core"
import { IconAlertTriangle, IconInfoCircle } from "@tabler/icons-react"
import { API_URL, apiFetch, clearToken, showToast } from "@/utils"

interface SettingDef {
    key: string
    type: "string" | "password" | "bool" | "int" | "select"
    default: string
    label: string
    description?: string
    group: string
    section: string
    options?: string[]
    unit?: string
    widget?: string
    advanced?: boolean
}

const DEV_OPTIONS_KEY = "kbarr-developer-options"

interface CredentialForm {
    currentPassword: string
    newUsername: string
    newPassword: string
    confirmPassword: string
}

const RELEASE_GROUPS = [
    "SubsPlease", "Erai-raws", "ASW", "Judas", "EMBER",
    "Yameii", "Golumpa", "Nekomoe kissaten", "ToxicRUS", "Baws", "HorribleSubs",
]

const SUBTITLE_LANGUAGES = ["multi", "eng", "jpn", "spa", "por", "ara", "fre", "ger"]

const SUBTITLE_LABELS: Record<string, string> = {
    multi: "Multiple subtitles",
    eng: "English",
    jpn: "Japanese",
    spa: "Spanish",
    por: "Portuguese",
    ara: "Arabic",
    fre: "French",
    ger: "German",
}

function SubtitleReorder({
    value,
    onChange,
}: {
    value: string
    onChange: (v: string) => void
}) {
    const [dragIndex, setDragIndex] = useState<number | null>(null)
    const [overIndex, setOverIndex] = useState<number | null>(null)

    const order = useMemo(() => {
        return value ? value.split(",").map((s) => s.trim()).filter(Boolean) : []
    }, [value])

    const available = useMemo(() => SUBTITLE_LANGUAGES.filter((g) => !order.includes(g)), [order])

    const moveItem = (from: number, to: number) => {
        const next = [...order]
        const [moved] = next.splice(from, 1)
        next.splice(to, 0, moved)
        onChange(next.join(","))
    }

    const remove = (index: number) => {
        const next = [...order]
        next.splice(index, 1)
        onChange(next.join(","))
    }

    const add = (lang: string) => {
        onChange([...order, lang].join(","))
    }

    const handleDragStart = (index: number) => {
        setDragIndex(index)
    }

    const handleDragOver = (e: DragEvent, index: number) => {
        e.preventDefault()
        setOverIndex(index)
    }

    const handleDrop = (index: number) => {
        if (dragIndex !== null && dragIndex !== index) {
            moveItem(dragIndex, index)
        }
        setDragIndex(null)
        setOverIndex(null)
    }

    const handleDragEnd = () => {
        setDragIndex(null)
        setOverIndex(null)
    }

    return (
        <div>
            {order.length === 0 && (
                <Text size="sm" c="dimmed" fs="italic" mb={8}>
                    No preferred languages configured. Normal ranking applies.
                </Text>
            )}
            <Stack gap={4}>
                {order.map((lang, i) => (
                    <Group
                        key={lang}
                        draggable
                        justify="space-between"
                        p="xs"
                        style={{
                            borderRadius: 4,
                            border: "1px solid var(--mantine-color-gray-3)",
                            cursor: "grab",
                            opacity: dragIndex === i ? 0.4 : 1,
                            borderColor: overIndex === i ? "var(--mantine-color-blue-5)" : "var(--mantine-color-gray-3)",
                            transition: "border-color 0.15s",
                        }}
                        onDragStart={() => handleDragStart(i)}
                        onDragOver={(e) => handleDragOver(e, i)}
                        onDrop={() => handleDrop(i)}
                        onDragEnd={handleDragEnd}
                    >
                        <Group gap="xs">
                            <Text size="sm" c="dimmed" w={20}>
                                {i + 1}.
                            </Text>
                            <Text size="sm">{SUBTITLE_LABELS[lang] ?? lang}</Text>
                        </Group>
                        <Button size="compact-xs" variant="subtle" color="red" onClick={() => remove(i)}>
                            ✕
                        </Button>
                    </Group>
                ))}
            </Stack>
            {available.length > 0 && (
                <>
                    <Text size="xs" c="dimmed" mt={8} mb={4}>
                        Add language:
                    </Text>
                    <Group gap={4}>
                        {available.map((lang) => (
                            <Badge
                                key={lang}
                                variant="outline"
                                style={{ cursor: "pointer" }}
                                onClick={() => add(lang)}
                            >
                                + {SUBTITLE_LABELS[lang] ?? lang}
                            </Badge>
                        ))}
                    </Group>
                </>
            )}
        </div>
    )
}

function ReleaseGroupReorder({
    value,
    onChange,
}: {
    value: string
    onChange: (v: string) => void
}) {
    const [dragIndex, setDragIndex] = useState<number | null>(null)
    const [overIndex, setOverIndex] = useState<number | null>(null)

    const order = useMemo(() => {
        return value ? value.split(",").map((s) => s.trim()).filter(Boolean) : []
    }, [value])

    const available = useMemo(() => RELEASE_GROUPS.filter((g) => !order.includes(g)), [order])

    const moveItem = (from: number, to: number) => {
        const next = [...order]
        const [moved] = next.splice(from, 1)
        next.splice(to, 0, moved)
        onChange(next.join(","))
    }

    const remove = (index: number) => {
        const next = [...order]
        next.splice(index, 1)
        onChange(next.join(","))
    }

    const add = (group: string) => {
        onChange([...order, group].join(","))
    }

    return (
        <div>
            {order.length === 0 && (
                <Text size="sm" c="dimmed" fs="italic" mb={8}>
                    No preferred groups configured. Normal ranking applies.
                </Text>
            )}
            <Stack gap={4}>
                {order.map((group, i) => (
                    <Group
                        key={group}
                        draggable
                        justify="space-between"
                        p="xs"
                        style={{
                            borderRadius: 4,
                            border: "1px solid var(--mantine-color-gray-3)",
                            cursor: "grab",
                            opacity: dragIndex === i ? 0.4 : 1,
                            borderColor: overIndex === i ? "var(--mantine-color-blue-5)" : "var(--mantine-color-gray-3)",
                            transition: "border-color 0.15s",
                        }}
                        onDragStart={() => setDragIndex(i)}
                        onDragOver={(e) => { e.preventDefault(); setOverIndex(i) }}
                        onDrop={() => {
                            if (dragIndex !== null && dragIndex !== i) moveItem(dragIndex, i)
                            setDragIndex(null); setOverIndex(null)
                        }}
                        onDragEnd={() => { setDragIndex(null); setOverIndex(null) }}
                    >
                        <Group gap="xs">
                            <Text size="sm" c="dimmed" w={20}>
                                {i + 1}.
                            </Text>
                            <Text size="sm">{group}</Text>
                        </Group>
                        <Button size="compact-xs" variant="subtle" color="red" onClick={() => remove(i)}>
                            ✕
                        </Button>
                    </Group>
                ))}
            </Stack>
            {available.length > 0 && (
                <>
                    <Text size="xs" c="dimmed" mt={8} mb={4}>
                        Add group:
                    </Text>
                    <Group gap={4}>
                        {available.map((group) => (
                            <Badge
                                key={group}
                                variant="outline"
                                style={{ cursor: "pointer" }}
                                onClick={() => add(group)}
                            >
                                + {group}
                            </Badge>
                        ))}
                    </Group>
                </>
            )}
        </div>
    )
}

const SECTION_DESCRIPTIONS: Record<string, string> = {
    "General settings": "Configure the application-wide behavior.",
    "Developer options": "These options are intended for development and testing only.",
    "AniDB": "Configure how the service syncs metadata from AniDB.",
    "Indexer": "Configure the indexer service behavior.",
    "kbdex": "AniDB-aware torrent search. Uses the AniDB series ID to resolve titles and filter results natively.",
    "Prowlarr": "Configure the Prowlarr indexer used for monitoring and searches.",
    "Downloader": "Configure the downloader service behavior.",
    "qBittorrent": "Configure the qBittorrent client used for downloading torrents.",
}

function SettingField({
    def,
    value,
    initialValue,
    onChange,
}: {
    def: SettingDef
    value: string
    initialValue: string
    onChange: (v: string) => void
}) {
    if (def.widget === "reorder") {
        return (
            <div>
                <Text size="sm" fw={500} mb={4}>
                    {def.label}
                </Text>
                {def.description && (
                    <Text size="sm" c="dimmed" mb={8}>
                        {def.description}
                    </Text>
                )}
                {def.key === "preferredSubtitleLanguage" ? (
                    <SubtitleReorder value={value} onChange={onChange} />
                ) : (
                    <ReleaseGroupReorder value={value} onChange={onChange} />
                )}
            </div>
        )
    }

    switch (def.type) {
        case "bool":
            return (
                <Group justify="space-between" align="start">
                    <div>
                        <Text fw={700}>{def.label}</Text>
                        {def.description && <Text size="sm" c="dimmed">{def.description}</Text>}
                    </div>
                    <Checkbox
                        checked={value === "true"}
                        onChange={(e) => onChange(e.currentTarget.checked ? "true" : "false")}
                    />
                </Group>
            )
        case "password":
            return (
                <PasswordInput
                    label={def.label}
                    description={def.description}
                    value={value}
                    onChange={(e) => onChange(e.currentTarget.value)}
                    onFocus={() => { if (value === initialValue) onChange("") }}
                    placeholder="••••••••"
                />
            )
        case "select":
            if (def.widget === "segmented") {
                return (
                    <div>
                        <Text size="sm" fw={500} mb={4}>{def.label}</Text>
                        <SegmentedControl
                            value={value}
                            onChange={onChange}
                            data={(def.options ?? []).map((o) => ({ label: o, value: o }))}
                        />
                    </div>
                )
            }
            return (
                <Select
                    label={def.label}
                    description={def.description}
                    value={value}
                    onChange={(v) => onChange(v ?? def.default)}
                    data={(def.options ?? []).map((o) => ({ value: o, label: o }))}
                />
            )
        case "int":
            return (
                <TextInput
                    label={def.label}
                    description={def.description}
                    value={value}
                    placeholder={def.default}
                    onChange={(e) => {
                        if (e.currentTarget.value === "" || /^[0-9]+$/.test(e.currentTarget.value))
                            onChange(e.currentTarget.value)
                    }}
                    rightSection={def.unit ? <Text size="xs" c="dimmed">{def.unit}</Text> : undefined}
                />
            )
        default:
            return (
                <TextInput
                    label={def.label}
                    description={def.description}
                    value={value}
                    placeholder={def.default || undefined}
                    onChange={(e) => onChange(e.currentTarget.value)}
                />
            )
    }
}

function SectionCard({
    section,
    defs,
    values,
    initialValues,
    onUpdate,
    devOptions,
    extra,
}: {
    section: string
    defs: SettingDef[]
    values: Record<string, string>
    initialValues: Record<string, string>
    onUpdate: (key: string, value: string) => void
    devOptions: boolean
    extra?: ReactNode
}) {
    const isDeveloper = section === "Developer options"
    const description = SECTION_DESCRIPTIONS[section]
    const visible = devOptions ? defs : defs.filter((d) => !d.advanced)
    // Don't render an empty card when every setting in the section is advanced
    // and the developer toggle is off (and there's no extra content to show).
    if (visible.length === 0 && !extra) return null
    return (
        <Card
            withBorder
            radius="xl"
            p="lg"
            style={isDeveloper ? { borderColor: "var(--mantine-color-orange-5)" } : undefined}
        >
            <Stack gap="md">
                <div>
                    {isDeveloper ? (
                        <Group gap="xs">
                            <IconAlertTriangle size={18} color="var(--mantine-color-orange-5)" />
                            <Title order={3} c="orange">{section}</Title>
                        </Group>
                    ) : (
                        <Title order={3}>{section}</Title>
                    )}
                    {description && <Text size="sm" c="dimmed">{description}</Text>}
                </div>
                {visible.map((def) => (
                    <div key={def.key}>
                        {def.advanced && (
                            <Badge size="xs" variant="light" color="orange" mb={4}>advanced</Badge>
                        )}
                        <SettingField
                            def={def}
                            value={values[def.key] ?? def.default}
                            initialValue={initialValues[def.key] ?? def.default}
                            onChange={(v) => onUpdate(def.key, v)}
                        />
                    </div>
                ))}
                {extra}
            </Stack>
        </Card>
    )
}

function sectionsForGroup(schema: SettingDef[], group: string): string[] {
    const seen = new Set<string>()
    const order: string[] = []
    for (const def of schema) {
        if (def.group === group && !seen.has(def.section)) {
            seen.add(def.section)
            order.push(def.section)
        }
    }
    return order
}

function defsForSection(schema: SettingDef[], group: string, section: string): SettingDef[] {
    return schema.filter((d) => d.group === group && d.section === section)
}

export function SettingsPage() {
    const location = useLocation()
    const path = location.pathname.toLowerCase()

    const [schema, setSchema] = useState<SettingDef[]>([])
    const [settings, setSettings] = useState<Record<string, string> | null>(null)
    const [initialSettings, setInitialSettings] = useState<Record<string, string> | null>(null)
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState(false)
    const [testingIndexer, setTestingIndexer] = useState(false)
    const [testingKbdex, setTestingKbdex] = useState(false)
    const [testingDownloader, setTestingDownloader] = useState(false)
    const [testingJellyfin, setTestingJellyfin] = useState(false)
    const [clearBlacklistModal, setClearBlacklistModal] = useState(false)
    const [clearingBlacklist, setClearingBlacklist] = useState(false)
    const [devOptions, setDevOptions] = useState<boolean>(() => localStorage.getItem(DEV_OPTIONS_KEY) === "true")

    const toggleDevOptions = (v: boolean) => {
        setDevOptions(v)
        localStorage.setItem(DEV_OPTIONS_KEY, String(v))
    }

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
    const [credsEnvManaged, setCredsEnvManaged] = useState(false)

    useEffect(() => {
        apiFetch(`${API_URL}/api/auth/me`)
            .then((res) => (res.ok ? res.json() : null))
            .then((data) => { if (data) setCredsEnvManaged(Boolean(data.envManaged)) })
            .catch(() => {})
    }, [])

    const fetchSettings = useCallback(async () => {
        const [schemaRes, valuesRes] = await Promise.all([
            apiFetch(`${API_URL}/api/settings/schema`),
            apiFetch(`${API_URL}/api/settings`),
        ])
        if (!schemaRes.ok || !valuesRes.ok) throw new Error("Failed to load settings")
        const schemaDefs: SettingDef[] = await schemaRes.json()
        const data: Record<string, string> = await valuesRes.json()

        // Seed all keys with schema defaults, then overwrite with actual DB values.
        const next: Record<string, string> = {}
        for (const def of schemaDefs) {
            next[def.key] = data[def.key] ?? def.default
        }

        setSchema(schemaDefs)
        setSettings(next)
        setInitialSettings(next)
    }, [])

    useEffect(() => {
        Promise.all([
            apiFetch(`${API_URL}/api/settings/schema`),
            apiFetch(`${API_URL}/api/settings`),
        ])
            .then(async ([schemaRes, valuesRes]) => {
                if (!schemaRes.ok || !valuesRes.ok) throw new Error("Failed to load settings")
                return [await schemaRes.json(), await valuesRes.json()] as const
            })
            .then(([schemaDefs, data]: readonly [SettingDef[], Record<string, string>]) => {
                // Seed all keys with schema defaults, then overwrite with actual DB values.
                const next: Record<string, string> = {}
                for (const def of schemaDefs) {
                    next[def.key] = data[def.key] ?? def.default
                }
                setSchema(schemaDefs)
                setSettings(next)
                setInitialSettings(next)
            })
            .catch((error) => {
                console.error("Fetch settings error:", error)
                showToast("Failed to load settings", "error")
            })
            .finally(() => setLoading(false))
    }, [])

    const isDirty = useMemo(
        () => Boolean(initialSettings && settings && JSON.stringify(initialSettings) !== JSON.stringify(settings)),
        [initialSettings, settings],
    )

    const update = (key: string, value: string) => {
        setSettings((prev) => (prev ? { ...prev, [key]: value } : prev))
    }

    const handleSave = async () => {
        if (!settings) return
        setSaving(true)
        try {
            const payload = { ...settings }
            if (payload.prowlarrApiKey !== undefined) {
                payload.prowlarrApiKey = payload.prowlarrApiKey.trim() || "error"
            }
            const response = await apiFetch(`${API_URL}/api/settings`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(payload),
            })
            if (response.ok) {
                showToast("Settings saved", "success")
                fetchSettings().catch((error) => {
                    console.error("Fetch settings error:", error)
                    showToast("Failed to load settings", "error")
                })
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

    const handleTestKbdex = async () => {
        setTestingKbdex(true)
        try {
            const res = await apiFetch(`${API_URL}/api/settings/test/kbdex`, { method: "POST" })
            const data = await res.json()
            showToast(data.message, data.ok === "true" ? "success" : "error")
        } catch { showToast("Test failed", "error") }
        finally { setTestingKbdex(false) }
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

    const handleTestJellyfin = async () => {
        setTestingJellyfin(true)
        try {
            const res = await apiFetch(`${API_URL}/api/settings/test/jellyfin`, { method: "POST" })
            const data = await res.json()
            showToast(data.message, data.ok === "true" ? "success" : "error")
        } catch { showToast("Test failed", "error") }
        finally { setTestingJellyfin(false) }
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

    if (loading) {
        return <Group justify="center" py="xl"><Loader color="gray" /></Group>
    }
    if (!settings) {
        return <Text c="red" ta="center" py="xl">Failed to load settings. Please refresh the page.</Text>
    }

    const showSaveBar =
        isDownloader ||
        path.includes("metadata") ||
        path.includes("indexer") ||
        (path.includes("general") && isDirty)

    return (
        <Stack gap="lg" pb={80}>
            <Title order={1}>
                {path.includes("metadata") ? "Metadata" : path.includes("indexer") ? "Indexer" : isDownloader ? "Downloader" : "General"}
            </Title>

            {path.includes("general") && (
                <>
                    <Card withBorder radius="xl" p="lg" style={{ borderColor: "var(--mantine-color-orange-5)" }}>
                        <Stack gap="md">
                            <div>
                                <Group gap="xs">
                                    <IconAlertTriangle size={18} color="var(--mantine-color-orange-5)" />
                                    <Title order={3} c="orange">Developer options</Title>
                                </Group>
                                <Text size="sm" c="dimmed">Reveals advanced tuning settings across all tabs. Defaults are tuned for normal use — only change these if you know what they do.</Text>
                            </div>
                            <Switch
                                label="Show developer options"
                                color="orange"
                                checked={devOptions}
                                onChange={(e) => toggleDevOptions(e.currentTarget.checked)}
                            />
                        </Stack>
                    </Card>

                    <SectionCard
                        section="General settings"
                        defs={defsForSection(schema, "general", "General settings")}
                        values={settings}
                        initialValues={initialSettings ?? {}}
                        onUpdate={update}
                        devOptions={devOptions}
                    />

                    <Card withBorder radius="xl" p="lg">
                        <Stack gap="md">
                            <div>
                                <Title order={3}>Account</Title>
                                <Text size="sm" c="dimmed">Change your login username or password. Current password is always required.</Text>
                            </div>
                            <Divider />
                            {credsEnvManaged ? (
                                <Alert color="gray" variant="light" icon={<IconInfoCircle size={18} />}>
                                    Credentials are managed by the <Text span fw={600}>KBARR_AUTH_USERNAME</Text> and <Text span fw={600}>KBARR_AUTH_PASSWORD</Text> environment variables. To change them, edit your environment and restart kbarr.
                                </Alert>
                            ) : (
                                <>
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
                                            variant="light"
                                            onClick={handleSaveCreds}
                                            loading={savingCreds}
                                            disabled={!creds.currentPassword || (!creds.newUsername && !creds.newPassword)}
                                        >
                                            Update credentials
                                        </Button>
                                    </Group>
                                </>
                            )}
                        </Stack>
                    </Card>

                    {sectionsForGroup(schema, "general")
                        .filter((s) => s !== "General settings")
                        .map((section) => (
                            <SectionCard
                                key={section}
                                section={section}
                                defs={defsForSection(schema, "general", section)}
                                values={settings}
                                initialValues={initialSettings ?? {}}
                                onUpdate={update}
                        devOptions={devOptions}
                            />
                        ))}
                </>
            )}

            {path.includes("metadata") && (
                <>
                    {sectionsForGroup(schema, "metadata").map((section) => (
                        <SectionCard
                            key={section}
                            section={section}
                            defs={defsForSection(schema, "metadata", section)}
                            values={settings}
                            initialValues={initialSettings ?? {}}
                            onUpdate={update}
                        devOptions={devOptions}
                        />
                    ))}
                </>
            )}

            {path.includes("indexer") && (
                <>
                    {sectionsForGroup(schema, "indexer").map((section) => (
                        <SectionCard
                            key={section}
                            section={section}
                            defs={defsForSection(schema, "indexer", section)}
                            values={settings}
                            initialValues={initialSettings ?? {}}
                            onUpdate={update}
                        devOptions={devOptions}
                            extra={
                                section === "kbdex" ? (
                                    <Group justify="flex-end">
                                        <Button variant="light" loading={testingKbdex} onClick={handleTestKbdex}>
                                            Test connection
                                        </Button>
                                    </Group>
                                ) : section === "Prowlarr" ? (
                                    <Group justify="flex-end">
                                        <Button variant="light" loading={testingIndexer} onClick={handleTestIndexer}>
                                            Test connection
                                        </Button>
                                    </Group>
                                ) : undefined
                            }
                        />
                    ))}
                </>
            )}

            {isDownloader && (
                <>
                    {sectionsForGroup(schema, "downloader").map((section) => (
                        <SectionCard
                            key={section}
                            section={section}
                            defs={defsForSection(schema, "downloader", section)}
                            values={settings}
                            initialValues={initialSettings ?? {}}
                            onUpdate={update}
                        devOptions={devOptions}
                            extra={
                                section === "qBittorrent" ? (
                                    <Group justify="flex-end">
                                        <Button variant="light" loading={testingDownloader} onClick={handleTestDownloader}>
                                            Test connection
                                        </Button>
                                    </Group>
                                ) : section === "Jellyfin" ? (
                                    <Group justify="flex-end">
                                        <Button variant="light" loading={testingJellyfin} onClick={handleTestJellyfin}>
                                            Test connection
                                        </Button>
                                    </Group>
                                ) : undefined
                            }
                        />
                    ))}

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

                    {devOptions && (
                        <Card withBorder radius="xl" p="lg" style={{ borderColor: "var(--mantine-color-orange-5)" }}>
                            <Stack gap="md">
                                <div>
                                    <Group gap="xs">
                                        <IconAlertTriangle size={18} color="var(--mantine-color-orange-5)" />
                                        <Title order={3} c="orange">Blacklist</Title>
                                    </Group>
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
                    )}
                </>
            )}

            {showSaveBar && (
                <Group justify="flex-end" style={{ position: "sticky", bottom: 16 }}>
                    {isDirty && (
                        <Alert color="gray" variant="light" style={{ flex: 1, maxWidth: 480 }}>
                            You have unsaved changes.
                        </Alert>
                    )}
                    <Button variant="light" onClick={handleSave} disabled={!isDirty || saving} loading={saving}>
                        Save changes
                    </Button>
                </Group>
            )}
        </Stack>
    )
}
