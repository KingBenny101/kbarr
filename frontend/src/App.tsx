import { useState, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
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

interface SearchResult {
  AID: number
  Title: string
}

interface ApiVersionResponse {
  version: string
}

type ViewType = "list" | "search"

export default function App() {
  const [query, setQuery] = useState<string>("")
  const [searchResults, setSearchResults] = useState<SearchResult[]>([])
  const [animeList, setAnimeList] = useState<Anime[]>([])
  const [searching, setSearching] = useState<boolean>(false)
  const [adding, setAdding] = useState<number | null>(null)
  const [view, setView] = useState<ViewType>("list")
  const [version, setVersion] = useState<string>("v0.1.0")

  useEffect(() => {
    fetchList()
    fetchVersion()
  }, [])

  const fetchVersion = async (): Promise<void> => {
    try {
      const res = await fetch(`${API}/api/version`)
      const data = (await res.json()) as ApiVersionResponse
      setVersion(data.version || "v0.1.0")
    } catch (err) {
      console.error("Failed to fetch version:", err)
    }
  }

  const fetchList = async (): Promise<void> => {
    try {
      const res = await fetch(`${API}/api/anime`)
      const data = (await res.json()) as Anime[]
      setAnimeList(data || [])
    } catch (err) {
      console.error("Failed to fetch anime list:", err)
    }
  }

  const handleSearch = async (): Promise<void> => {
    if (!query.trim()) return
    setSearching(true)
    try {
      const res = await fetch(`${API}/api/anime/search?q=${encodeURIComponent(query)}`)
      const data = (await res.json()) as SearchResult[]
      setSearchResults(data || [])
    } catch (err) {
      console.error("Search failed:", err)
    } finally {
      setSearching(false)
    }
  }

  const handleAdd = async (aid: number, title: string): Promise<void> => {
    setAdding(aid)
    try {
      const res = await fetch(`${API}/api/anime`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ aid })
      })
      if (res.ok) {
        await fetchList()
        setView("list")
      }
    } catch (err) {
      console.error("Failed to add anime:", err)
    } finally {
      setAdding(null)
    }
  }

  return (
    <div className="flex h-screen bg-background text-foreground">
      {/* Sidebar */}
      <aside className="w-40 bg-secondary border-r border-border flex flex-col sticky top-0 h-screen overflow-y-auto">
        {/* Logo */}
        <div className="p-4 border-b border-border">
          <h1 className="text-2xl font-bold text-accent">kbarr</h1>
        </div>

        {/* Navigation */}
        <nav className="flex-1 p-3 space-y-2">
          <Button
            onClick={() => {
              setView("list")
              fetchList()
            }}
            variant={view === "list" ? "default" : "ghost"}
            className="w-full justify-start"
          >
            Library
          </Button>
          <Button
            onClick={() => setView("search")}
            variant={view === "search" ? "default" : "ghost"}
            className="w-full justify-start"
          >
            Search
          </Button>
        </nav>

        {/* Footer */}
        <div className="p-4 border-t border-border">
          <p className="text-xs text-muted-foreground text-center">v{version}</p>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-auto">
        <div className="p-6 max-w-7xl mx-auto">
          {view === "list" && (
            <div>
              <h2 className="text-3xl font-bold mb-6">My Library</h2>
              {animeList.length === 0 ? (
                <div className="flex items-center justify-center py-12 rounded-lg border border-dashed border-border bg-muted/30">
                  <p className="text-muted-foreground text-lg">
                    Head to <span className="font-semibold text-foreground">Search</span> to add some!
                  </p>
                </div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {animeList.map((anime) => (
                    <Card key={anime.ID} className="flex flex-col hover:shadow-lg transition-shadow">
                      <CardHeader>
                        <CardTitle className="line-clamp-2">{anime.Title}</CardTitle>
                        <CardDescription>ID: {anime.ID}</CardDescription>
                      </CardHeader>
                      <CardContent className="flex-1 space-y-3">
                        <div className="space-y-2 text-sm">
                          <div className="flex justify-between">
                            <span className="text-muted-foreground">Episodes</span>
                            <span className="font-medium">{anime.Episodes || "—"}</span>
                          </div>
                          <div className="flex justify-between">
                            <span className="text-muted-foreground">Status</span>
                            <Badge variant="secondary">{anime.Status}</Badge>
                          </div>
                          <div className="flex justify-between">
                            <span className="text-muted-foreground">Added</span>
                            <span className="font-medium">
                              {anime.AddedAt
                                ? new Date(anime.AddedAt).toLocaleDateString()
                                : "—"}
                            </span>
                          </div>
                          {anime.TitleJP && (
                            <div className="pt-2 border-t border-border">
                              <p className="text-xs italic text-muted-foreground">
                                {anime.TitleJP}
                              </p>
                            </div>
                          )}
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              )}
            </div>
          )}

          {view === "search" && (
            <div>
              <h2 className="text-3xl font-bold mb-6">Search AniDB</h2>

              {/* Search Input */}
              <div className="mb-8 flex gap-2">
                <Input
                  type="text"
                  placeholder="Enter anime title..."
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && handleSearch()}
                  disabled={searching}
                  className="flex-1"
                />
                <Button
                  onClick={handleSearch}
                  disabled={searching || !query.trim()}
                >
                  {searching ? "Searching..." : "Search"}
                </Button>
              </div>

              {/* Results Grid */}
              {searchResults.length > 0 && (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {searchResults.map((result) => (
                    <Card key={result.AID} className="flex flex-col hover:shadow-lg transition-shadow">
                      <CardHeader>
                        <CardTitle className="line-clamp-2">{result.Title}</CardTitle>
                        <CardDescription>AID: {result.AID}</CardDescription>
                      </CardHeader>
                      <CardContent className="flex-1 flex flex-col justify-end gap-4">
                        <Button
                          onClick={() => handleAdd(result.AID, result.Title)}
                          disabled={adding === result.AID}
                          variant="default"
                          className="w-full"
                        >
                          {adding === result.AID ? "Adding..." : "Add to Library"}
                        </Button>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              )}

              {/* Empty States */}
              {!searching && query && searchResults.length === 0 && (
                <div className="flex items-center justify-center py-12 rounded-lg border border-dashed border-border bg-muted/30">
                  <p className="text-muted-foreground text-lg">
                    No anime found for <span className="font-semibold">"{query}"</span>
                  </p>
                </div>
              )}

              {!query && (
                <div className="flex items-center justify-center py-12 rounded-lg border border-dashed border-border bg-muted/30">
                  <p className="text-muted-foreground text-lg">
                    Enter an anime title to search AniDB
                  </p>
                </div>
              )}
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
