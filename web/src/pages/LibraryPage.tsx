import { useEffect, useMemo, useRef, useState } from "react"
import { useNavigate } from "react-router"
import { useVirtualizer } from "@tanstack/react-virtual"
import { Badge, Card, Center, Collapse, Divider, Group, ScrollArea, Select, Stack, Text, TextInput, Title, UnstyledButton } from "@mantine/core"
import { IconChevronDown, IconChevronRight, IconPlayerPlayFilled, IconSearch } from "@tabler/icons-react"
import { API_URL, apiFetch, resolvePosterUrl, showToast } from "@/utils"
import type { Media } from "@/types"
import { EmptyState } from "@/components"

type SortKey = "name-asc" | "name-desc" | "date-new" | "date-old"

const SORT_OPTIONS = [
    { value: "name-asc", label: "Name A–Z" },
    { value: "name-desc", label: "Name Z–A" },
    { value: "date-new", label: "Date Added (Newest)" },
    { value: "date-old", label: "Date Added (Oldest)" },
]

function sortMedia(items: Media[], key: SortKey): Media[] {
    return [...items].sort((a, b) => {
        switch (key) {
            case "name-asc":
                return a.title.localeCompare(b.title)
            case "name-desc":
                return b.title.localeCompare(a.title)
            case "date-new":
                return new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime()
            case "date-old":
                return new Date(a.CreatedAt).getTime() - new Date(b.CreatedAt).getTime()
        }
    })
}

function MediaCard({ media, onClick }: { media: Media; onClick: () => void }) {
    return (
        <Card withBorder radius="xl" p={0} h={CARD_HEIGHT} style={{ overflow: "hidden", cursor: "pointer", position: "relative" }} onClick={onClick}>
            {media.is_nsfw && (
                <Badge color="red" size="xs" style={{ position: "absolute", top: 8, right: 8, zIndex: 1 }}>
                    NSFW
                </Badge>
            )}
            {media.poster_url ? (
                <div style={{ width: "100%", height: POSTER_HEIGHT }}>
                    <img src={resolvePosterUrl(media.poster_url)} alt={media.title} style={{ width: "100%", height: "100%", objectFit: "cover" }} onError={(e) => { e.currentTarget.src = "/placeholder.svg" }} />
                </div>
            ) : (
                <Center style={{ width: "100%", height: POSTER_HEIGHT, background: "rgba(255,255,255,0.04)" }}>
                    <IconPlayerPlayFilled size={30} opacity={0.5} />
                </Center>
            )}
            <Stack gap={0} p="xs" style={{ flex: 1, minHeight: 0, overflow: "hidden" }}>
                <Title order={6} lineClamp={2}>
                    {media.title}
                </Title>
                <Text size="10px" c="dimmed">
                    {media.source} ID {media.source_id}
                </Text>
            </Stack>
        </Card>
    )
}

