import { useEffect, useState } from "react"
import { AppShell, Burger, Box, Group, NavLink, Text } from "@mantine/core"
import { useDisclosure } from "@mantine/hooks"
import { IconLibraryPhoto, IconSearch, IconListCheck, IconTimeline, IconActivity, IconSettings } from "@tabler/icons-react"
import { Link, useLocation } from "react-router-dom"
import { ColorModeToggle } from "./ColorModeToggle"
import { API_URL } from "@/lib/api"

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
            { label: "AniDB", to: "/settings/anidb", icon: <IconSettings size={18} />, match: (p: string | string[]) => p.includes("anidb") },
            { label: "Prowlarr", to: "/settings/prowlarr", icon: <IconSettings size={18} />, match: (p: string | string[]) => p.includes("prowlarr") },
        ]
    },
]

export function PageShell({ children }: { children: React.ReactNode }) {
    const [opened, { toggle, close }] = useDisclosure(false)
    const pathname = useLocation().pathname
    const [version, setVersion] = useState("0.0.0")

    useEffect(() => {
        fetch(`${API_URL}/api/version`)
            .then((res) => res.ok ? res.json() : Promise.reject())
            .then((data) => setVersion(data.version ?? "0.0.0"))
            .catch(() => setVersion("0.0.0"))
    }, [])

    return (
        <AppShell navbar={{ width: 260, breakpoint: "sm", collapsed: { mobile: !opened } }} padding="md">
            <AppShell.Navbar pt="xs" px="md" pb="md" style={{ display: "flex", flexDirection: "column" }}>
                <Group mb="md">
                    <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
                </Group>

                <Box style={{ flex: 1 }}>
                    {navigation.map(({ group, items }) => (
                        <div key={group} style={{ marginBottom: "24px" }}>
                            <Text size="xs" tt="uppercase" fw={700} c="dimmed" mb="sm">{group}</Text>
                            {items.map(({ to, label, icon, match }) => {
                                const active = match ? match(pathname) : pathname === to
                                return (
                                    <NavLink key={to} component={Link} to={to} label={label} leftSection={icon} active={active} variant={active ? "filled" : "subtle"} color="gray" onClick={close} />
                                )
                            })}
                        </div>
                    ))}
                </Box>

                <Group mt="md" justify="space-between" align="center" wrap="nowrap">
                    <Text fw={700} size="xl" style={{ lineHeight: 1 }}>
                        kbarr <Text component="span" size="xs" c="dimmed">v{version}</Text>
                    </Text>
                    <ColorModeToggle />
                </Group>
            </AppShell.Navbar>

            <AppShell.Main>{children}</AppShell.Main>
        </AppShell>
    )
}