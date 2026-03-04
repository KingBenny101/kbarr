"use client"

import { useState } from "react"
import { Search } from "lucide-react"
import { Button } from "@/components/ui/button"
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from "@/components/ui/input-group"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { API_URL } from "@/lib/api"
import { showToast } from "@/lib/utils"

interface SearchResult {
    AID: number
    Title: string
}

interface SearchPageProps {
    onMediaAdded?: () => void
}

export function SearchPage({ onMediaAdded }: SearchPageProps) {
    const [query, setQuery] = useState<string>("")
    const [searchResults, setSearchResults] = useState<SearchResult[]>([])
    const [searching, setSearching] = useState<boolean>(false)
    const [adding, setAdding] = useState<number | null>(null)

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

    const handleAdd = async (aid: number): Promise<void> => {
        setAdding(aid)
        try {
            const res = await fetch(`${API_URL}/api/library`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ aid })
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

    return (
        <div>
            <InputGroup className="mb-6">
                <InputGroupInput
                    type="text"
                    placeholder="Search..."
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
                <div className="grid gap-4 grid-cols-3">
                    {searchResults.map((result) => (
                        <Card key={result.AID}>
                            <CardHeader>
                                <CardTitle>{result.Title}</CardTitle>
                                <CardDescription>AID: {result.AID}</CardDescription>
                            </CardHeader>
                            <CardContent>
                                <Button
                                    onClick={() => handleAdd(result.AID)}
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
