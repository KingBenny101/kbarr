import { useEffect, useState, type ReactNode } from "react"
import { AppShell, Burger, Group, NavLink, ScrollArea, Stack, Text } from "@mantine/core"
import { useDisclosure } from "@mantine/hooks"
import { IconActivity, IconLibraryPhoto, IconListCheck, IconSearch, IconSettings, IconTimeline } from "@tabler/icons-react"
import { Link, useLocation } from "react-router-dom"
import { API_URL } from "@/lib/api"
import { ColorModeToggle } from "./ColorModeToggle"

type NavEntry = {
    label: string
    to: string
    icon: ReactNode
    match?: (pathname: string) => boolean
}

const navigation: Array<{ group: string; items: NavEntry[] }> = [
    {
        group: "Media",
        items: [
            { label: "Library", to: "/", icon: <IconLibraryPhoto size={18} />, match: (pathname) => pathname === "/" || pathname.startsWith("/media") },
            { label: "Search", to: "/search", icon: <IconSearch size={18} /> },
            { label: "Monitored", to: "/monitored", icon: <IconListCheck size={18} /> },
        ],
    },
    {
        group: "System",
        items: [
            { label: "Search Queue", to: "/search-queue", icon: <IconTimeline size={18} /> },
            { label: "Workers", to: "/workers", icon: <IconActivity size={18} /> },
        ],
    },
    {
        group: "Settings",
        items: [
            { label: "General", to: "/settings/general", icon: <IconSettings size={18} />, match: (pathname) => pathname === "/settings" || pathname.includes("general") },
            { label: "AniDB", to: "/settings/anidb", icon: <IconSettings size={18} />, match: (pathname) => pathname.includes("anidb") },
            { label: "Prowlarr", to: "/settings/prowlarr", icon: <IconSettings size={18} />, match: (pathname) => pathname.includes("prowlarr") },
        ],
    },
]

export function PageShell({ children }: { children: React.ReactNode }) {
    const [opened, { toggle, close }] = useDisclosure(false)
    const location = useLocation()
    const [version, setVersion] = useState("0.0.0")

    useEffect(() => {
        fetch(`${API_URL}/api/version`)
            .then((response) => {
                if (!response.ok) throw new Error("version fetch failed")
                return response.json()
            })
            .then((data) => setVersion(data.version ?? "0.0.0"))
            .catch(() => setVersion("0.0.0"))
    }, [])

    return (
        <AppShell
            header={{ height: 56 }}
            navbar={{ width: 260, breakpoint: "sm", collapsed: { mobile: !opened } }}
            padding="md"
            styles={{ main: { background: "transparent" } }}
        >
            <AppShell.Header>
                <Group h="100%" px="md" justify="space-between">
                    <Group gap="sm">
                        <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
                        <Text fw={700}>kbarr</Text>
                        <Text size="xs" c="dimmed">
                            v{version}
                        </Text>
                    </Group>
                    <ColorModeToggle />
                </Group>
            </AppShell.Header>

            <AppShell.Navbar p="md">
                <Stack gap="xs" h="100%">
                    <ScrollArea type="auto" h="100%">
                        <Stack gap="md">
                            {navigation.map((section) => (
                                <Stack key={section.group} gap={4}>
                                    <Text size="xs" tt="uppercase" fw={700} c="dimmed">
                                        {section.group}
                                    </Text>
                                    {section.items.map((item) => {
                                        const active = item.match ? item.match(location.pathname) : location.pathname === item.to
                                        return (
                                            <NavLink
                                                key={item.to}
                                                component={Link}
                                                to={item.to}
                                                label={item.label}
                                                leftSection={item.icon}
                                                active={active}
                                                variant={active ? "filled" : "subtle"}
                                                color="gray"
                                                onClick={close}
                                            />
                                        )
                                    })}
                                </Stack>
                            ))}
                        </Stack>
                    </ScrollArea>
                </Stack>
            </AppShell.Navbar>

            <AppShell.Main>
                {children}
            </AppShell.Main>
        </AppShell>
    )
}