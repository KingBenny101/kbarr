import { useState, useEffect } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { API_URL } from "@/lib/api"

interface Anime {
    ID: number
    Title: string
    TitleJP?: string
    Episodes?: number
    Status: string
    AddedAt?: string
}


export function LibraryPage() {
    const [animeList, setAnimeList] = useState<Anime[]>([])

    useEffect(() => {
        fetchList()
    }, [])

    const fetchList = async (): Promise<void> => {
        try {
            const res = await fetch(`${API_URL}/api/library`)
            const data = (await res.json()) as Anime[]
            setAnimeList(data || [])
        } catch (err) {
            console.error("Failed to fetch media list:", err)
        }
    }

    return (
        <div>
            {animeList.length === 0 ? (
                <p className="text-center py-8">No media added yet</p>
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
