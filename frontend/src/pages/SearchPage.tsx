import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

const API = "http://localhost:8282"

interface SearchResult {
    AID: number
    Title: string
}

interface SearchPageProps {
    onAdd: (aid: number, title: string) => Promise<void>
    adding: number | null
}

export function SearchPage({ onAdd, adding }: SearchPageProps) {
    const [query, setQuery] = useState<string>("")
    const [searchResults, setSearchResults] = useState<SearchResult[]>([])
    const [searching, setSearching] = useState<boolean>(false)

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

    return (
        <div>
            <div className="flex gap-2 mb-6">
                <Input
                    type="text"
                    placeholder="Search anime..."
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && handleSearch()}
                    disabled={searching}
                />
                <Button onClick={handleSearch} disabled={searching || !query.trim()}>
                    {searching ? "..." : "Search"}
                </Button>
            </div>

            {searchResults.length > 0 && (
                <div className="grid gap-4 grid-cols-3">
                    {searchResults.map((result) => (
                        <Card key={result.AID}>
                            <CardHeader>
                                <CardTitle>{result.Title}</CardTitle>
                                <CardDescription>AID: {result.AID}</CardDescription>
                            </CardHeader>
                            <CardContent>
                                <Button
                                    onClick={() => onAdd(result.AID, result.Title)}
                                    disabled={adding === result.AID}
                                    className="w-full"
                                >
                                    {adding === result.AID ? "Adding..." : "Add"}
                                </Button>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            )}

            {!searching && query && searchResults.length === 0 && (
                <p className="text-center py-8">No results for "{query}"</p>
            )}

            {!query && <p className="text-center py-8">Press search button</p>}
        </div>
    )
}
