import { useEffect, useState } from "react"
import { ActionIcon, Button, Card, Group, Pagination, SimpleGrid, Stack, Text, TextInput, Title } from "@mantine/core"
import { IconExternalLink, IconSearch } from "@tabler/icons-react"
import { API_URL, apiFetch, showToast } from "@/utils"
import type { Media } from "@/types"
import { EmptyState, StatusPill } from "@/components"

const SEARCH_CACHE_KEY = "kbarr.search.cache.v2"

type SearchCache = {
    query: string
    results: Media[]
}

function readSearchCache(): SearchCache | null {
    if (typeof window === "undefined") return null

    try {
        const rawCache = window.localStorage.getItem(SEARCH_CACHE_KEY)
        if (!rawCache) return null

        const parsed = JSON.parse(rawCache) as SearchCache
        if (!Array.isArray(parsed.results)) return null

        return {
            query: typeof parsed.query === "string" ? parsed.query : "",
            results: parsed.results,
        }
    } catch {
        return null
    }
}

export function SearchPage() {
    const [cachedSearch] = useState(() => readSearchCache())
    const [query, setQuery] = useState(() => cachedSearch?.query ?? "")
    const [lastSearchQuery, setLastSearchQuery] = useState(() => cachedSearch?.query ?? "")
    const [medias, setMedias] = useState<Media[]>(() => cachedSearch?.results ?? [])
    const [hasSearched, setHasSearched] = useState(() => cachedSearch !== null)
    const [searching, setSearching] = useState(false)
    const [adding, setAdding] = useState<string | null>(null)
    const [currentPage, setCurrentPage] = useState(1)
    const itemsPerPage = 9

    useEffect(() => {
        if (typeof window === "undefined") return

        try {
            if (!hasSearched) {
                window.localStorage.removeItem(SEARCH_CACHE_KEY)
                return
            }

            window.localStorage.setItem(SEARCH_CACHE_KEY, JSON.stringify({ query: lastSearchQuery, results: medias } satisfies SearchCache))
        } catch (error) {
            console.error("Failed to cache search results:", error)
        }
    }, [hasSearched, lastSearchQuery, medias])

    const totalPages = Math.ceil(medias.length / itemsPerPage)
    const paginatedResults = medias.slice((currentPage - 1) * itemsPerPage, currentPage * itemsPerPage)

    const handleSearch = async () => {
        const trimmedQuery = query.trim()
        if (!trimmedQuery) return
        setCurrentPage(1)
        setSearching(true)
        try {
            const response = await apiFetch(`${API_URL}/api/search?q=${encodeURIComponent(trimmedQuery)}`)
            const data = (await response.json()) as Media[]
            setHasSearched(true)
            setLastSearchQuery(trimmedQuery)
            setMedias(data || [])
        } catch (error) {
            console.error("Search failed:", error)
        } finally {
            setSearching(false)
        }
    }

    const handleAdd = async (result: Media) => {
        const key = `${result.source}:${result.source_id}`
        setAdding(key)
        try {
            const response = await apiFetch(`${API_URL}/api/library`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ title: result.title, source: result.source, source_id: result.source_id }),
            })
            if (response.ok) {
                const data = await response.json()
                showToast(data.message, "success")
                setMedias((previous) =>
                    previous.map((media) =>
                        media.source === result.source && media.source_id === result.source_id ? { ...media, added: true } : media
                    )
                )
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
            <Title order={1}>Search</Title>

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
                <Button color="gray" onClick={handleSearch} loading={searching} size="md">
                    Search
                </Button>
            </Group>

            {medias.length > 0 ? (
                <Stack gap="lg">
                    <SimpleGrid cols={{ base: 1, sm: 2, lg: 3 }} spacing="md">
                        {paginatedResults.map((result) => {
                            const key = `${result.source}:${result.source_id}`
                            return (
                                <Card key={key} withBorder radius="xl">
                                    <Stack gap="sm" justify="space-between" h="100%">
                                        <Stack gap={6}>
                                            <Group justify="space-between" align="start" wrap="nowrap">
                                                <Title order={4} lineClamp={2} style={{ flex: 1 }}>
                                                    {result.title}
                                                </Title>
                                                {result.source === "anidb" && (
                                                    <ActionIcon component="a" href={`https://anidb.net/anime/${result.source_id}`} target="_blank" rel="noreferrer" variant="subtle" color="gray" aria-label="Open AniDB entry">
                                                        <IconExternalLink size={18} />
                                                    </ActionIcon>
                                                )}
                                            </Group>
                                            <Text size="sm" c="dimmed">
                                                {result.source} ID {result.source_id}
                                            </Text>
                                            <Group gap="xs">
                                                <StatusPill label={result.added ? "Added" : "Ready"} tone={result.added ? "green" : "blue"} />
                                            </Group>
                                        </Stack>

                                        <Button color="gray" variant={result.added ? "light" : "filled"} onClick={() => handleAdd(result)} disabled={adding === key || result.added} loading={adding === key} fullWidth>
                                            {result.added ? "Added" : "Add to Library"}
                                        </Button>
                                    </Stack>
                                </Card>
                            )
                        })}
                    </SimpleGrid>

                    <Group justify="space-between" align="center">
                        <Text size="sm" c="dimmed">
                            Page {totalPages === 0 ? 0 : currentPage} of {totalPages}
                        </Text>
                        <Pagination value={currentPage} onChange={setCurrentPage} total={Math.max(totalPages, 1)} color="gray" />
                    </Group>
                </Stack>
            ) : null}

            {!searching && hasSearched && medias.length === 0 ? (
                <EmptyState icon={<IconSearch size={28} />} title="No results found" description={`Nothing matched "${lastSearchQuery || query}". Try a different title or spelling.`} />
            ) : null}

            {!searching && !hasSearched ? (
                <EmptyState icon={<IconSearch size={28} />} title="Start searching" description="Use the search field above to discover anime to add to your library." />
            ) : null}
        </Stack>
    )
}
