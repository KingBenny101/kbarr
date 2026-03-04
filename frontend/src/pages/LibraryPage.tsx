import { useState, useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { Card, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { PlayCircle, Film, Tv } from "lucide-react"
import { API_URL } from "@/lib/api"

interface Media {
    id: number
    title: string
    title_original: string
    description: string
    status: string
    type: "anime" | "movie" | "tv"
    source: "anidb" | "tmdb"
    episodes: number
    seasons: number
    year: number
    cover_image: string
    external_id: string
    monitored: boolean
    added_at: string
}

export function LibraryPage() {
    const [mediaList, setMediaList] = useState<Media[]>([])
    const navigate = useNavigate()

    useEffect(() => {
        fetchList()
    }, [])

    const fetchList = async (): Promise<void> => {
        try {
            const res = await fetch(`${API_URL}/api/library`)
            const data = (await res.json()) as Media[]
            setMediaList(data || [])
        } catch (err) {
            console.error("Failed to fetch media list:", err)
        }
    }

    const getIcon = (type: string) => {
        switch (type) {
            case "anime": return <PlayCircle className="size-4 text-primary" />
            case "movie": return <Film className="size-4 text-primary" />
            case "tv": return <Tv className="size-4 text-primary" />
            default: return null
        }
    }

    return (
        <div className="space-y-6">
    
            {mediaList.length === 0 ? (
                <Card className="border-dashed flex flex-col items-center justify-center p-12 text-center space-y-4">
                    <div className="bg-muted rounded-full p-4">
                        <Film className="h-8 w-8 text-muted-foreground" />
                    </div>
                    <div className="space-y-1">
                        <p className="font-medium">No media added yet</p>
                        <p className="text-sm text-muted-foreground">Go to search and add some anime or movies!</p>
                    </div>
                </Card>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                    {mediaList.map((media) => (
                        <Card 
                            key={media.id} 
                            className="group cursor-pointer hover:bg-accent/50 transition-all duration-200 border border-border/50 shadow-sm hover:shadow-md h-32 flex flex-col justify-center"
                            onClick={() => navigate(`/media/${media.id}`)}
                        >
                            <CardHeader className="py-0 px-4 flex flex-row items-center space-y-0 gap-3">
                                <div className="p-2 bg-muted rounded-md group-hover:bg-primary/10 transition-colors">
                                    {getIcon(media.type)}
                                </div>
                                <div className="flex-grow min-w-0">
                                    <CardTitle className="text-sm font-semibold line-clamp-2 leading-tight">
                                        {media.title}
                                    </CardTitle>
                                    <div className="flex items-center gap-2 mt-1">
                                        {media.year > 0 && (
                                            <span className="text-[10px] text-muted-foreground font-medium">{media.year}</span>
                                        )}
                                        {media.monitored && (
                                            <Badge variant="secondary" className="text-[8px] h-4 px-1 bg-primary/10 text-primary border-none">
                                                MONITORED
                                            </Badge>
                                        )}
                                    </div>
                                </div>
                            </CardHeader>
                        </Card>
                    ))}
                </div>
            )}
        </div>
    )
}
