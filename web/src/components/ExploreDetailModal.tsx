import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Anchor, Badge, Button, Divider, Grid, Group, Image, Loader, Modal, ScrollArea, Stack, Text, Title } from "@mantine/core"
import { IconExternalLink } from "@tabler/icons-react"
import { API_URL, apiFetch, showToast } from "@/utils"
import { displayTitle, fetchMediaDetails, type AniListMedia, type AniListMediaDetails } from "@/lib/anilist"

type Props = {
    media: AniListMedia | null
    opened: boolean
    onClose: () => void
}

// Strips AniList's lightweight HTML (mostly <br>, <i>, <b>) for plain display.
function stripHtml(html: string | null): string {
    if (!html) return ""
    return html
        .replace(/<br\s*\/?>/gi, "\n")
        .replace(/<[^>]+>/g, "")
        .replace(/\n{3,}/g, "\n\n")
        .trim()
}

export function ExploreDetailModal({ media, opened, onClose }: Props) {
    const navigate = useNavigate()
    const [details, setDetails] = useState<AniListMediaDetails | null>(null)
    const [loading, setLoading] = useState(false)
    const [adding, setAdding] = useState(false)

    useEffect(() => {
        if (!opened || !media) return
        let cancelled = false
        setDetails(null)
        setLoading(true)
        fetchMediaDetails(media.id)
            .then((d) => {
                if (!cancelled) setDetails(d)
            })
            .catch((error) => {
                console.error("Failed to load AniList details:", error)
                if (!cancelled) showToast("Failed to load details", "error")
            })
            .finally(() => {
                if (!cancelled) setLoading(false)
            })
        return () => {
            cancelled = true
        }
    }, [opened, media])

    if (!media) return null

    const title = displayTitle(media)

    const handleAdd = async () => {
        setAdding(true)
        try {
            // Pass all title variants so the backend can fuzzy-match against the
            // AniDB title index when there's no direct ID mapping.
            const titleHints = [media.title.english, media.title.romaji, media.title.native]
                .filter((t): t is string => !!t)
            const titleParams = Array.from(new Set(titleHints))
                .map((t) => `title=${encodeURIComponent(t)}`)
                .join("&")
            const resolveUrl = `${API_URL}/api/resolve/anilist/${media.id}${titleParams ? `?${titleParams}` : ""}`
            const resolveRes = await apiFetch(resolveUrl)
            const resolved = (await resolveRes.json()) as { aid: string; found: boolean; matched: string }

            if (!resolved.found) {
                showToast("No AniDB match for this title — search for it manually to add it", "error")
                onClose()
                navigate(`/search?q=${encodeURIComponent(title)}`)
                return
            }

            const addRes = await apiFetch(`${API_URL}/api/library`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ title, source: "anidb", source_id: resolved.aid }),
            })
            if (addRes.ok) {
                const data = await addRes.json()
                const base = data.message ?? "Added to library"
                showToast(resolved.matched === "title" ? `${base} (matched by title)` : base, "success")
                onClose()
            } else {
                showToast("Failed to add media", "error")
            }
        } catch (error) {
            console.error("Failed to add media:", error)
            showToast("An error occurred while adding media", "error")
        } finally {
            setAdding(false)
        }
    }

    const description = stripHtml(details?.description ?? null)

    return (
        <Modal opened={opened} onClose={onClose} title={title} size="xl" radius="lg" scrollAreaComponent={ScrollArea.Autosize}>
            <Grid>
                <Grid.Col span={{ base: 12, sm: 4 }}>
                    <Image src={media.coverImage.extraLarge ?? media.coverImage.large ?? undefined} radius="md" alt={title} fallbackSrc="https://placehold.co/300x420?text=No+Image" />
                </Grid.Col>
                <Grid.Col span={{ base: 12, sm: 8 }}>
                    <Stack gap="sm">
                        <Group gap="xs">
                            {media.format && <Badge variant="light" color="gray">{media.format}</Badge>}
                            {media.status && <Badge variant="light" color="blue">{media.status.replace(/_/g, " ")}</Badge>}
                            {typeof media.averageScore === "number" && <Badge variant="light" color="green">{media.averageScore}%</Badge>}
                            {media.isAdult && <Badge variant="light" color="red">NSFW</Badge>}
                        </Group>

                        <Group gap="lg">
                            {media.seasonYear && (
                                <Text size="sm" c="dimmed">
                                    {media.season ? `${media.season.charAt(0)}${media.season.slice(1).toLowerCase()} ` : ""}
                                    {media.seasonYear}
                                </Text>
                            )}
                            {typeof media.episodes === "number" && (
                                <Text size="sm" c="dimmed">{media.episodes} episodes</Text>
                            )}
                            {details?.duration && <Text size="sm" c="dimmed">{details.duration} min/ep</Text>}
                            {media.countryOfOrigin && <Text size="sm" c="dimmed">{media.countryOfOrigin}</Text>}
                        </Group>

                        {media.genres.length > 0 && (
                            <Group gap={6}>
                                {media.genres.map((g) => (
                                    <Badge key={g} variant="outline" color="gray" radius="sm">{g}</Badge>
                                ))}
                            </Group>
                        )}

                        {details?.studios.nodes.length ? (
                            <Text size="sm">
                                <Text span c="dimmed">Studio: </Text>
                                {details.studios.nodes.map((s) => s.name).join(", ")}
                            </Text>
                        ) : null}

                        {details?.siteUrl && (
                            <Anchor href={details.siteUrl} target="_blank" rel="noreferrer" size="sm">
                                <Group gap={4}>
                                    <IconExternalLink size={14} /> More details
                                </Group>
                            </Anchor>
                        )}
                    </Stack>
                </Grid.Col>
            </Grid>

            <Divider my="md" />

            {loading ? (
                <Group justify="center" py="lg"><Loader size="sm" /></Group>
            ) : (
                <Stack gap="md">
                    {description ? (
                        <Stack gap={4}>
                            <Title order={5}>Synopsis</Title>
                            <Text size="sm" style={{ whiteSpace: "pre-line" }}>{description}</Text>
                        </Stack>
                    ) : null}

                    <Button onClick={handleAdd} loading={adding} fullWidth color="gray">
                        Add to Library
                    </Button>
                </Stack>
            )}
        </Modal>
    )
}