export function LibraryPage() {
    const [medias, setMedias] = useState<Media[]>([])
    const [filter, setFilter] = useState("")
    const [sort, setSort] = useState<SortKey>(() => (localStorage.getItem("library.sort") as SortKey) ?? "name-asc")
    const [nsfwOpen, setNsfwOpen] = useState(() => sessionStorage.getItem("library.nsfwOpen") === "true")
    const navigate = useNavigate()

    useEffect(() => {
        apiFetch(`${API_URL}/api/library`)
            .then((response) => {
                if (!response.ok) throw new Error(response.statusText)
                return response.json()
            })
            .then((data: Media[]) => setMedias(data || []))
            .catch((error) => {
                console.error("Failed to fetch media list:", error)
                showToast("Failed to load library", "error")
            })
    }, [])

    const { normalItems, nsfwItems } = useMemo(() => {
        const query = filter.toLowerCase()
        const filtered = query ? medias.filter((m) => m.title.toLowerCase().includes(query)) : medias
        const sorted = sortMedia(filtered, sort)
        return {
            normalItems: sorted.filter((m) => !m.is_nsfw),
            nsfwItems: sorted.filter((m) => m.is_nsfw),
        }
    }, [medias, filter, sort])

    return (
        <Stack gap="lg">
            <Title order={1}>Library</Title>

            {medias.length === 0 ? (
                <EmptyState
                    icon={<IconPlayerPlayFilled size={30} />}
                    title="No anime added yet"
                    description="Use search to add your first title and start building out the library."
                />
            ) : (
                <Stack gap="lg">
                    <Group gap="sm">
                        <TextInput
                            placeholder="Filter titles…"
                            leftSection={<IconSearch size={18} />}
                            value={filter}
                            onChange={(e) => setFilter(e.currentTarget.value)}
                            style={{ flex: 1 }}
                            size="md"
                        />
                        <Select
                            data={SORT_OPTIONS}
                            value={sort}
                            onChange={(v) => { if (v) { setSort(v as SortKey); localStorage.setItem("library.sort", v) } }}
                            w={200}
                            allowDeselect={false}
                            size="md"
                        />
                    </Group>

                    {normalItems.length === 0 && filter ? (
                        <Text c="dimmed" size="sm">No titles match "{filter}"</Text>
                    ) : (
                        <HorizontalGrid items={normalItems} onCardClick={(id) => navigate(`/media/${id}`)} />
                    )}

                    {nsfwItems.length > 0 && (
                        <Stack gap="sm">
                            <Divider />
                            <UnstyledButton onClick={() => setNsfwOpen((o) => { const next = !o; sessionStorage.setItem("library.nsfwOpen", String(next)); return next })}>
                                <Group gap="xs">
                                    {nsfwOpen ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
                                    <Text size="sm" c="dimmed">
                                        NSFW Titles ({nsfwItems.length})
                                    </Text>
                                </Group>
                            </UnstyledButton>
                            <Collapse expanded={nsfwOpen}>
                                <HorizontalGrid items={nsfwItems} onCardClick={(id) => navigate(`/media/${id}`)} />
                            </Collapse>
                        </Stack>
                    )}
                </Stack>
            )}
        </Stack>
    )
}

const ROWS = 2
const CARD_WIDTH = 180
const GAP = 16
// Fixed card dimensions: a 3/4 poster (180×240) plus an 80px footer that fits a
// two-line title and the source line. Shared with the Explore page for parity.
const POSTER_HEIGHT = 240
const FOOTER_HEIGHT = 80
const CARD_HEIGHT = POSTER_HEIGHT + FOOTER_HEIGHT
const GRID_HEIGHT = ROWS * CARD_HEIGHT + (ROWS - 1) * GAP + GAP

// Renders cards in a 2-row, horizontally-scrolling strip. Only the columns
// currently in view are mounted (windowing), so a large library stays fast.
function HorizontalGrid({ items, onCardClick }: { items: Media[]; onCardClick: (id: number) => void }) {
    const viewportRef = useRef<HTMLDivElement>(null)
    const columnCount = Math.ceil(items.length / ROWS)

    const virtualizer = useVirtualizer({
        count: columnCount,
        horizontal: true,
        getScrollElement: () => viewportRef.current,
        estimateSize: () => CARD_WIDTH + GAP,
        overscan: 4,
    })

    return (
        <ScrollArea scrollbars="x" type="always" scrollbarSize={6} viewportRef={viewportRef}>
            <div style={{ position: "relative", width: virtualizer.getTotalSize(), height: GRID_HEIGHT }}>
                {virtualizer.getVirtualItems().map((column) => {
                    const start = column.index * ROWS
                    const colItems = items.slice(start, start + ROWS)
                    return (
                        <div
                            key={column.key}
                            style={{
                                position: "absolute",
                                top: 0,
                                left: 0,
                                transform: `translateX(${column.start}px)`,
                                width: CARD_WIDTH,
                                display: "grid",
                                gridTemplateRows: `repeat(${ROWS}, ${CARD_HEIGHT}px)`,
                                gap: `${GAP}px`,
                            }}
                        >
                            {colItems.map((media) => (
                                <MediaCard key={media.ID} media={media} onClick={() => onCardClick(media.ID)} />
                            ))}
                        </div>
                    )
                })}
            </div>
        </ScrollArea>
    )
}
