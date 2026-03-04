"use client"

import { useState, useMemo } from "react"
import { Search, Film, Tv, PlayCircle, ExternalLink } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from "@/components/ui/input-group"
import { Card, CardDescription, CardHeader, CardTitle, CardFooter } from "@/components/ui/card"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { API_URL } from "@/lib/api"
import { showToast } from "@/lib/utils"

interface SearchResult {
    id: string
    title: string
    source: "anidb" | "tmdb"
    media_type: "anime" | "movie" | "tv"
    poster?: string
    year?: string
}

interface SearchPageProps {
    onMediaAdded?: () => void
}

export function SearchPage({ onMediaAdded }: SearchPageProps) {
    const [query, setQuery] = useState<string>("")
    const [searchResults, setSearchResults] = useState<SearchResult[]>([])
    const [searching, setSearching] = useState<boolean>(false)
    const [adding, setAdding] = useState<string | null>(null)
    const [sourceFilter, setSourceFilter] = useState<string>("all")

    const filteredResults = useMemo(() => {
        if (sourceFilter === "all") return searchResults
        return searchResults.filter(r => r.source === sourceFilter)
    }, [searchResults, sourceFilter])

    const handleSearch = async (): Promise<void> => {
        if (!query.trim()) return
        setSearching(true)
        try {
            const res = await fetch(`${API_URL}/api/library/search?q=${encodeURIComponent(query)}`)
            const data = (await res.json()) as SearchResult[]
            setSearchResults(data || [])
        } catch (err) {
            console.error("Search failed:", err)
        } finally {
            setSearching(false)
        }
    }

    const handleAdd = async (result: SearchResult): Promise<void> => {
        setAdding(result.id)
        try {
            // Prepared body for both sources
            const body = result.source === "anidb"
                ? { aid: result.id }
                : { tmdb_id: result.id, media_type: result.media_type };

            const res = await fetch(`${API_URL}/api/library`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(body)
            })
            if (res.ok && onMediaAdded) {
                showToast("Media added successfully", "success")
                onMediaAdded()
            }

            if (!res.ok) {
                showToast("Failed to add media", "error")
            }
        } catch (err) {
            showToast("An error occurred while adding media", "error")
            console.error("Failed to add media:", err)
        } finally {
            setAdding(null)
        }
    }

    const getIcon = (type: string) => {
        switch (type) {
            case "anime": return <PlayCircle className="size-3" />
            case "movie": return <Film className="size-3" />
            case "tv": return <Tv className="size-3" />
            default: return null
        }
    }

    const getExternalUrl = (result: SearchResult) => {
        if (result.source === "anidb") {
            return `https://anidb.net/anime/${result.id}`
        }
        return `https://www.themoviedb.org/${result.media_type}/${result.id}`
    }

    return (
        <div>
            <InputGroup className="mb-6">
                <InputGroupInput
                    type="text"
                    placeholder="Search for anime, movies or TV shows..."
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && handleSearch()}
                    disabled={searching}
                />
                <InputGroupAddon align="inline-start">
                    <Search className="size-4" />
                </InputGroupAddon>
                <InputGroupAddon align="inline-end">
                    <InputGroupButton variant="secondary"
                        onClick={handleSearch}
                        disabled={searching || !query.trim()}
                    >
                        {searching ? "..." : "Search"}
                    </InputGroupButton>
                </InputGroupAddon>
            </InputGroup>

            {searchResults.length > 0 && (
                <div className="flex flex-col gap-6">
                    <div className="flex justify-center sm:justify-start">
                        <Tabs defaultValue="all" value={sourceFilter} onValueChange={setSourceFilter} className="w-full sm:w-auto">
                            <TabsList className="grid w-full grid-cols-3 sm:w-[300px]">
                                <TabsTrigger value="all">All</TabsTrigger>
                                <TabsTrigger value="anidb">AniDB</TabsTrigger>
                                <TabsTrigger value="tmdb">TMDB</TabsTrigger>
                            </TabsList>
                        </Tabs>
                    </div>

                    <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
                        {filteredResults.map((result) => (
                            <Card key={`${result.source}-${result.id}`} className="flex flex-col group">
                                <CardHeader className="p-4 pb-2 space-y-2">
                                    <div className="flex justify-between items-start gap-4">
                                        <div className="space-y-1 flex-1">
                                            <CardTitle className="text-sm line-clamp-1 leading-tight">{result.title}</CardTitle>
                                            <CardDescription className="text-xs flex gap-2 items-center">
                                                <span>ID: {result.id}</span>
                                                {result.year && <span>•</span>}
                                                {result.year && <span>{result.year}</span>}
                                            </CardDescription>
                                        </div>
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            className="size-8 text-muted-foreground hover:text-primary shrink-0"
                                            asChild
                                        >
                                            <a href={getExternalUrl(result)} target="_blank" rel="noopener noreferrer">
                                                <ExternalLink className="size-4" />
                                            </a>
                                        </Button>
                                    </div>
                                    <div className="flex gap-2">
                                        <Badge variant="secondary" className="capitalize flex items-center gap-1.5 text-[10px] h-5 px-1.5">
                                            {getIcon(result.media_type)}
                                            {result.media_type}
                                        </Badge>
                                        <Badge variant="outline" className="text-[10px] h-5 px-1.5 font-mono">
                                            {result.source.toUpperCase()}
                                        </Badge>
                                    </div>
                                </CardHeader>
                                <CardFooter className="p-4 pt-2 mt-auto">
                                    <Button
                                        size="sm"
                                        variant="secondary"
                                        onClick={() => handleAdd(result)}
                                        disabled={adding === result.id}
                                        className="w-full h-8 text-xs font-semibold"
                                    >
                                        {adding === result.id ? "Adding..." : "Add to Library"}
                                    </Button>
                                </CardFooter>
                            </Card>
                        ))}
                    </div>
                </div>
            )}

            {!searching && query && searchResults.length === 0 && (
                <p className="text-center py-12 text-muted-foreground italic">No results found for "{query}"</p>
            )}

            {!query && !searching && (
                <div className="text-center py-20 text-muted-foreground flex flex-col items-center gap-3">
                    <Search className="size-10 opacity-20" />
                    <p className="italic">Search for something to add to your library</p>
                </div>
            )}
        </div>
    )
}
