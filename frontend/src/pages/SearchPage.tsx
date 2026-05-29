import { useState } from "react"
import { ActionIcon, Button, Group, Pagination, SimpleGrid, Stack, Text, TextInput, Title } from "@mantine/core"
import { IconExternalLink, IconSearch } from "@tabler/icons-react"
import { API_URL } from "@/lib/api"
import { showToast } from "@/lib/notifications"
import type { Media } from "@/types/media"
import { EmptyState } from "@/components/EmptyState"
import { SectionCard } from "@/components/SectionCard"
import { StatusPill } from "@/components/StatusPill"

export function SearchPage() {
    const [query, setQuery] = useState("")
    const [medias, setMedias] = useState<Media[]>([])
    const [searching, setSearching] = useState(false)
    const [adding, setAdding] = useState<number | null>(null)
    const [currentPage, setCurrentPage] = useState(1)
    const itemsPerPage = 12

    const totalPages = Math.ceil(medias.length / itemsPerPage)
    const paginatedResults = medias.slice((currentPage - 1) * itemsPerPage, currentPage * itemsPerPage)

    const handleSearch = async () => {
        if (!query.trim()) return
        setCurrentPage(1)
        setSearching(true)
        try {
            const response = await fetch(`${API_URL}/api/search?q=${encodeURIComponent(query)}`)
            const data = (await response.json()) as Media[]
            setMedias(data || [])
        } catch (error) {
            console.error("Search failed:", error)
        } finally {
            setSearching(false)
        }
    }

    const handleAdd = async (result: Media) => {
        setAdding(result.aid)
        try {
            const response = await fetch(`${API_URL}/api/library`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(result),
            })
            if (response.ok) {
                const data = await response.json()
                showToast(data.message, "success")
                setMedias((previous) => previous.map((media) => (media.aid === result.aid ? { ...media, added: true } : media)))
            } else {
                showToast("Failed to add media", "error")
            }
        } catch (error) {
            console.error("Failed to add media:", error)
            showToast("An error occurred while adding media", "error")
        } finally {
            setAdding(null)
        }
    }

    return (
        <Stack gap="lg">
            <Stack gap={4}>
                <Text size="sm" c="dimmed" tt="uppercase" fw={700}>
                    Search
                </Text>
                <Title order={1}>Find anime and add it to the library</Title>
                <Text c="dimmed" maw={680}>
                    Query AniDB results, open the source entry, and add titles without leaving the page.
                </Text>
            </Stack>

            <Group align="end" wrap="nowrap">
                <TextInput
                    value={query}
                    onChange={(event) => setQuery(event.currentTarget.value)}
                    onKeyDown={(event) => event.key === "Enter" && handleSearch()}
                    placeholder="Search for anime..."
                    leftSection={<IconSearch size={18} />}
                    style={{ flex: 1 }}
                    size="md"
                />
                <Button color="gray" onClick={handleSearch} loading={searching}>
                    Search
                </Button>
            </Group>

            {medias.length > 0 ? (
                <Stack gap="lg">
                    <SimpleGrid cols={{ base: 1, sm: 2, lg: 3 }} spacing="md">
                        {paginatedResults.map((result) => (
                            <SectionCard key={result.aid} withBorder radius="xl">
                                <Stack gap="sm" justify="space-between" h="100%">
                                    <Stack gap={6}>
                                        <Group justify="space-between" align="start" wrap="nowrap">
                                            <Title order={4} lineClamp={2} style={{ flex: 1 }}>
                                                {result.title}
                                            </Title>
                                            <ActionIcon component="a" href={`https://anidb.net/anime/${result.aid}`} target="_blank" rel="noreferrer" variant="subtle" color="gray" aria-label="Open AniDB entry">
                                                <IconExternalLink size={18} />
                                            </ActionIcon>
                                        </Group>
                                        <Text size="sm" c="dimmed">
                                            AniDB ID {result.aid}
                                        </Text>
                                        <Group gap="xs">
                                            <StatusPill label={result.added ? "Added" : "Ready"} tone={result.added ? "green" : "blue"} />
                                        </Group>
                                    </Stack>

                                    <Button color="gray" variant={result.added ? "light" : "filled"} onClick={() => handleAdd(result)} disabled={adding === result.aid || result.added} loading={adding === result.aid} fullWidth>
                                        {result.added ? "Added" : "Add to Library"}
                                    </Button>
                                </Stack>
                            </SectionCard>
                        ))}
                    </SimpleGrid>

                    <Group justify="space-between" align="center">
                        <Text size="sm" c="dimmed">
                            Page {totalPages === 0 ? 0 : currentPage} of {totalPages}
                        </Text>
                        <Pagination value={currentPage} onChange={setCurrentPage} total={Math.max(totalPages, 1)} color="gray" />
                    </Group>
                </Stack>
            ) : null}

            {!searching && query && medias.length === 0 ? (
                <EmptyState icon={<IconSearch size={28} />} title="No results found" description={`Nothing matched “${query}”. Try a different title or spelling.`} />
            ) : null}

            {!query && !searching ? (
                <EmptyState icon={<IconSearch size={28} />} title="Start searching" description="Use the search field above to discover anime to add to your library." />
            ) : null}
        </Stack>
    )
}