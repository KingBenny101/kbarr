import { useEffect, useState } from "react"
import { Navigate, Route, Routes, Link, useLocation } from "react-router-dom"
import { AppShell, Burger, Group, NavLink, Text, ActionIcon, useMantineColorScheme, useComputedColorScheme } from "@mantine/core"
import { useDisclosure } from "@mantine/hooks"
import { IconLibraryPhoto, IconSearch, IconListCheck, IconActivity, IconSettings, IconDatabase, IconPlug, IconDownload, IconCloudDownload, IconMoonFilled, IconSunFilled } from "@tabler/icons-react"
import { API_URL, apiFetch, clearToken, getToken } from "@/utils"
import { LibraryPage } from "@/pages/LibraryPage"
import { LoginPage } from "@/pages/LoginPage"
import { MediaDetailPage } from "@/pages/MediaDetailPage"
import { MonitorPage } from "@/pages/MonitorPage"
import { SearchPage } from "@/pages/SearchPage"
import { DownloadQueuePage } from "@/pages/DownloadQueuePage"
import { SettingsPage } from "@/pages/SettingsPage"
import { WorkersPage } from "@/pages/WorkersPage"

const navigation = [
    {
        group: "Media", items: [
            { label: "Library", to: "/", icon: <IconLibraryPhoto size={18} />, match: (p: string) => p === "/" || p.startsWith("/media") },
            { label: "Search", to: "/search", icon: <IconSearch size={18} /> },
            { label: "Monitored", to: "/monitored", icon: <IconListCheck size={18} /> },
        ]
    },
    {
        group: "System", items: [
{ label: "Download Queue", to: "/downloads", icon: <IconCloudDownload size={18} /> },
            { label: "Workers", to: "/workers", icon: <IconActivity size={18} /> },
        ]
    },
    {
        group: "Settings", items: [
            { label: "General", to: "/settings/general", icon: <IconSettings size={18} />, match: (p: string | string[]) => p === "/settings" || p.includes("general") },
            { label: "Metadata", to: "/settings/metadata", icon: <IconDatabase size={18} />, match: (p: string | string[]) => p.includes("metadata") },
            { label: "Indexer", to: "/settings/indexer", icon: <IconPlug size={18} />, match: (p: string | string[]) => p.includes("indexer") },
            { label: "Downloader", to: "/settings/downloader", icon: <IconDownload size={18} />, match: (p: string | string[]) => p.includes("downloader") },
        ]
    },
]

function PageShell({ children }: { children: React.ReactNode }) {
    const [opened, { toggle, close }] = useDisclosure(false)
    const pathname = useLocation().pathname
    const [version, setVersion] = useState("0.0.0")
    const [username, setUsername] = useState("")
    const [lastMediaPath, setLastMediaPath] = useState<string>("/")

    useEffect(() => {
        if (pathname.startsWith("/media/")) setLastMediaPath(pathname)
    }, [pathname])
    const { toggleColorScheme } = useMantineColorScheme()
    const computedColorScheme = useComputedColorScheme("light")

    useEffect(() => {
        apiFetch(`${API_URL}/api/version`)
            .then((res) => res.ok ? res.json() : Promise.reject())
            .then((data) => setVersion(data.version ?? "0.0.0"))
            .catch(() => setVersion("0.0.0"))
        apiFetch(`${API_URL}/api/auth/me`)
            .then((res) => res.ok ? res.json() : Promise.reject())
            .then((data) => setUsername(data.username ?? ""))
            .catch(() => {})
    }, [])

    const handleLogout = async () => {
        await apiFetch(`${API_URL}/api/auth/logout`, { method: "POST" })
        clearToken()
        window.location.href = "/login"
    }

    return (
        <AppShell
            header={{ height: 56 }}
            navbar={{ width: 260, breakpoint: "sm", collapsed: { mobile: !opened } }}
            padding="md"
        >
            <AppShell.Header>
                <Group h="100%" px="md" justify="space-between">
                    <Group gap="sm">
                        <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
                        <Text fw={700} size="lg" style={{ lineHeight: 1 }}>
                            kbarr <Text component="span" size="xs" c="dimmed">v{version}</Text>
                        </Text>
                    </Group>
                    <Group gap="xs">
                        {username && <Text size="sm" c="dimmed">{username}</Text>}
                        <ActionIcon variant="subtle" color="gray" onClick={handleLogout} aria-label="Sign out" title="Sign out">
                            <svg width={18} height={18} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
                                <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
                                <polyline points="16 17 21 12 16 7" />
                                <line x1="21" y1="12" x2="9" y2="12" />
                            </svg>
                        </ActionIcon>
                    </Group>
                </Group>
            </AppShell.Header>

            <AppShell.Navbar p="md">
                <AppShell.Section grow>
                    {navigation.map(({ group, items }) => (
                        <div key={group} style={{ marginBottom: 24 }}>
                            <Text size="xs" tt="uppercase" fw={700} c="dimmed" mb="sm">{group}</Text>
                            {items.map(({ to, label, icon, match }) => {
                                const resolvedTo = label === "Library" ? lastMediaPath : to
                                const active = match ? match(pathname) : pathname === to
                                return (
                                    <NavLink key={to} component={Link} to={resolvedTo} label={label} leftSection={icon} active={active} variant={active ? "filled" : "subtle"} color="gray" styles={{ root: { borderRadius: 'var(--mantine-radius-xl)' } }} onClick={close} />
                                )
                            })}
                        </div>
                    ))}
                </AppShell.Section>
                <AppShell.Section>
                    <ActionIcon variant="subtle" color="gray" onClick={toggleColorScheme} aria-label="Toggle color scheme">
                        {computedColorScheme === "dark" ? <IconSunFilled size={18} /> : <IconMoonFilled size={18} />}
                    </ActionIcon>
                </AppShell.Section>
            </AppShell.Navbar>

            <AppShell.Main>{children}</AppShell.Main>
        </AppShell>
    )
}

function AuthGate({ children }: { children: React.ReactNode }) {
    const token = getToken()
    const pathname = useLocation().pathname

    if (!token && pathname !== "/login") {
        return <Navigate to="/login" replace />
    }
    if (token && pathname === "/login") {
        return <Navigate to="/" replace />
    }
    return <>{children}</>
}

export default function App() {
    return (
        <Routes>
            <Route path="/login" element={<AuthGate><LoginPage /></AuthGate>} />
            <Route path="*" element={
                <AuthGate>
                    <PageShell>
                        <Routes>
                            <Route path="/" element={<LibraryPage />} />
                            <Route path="/search" element={<SearchPage />} />
                            <Route path="/settings/*" element={<SettingsPage />} />
                            <Route path="/media/:id" element={<MediaDetailPage />} />
                            <Route path="/monitored" element={<MonitorPage />} />
                            <Route path="/workers" element={<WorkersPage />} />
<Route path="/downloads" element={<DownloadQueuePage />} />
                            <Route path="*" element={<Navigate to="/" replace />} />
                        </Routes>
                    </PageShell>
                </AuthGate>
            } />
        </Routes>
    )
}
