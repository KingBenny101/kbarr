import { useState, useEffect } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"

const API = "http://localhost:8282"

interface Anime {
    ID: number
    Title: string
    TitleJP?: string
    Episodes?: number
    Status: string
    AddedAt?: string
}

interface LibraryPageProps {
    onAnimeAdded?: () => void
}

export function LibraryPage({ onAnimeAdded }: LibraryPageProps) {
    const [animeList, setAnimeList] = useState<Anime[]>([])

    useEffect(() => {
        fetchList()
    }, [])

    const fetchList = async (): Promise<void> => {
        try {
            const res = await fetch(`${API}/api/anime`)
            const data = (await res.json()) as Anime[]
            setAnimeList(data || [])
        } catch (err) {
            console.error("Failed to fetch anime list:", err)
        }
    }

    return (
        <div>
            {animeList.length === 0 ? (
                <p className="text-center py-8">No anime added yet</p>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {animeList.map((anime) => (
                        <Card key={anime.ID}>
                            <CardHeader>
                                <CardTitle>{anime.Title}</CardTitle>
                                <CardDescription>ID: {anime.ID}</CardDescription>
                            </CardHeader>
                            <CardContent className="text-sm space-y-2">
                                <div>Episodes: {anime.Episodes || "—"}</div>
                                <div>Status: <Badge variant="secondary">{anime.Status}</Badge></div>
                                <div>Added: {anime.AddedAt ? new Date(anime.AddedAt).toLocaleDateString() : "—"}</div>
                                {anime.TitleJP && <div className="italic text-xs">{anime.TitleJP}</div>}
                            </CardContent>
                        </Card>
                    ))}
                </div>
            )}
        </div>
    )
}
