import { useEffect, useState } from "react"
import { Navigate, Route, Routes, Link, useLocation } from "react-router-dom"
import { AppShell, Burger, Group, NavLink, Text, ActionIcon, useMantineColorScheme, useComputedColorScheme } from "@mantine/core"
import { useDisclosure } from "@mantine/hooks"
import { IconLibraryPhoto, IconSearch, IconListCheck, IconTimeline, IconActivity, IconSettings, IconDatabase, IconPlug, IconMoonFilled, IconSunFilled } from "@tabler/icons-react"
import { API_URL } from "@/utils"
import { LibraryPage } from "@/pages/LibraryPage"
import { MediaDetailPage } from "@/pages/MediaDetailPage"
import { MonitorPage } from "@/pages/MonitorPage"
import { SearchPage } from "@/pages/SearchPage"
import { SearchQueuePage } from "@/pages/SearchQueuePage"
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
            { label: "Search Queue", to: "/search-queue", icon: <IconTimeline size={18} /> },
            { label: "Workers", to: "/workers", icon: <IconActivity size={18} /> },
        ]
    },
    {
        group: "Settings", items: [
            { label: "General", to: "/settings/general", icon: <IconSettings size={18} />, match: (p: string | string[]) => p === "/settings" || p.includes("general") },
            { label: "Metadata", to: "/settings/metadata", icon: <IconDatabase size={18} />, match: (p: string | string[]) => p.includes("metadata") },
            { label: "Indexer", to: "/settings/indexer", icon: <IconPlug size={18} />, match: (p: string | string[]) => p.includes("indexer") },
        ]
    },
]

function PageShell({ children }: { children: React.ReactNode }) {
    const [opened, { toggle, close }] = useDisclosure(false)
    const pathname = useLocation().pathname
    const [version, setVersion] = useState("0.0.0")
    const { toggleColorScheme } = useMantineColorScheme()
    const computedColorScheme = useComputedColorScheme("light")

    useEffect(() => {
        fetch(`${API_URL}/api/version`)
            .then((res) => res.ok ? res.json() : Promise.reject())
            .then((data) => setVersion(data.version ?? "0.0.0"))
            .catch(() => setVersion("0.0.0"))
    }, [])

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
                    <ActionIcon variant="subtle" color="gray" onClick={toggleColorScheme} aria-label="Toggle color scheme">
                        {computedColorScheme === "dark" ? <IconSunFilled size={18} /> : <IconMoonFilled size={18} />}
                    </ActionIcon>
                </Group>
            </AppShell.Header>

            <AppShell.Navbar p="md">
                <AppShell.Section grow>
                    {navigation.map(({ group, items }) => (
                        <div key={group} style={{ marginBottom: 24 }}>
                            <Text size="xs" tt="uppercase" fw={700} c="dimmed" mb="sm">{group}</Text>
                            {items.map(({ to, label, icon, match }) => {
                                const active = match ? match(pathname) : pathname === to
                                return (
                                    <NavLink key={to} component={Link} to={to} label={label} leftSection={icon} active={active} variant={active ? "filled" : "subtle"} color="gray" styles={{ root: { borderRadius: 'var(--mantine-radius-xl)' } }} onClick={close} />
                                )
                            })}
                        </div>
                    ))}
                </AppShell.Section>
            </AppShell.Navbar>

            <AppShell.Main>{children}</AppShell.Main>
        </AppShell>
    )
}

export default function App() {
    return (
        <PageShell>
            <Routes>
                <Route path="/" element={<LibraryPage />} />
                <Route path="/search" element={<SearchPage />} />
                <Route path="/settings/*" element={<SettingsPage />} />
                <Route path="/media/:id" element={<MediaDetailPage />} />
                <Route path="/monitored" element={<MonitorPage />} />
                <Route path="/workers" element={<WorkersPage />} />
                <Route path="/search-queue" element={<SearchQueuePage />} />
                <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
        </PageShell>
    )
}
