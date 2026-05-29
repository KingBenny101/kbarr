import { useEffect, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Center, Group, Pagination, SimpleGrid, Stack, Text, Title, UnstyledButton } from "@mantine/core"
import { IconPlayerPlayFilled } from "@tabler/icons-react"
import { API_URL, resolvePosterUrl } from "@/lib/api"
import type { Media } from "@/types/media"
import { EmptyState } from "@/components/EmptyState"
import { SectionCard } from "@/components/SectionCard"

export function LibraryPage() {
    const [medias, setMedias] = useState<Media[]>([])
    const [currentPage, setCurrentPage] = useState(1)
    const navigate = useNavigate()
    const itemsPerPage = 18

    useEffect(() => {
        fetch(`${API_URL}/api/library`)
            .then((response) => response.json())
            .then((data: Media[]) => setMedias(data || []))
            .catch((error) => console.error("Failed to fetch media list:", error))
    }, [])

    const totalPages = Math.ceil(medias.length / itemsPerPage)
    const paginatedResults = useMemo(
        () => medias.slice((currentPage - 1) * itemsPerPage, currentPage * itemsPerPage),
        [currentPage, medias],
    )

    return (
        <Stack gap="lg">
            <Stack gap={4}>
                <Text size="sm" c="dimmed" tt="uppercase" fw={700}>
                    Library
                </Text>
                <Title order={1}>Your anime collection</Title>
                <Text c="dimmed" maw={680}>
                    Browse everything you have added so far. Cards are intentionally quiet and image-first so the content stays readable.
                </Text>
            </Stack>

            {medias.length === 0 ? (
                <EmptyState
                    icon={<IconPlayerPlayFilled size={30} />}
                    title="No anime added yet"
                    description="Use search to add your first title and start building out the library."
                />
            ) : (
                <Stack gap="lg">
                    <SimpleGrid cols={{ base: 1, sm: 2, lg: 3, xl: 4 }} spacing="md">
                        {paginatedResults.map((media) => (
                            <UnstyledButton key={media.ID} onClick={() => navigate(`/media/${media.ID}`)} style={{ borderRadius: 24, textAlign: "left" }}>
                                <SectionCard withBorder radius="xl" p={0} style={{ overflow: "hidden", cursor: "pointer" }}>
                                    {media.poster_url ? (
                                        <img
                                            src={resolvePosterUrl(media.poster_url)}
                                            alt={media.title}
                                            style={{ width: "100%", aspectRatio: "3 / 4", objectFit: "cover" }}
                                        />
                                    ) : (
                                        <Center style={{ width: "100%", aspectRatio: "3 / 4", background: "rgba(255,255,255,0.04)" }}>
                                            <IconPlayerPlayFilled size={30} opacity={0.5} />
                                        </Center>
                                    )}
                                    <Stack gap={4} p="md">
                                        <Title order={5} lineClamp={2}>
                                            {media.title}
                                        </Title>
                                        <Text size="xs" c="dimmed">
                                            AniDB ID {media.aid}
                                        </Text>
                                    </Stack>
                                </SectionCard>
                            </UnstyledButton>
                        ))}
                    </SimpleGrid>

                    <Group justify="space-between" align="center">
                        <Text size="sm" c="dimmed">
                            Page {totalPages === 0 ? 0 : currentPage} of {totalPages}
                        </Text>
                        <Pagination value={currentPage} onChange={setCurrentPage} total={Math.max(totalPages, 1)} color="gray" />
                    </Group>
                </Stack>
            )}
        </Stack>
    )
}