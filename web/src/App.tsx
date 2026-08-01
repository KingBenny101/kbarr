import { useEffect, useState } from "react"
import { Navigate, Route, Routes, Link, useLocation } from "react-router"
import { AppShell, Burger, Group, NavLink, Text, ActionIcon, useMantineColorScheme, useComputedColorScheme } from "@mantine/core"
import { useDisclosure } from "@mantine/hooks"
import { IconLibraryPhoto, IconSearch, IconCompass, IconListCheck, IconActivity, IconGauge, IconSettings, IconDatabase, IconPlug, IconDownload, IconCloudDownload, IconMoonFilled, IconSunFilled } from "@tabler/icons-react"
import { API_URL, apiFetch, clearToken, getToken } from "@/utils"
import { LibraryPage } from "@/pages/LibraryPage"
import { LoginPage } from "@/pages/LoginPage"
import { MediaDetailPage } from "@/pages/MediaDetailPage"
import { MonitorPage } from "@/pages/MonitorPage"
import { SearchPage } from "@/pages/SearchPage"
import { ExplorePage } from "@/pages/ExplorePage"
import { DownloadsPage } from "@/pages/DownloadsPage"
import { SettingsPage } from "@/pages/SettingsPage"
import { LogsPage } from "@/pages/LogsPage"
import { SystemPage } from "@/pages/SystemPage"

const navigation = [
    {
        group: "Media", items: [
            { label: "Explore", to: "/explore", icon: <IconCompass size={18} /> },
            { label: "Search", to: "/search", icon: <IconSearch size={18} /> },
            { label: "Library", to: "/", icon: <IconLibraryPhoto size={18} />, match: (p: string) => p === "/" || p.startsWith("/media") },
        ]
    },
    {
        group: "System", items: [
            { label: "Monitored", to: "/monitored", icon: <IconListCheck size={18} /> },
            { label: "Downloads", to: "/downloads", icon: <IconCloudDownload size={18} /> },
            { label: "Logs", to: "/logs", icon: <IconActivity size={18} /> },
            { label: "System", to: "/system", icon: <IconGauge size={18} /> },
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
    const [lastMediaPath, setLastMediaPath] = useState<string>(() => sessionStorage.getItem("last-media-path") ?? "/")
    const [prevPathname, setPrevPathname] = useState(pathname)

    // Adjust state during render (React's "adjusting state when props change"
    // pattern) so the sidebar link updates in the same pass as the navigation,
    // instead of round-tripping through an effect.
    if (prevPathname !== pathname) {
        setPrevPathname(pathname)
        if (pathname.startsWith("/media/")) {
            setLastMediaPath(pathname)
        } else if (pathname === "/") {
            setLastMediaPath("/")
        }
    }

    // sessionStorage is the external system — keeping it in sync is a real
    // effect, not render work.
    useEffect(() => {
        if (pathname.startsWith("/media/")) {
            sessionStorage.setItem("last-media-path", pathname)
        } else if (pathname === "/") {
            sessionStorage.removeItem("last-media-path")
        }
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
        <>
            <style>{`@keyframes kb-fade-in { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: translateY(0); } }`}</style>
            <AppShell
            header={{ height: 56 }}
            navbar={{ width: 260, breakpoint: "sm", collapsed: { mobile: !opened } }}
            padding="md"
        >
            <AppShell.Header>
                <Group h="100%" px="md" justify="space-between">
                    <Group gap="sm">
                        <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
                        <Group gap="xs">
                            <svg width={22} height={22} viewBox="0 0 32 32" fill="none">
                                <rect width="32" height="32" rx="7" fill="var(--mantine-color-yellow-6)"/>
                                <polygon points="16,3 26.2,7.9 28.7,18.9 21.6,27.7 10.4,27.7 3.3,18.9 5.8,7.9" fill="none" stroke="#fff" stroke-width="1.5" stroke-linejoin="round"/>
                                <polygon points="21.3,8.7 24.6,18.8 16,25 7.4,18.8 10.7,8.7" fill="none" stroke="#fff" stroke-width="1.5" stroke-linejoin="round"/>
                                <polygon points="16,11 20.3,18.5 11.7,18.5" fill="none" stroke="#fff" stroke-width="1.5" stroke-linejoin="round"/>
                            </svg>
                            <Text fw={700} size="lg" style={{ lineHeight: 1 }}>
                                kbarr <Text component="span" size="xs" c="dimmed">v{version}</Text>
                            </Text>
                        </Group>
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
                                // The Library button normally returns to the last-viewed media
                                // page as a shortcut, but when already on a media page it should
                                // go to the library home instead.
                                const resolvedTo = label === "Library"
                                    ? (pathname.startsWith("/media/") ? "/" : lastMediaPath)
                                    : to
                                const active = match ? match(pathname) : pathname === to
                                return (
                                    <NavLink key={to} component={Link} to={resolvedTo} label={label} leftSection={icon} active={active} variant={active ? "filled" : "subtle"} color="yellow" styles={{ root: { borderRadius: 'var(--mantine-radius-xl)' } }} onClick={close} />
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

            <AppShell.Main>
                <div key={pathname} style={{ animation: "kb-fade-in 0.15s ease-out" }}>
                    {children}
                </div>
            </AppShell.Main>
        </AppShell>
        </>
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
                            <Route path="/explore" element={<ExplorePage />} />
                            <Route path="/search" element={<SearchPage />} />
                            <Route path="/settings/*" element={<SettingsPage />} />
                            <Route path="/media/:id" element={<MediaDetailPage />} />
                            <Route path="/monitored" element={<MonitorPage />} />
                            <Route path="/logs" element={<LogsPage />} />
                            <Route path="/system" element={<SystemPage />} />
                            <Route path="/downloads" element={<DownloadsPage />} />
                            <Route path="*" element={<Navigate to="/" replace />} />
                        </Routes>
                    </PageShell>
                </AuthGate>
            } />
        </Routes>
    )
}
