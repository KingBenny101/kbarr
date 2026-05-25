"use client"

import { useState } from "react"
import { Search, ExternalLink } from "lucide-react"
import { Button } from "@/components/ui/button"
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from "@/components/ui/input-group"
import { Card, CardDescription, CardHeader, CardTitle, CardFooter } from "@/components/ui/card"
import { API_URL } from "@/lib/api"
import { showToast } from "@/lib/utils"
import type { Media } from "@/types/media"

import {
    Pagination,
    PaginationContent,
    PaginationItem,
    PaginationNext,
    PaginationPrevious,
} from "@/components/ui/pagination"



interface SearchPageProps {
    // onMediaAdded?: () => void
}

export function SearchPage({ }: SearchPageProps) {
    const [query, setQuery] = useState<string>("")
    const [Medias, setMedias] = useState<Media[]>([])
    const [searching, setSearching] = useState<boolean>(false)
    const [adding, setAdding] = useState<number | null>(null)
    const [currentPage, setCurrentPage] = useState(1)
    const itemsPerPage = 12

    const totalPages = Math.ceil(Medias.length / itemsPerPage)
    const paginatedResults = Medias.slice(
        (currentPage - 1) * itemsPerPage,
        currentPage * itemsPerPage
    )

    const handleSearch = async (): Promise<void> => {
        if (!query.trim()) return
        setCurrentPage(1)
        setSearching(true)
        try {
            const res = await fetch(`${API_URL}/api/search?q=${encodeURIComponent(query)}`)
            const data = (await res.json()) as Media[]
            setMedias(data || [])
        } catch (err) {
            console.error("Search failed:", err)
        } finally {
            setSearching(false)
        }
    }

    const handleAdd = async (result: Media): Promise<void> => {
        setAdding(result.aid)
        try {
            const res = await fetch(`${API_URL}/api/library`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(result)
            })
            if (res.ok) {
                const data = await res.json()
                showToast(data.message, "success")
                // Mark as added in the local list
                setMedias(prev => prev.map(m => m.aid === result.aid ? { ...m, added: true } : m))
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
                    placeholder="Search for anime..."
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

            {Medias.length > 0 && (
                <div className="flex flex-col gap-6">
                    <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
                        {paginatedResults.map((result) => (
                            <Card key={result.aid} className="group">
                                <CardHeader className="p-4 pb-2">
                                    <div className="flex justify-between gap-4">
                                        <div className="space-y-1 flex-1">
                                            <CardTitle className="text-sm line-clamp-1 leading-tight">{result.title}</CardTitle>
                                            <CardDescription className="text-xs flex gap-2 items-center">
                                                <span>ID: {result.aid}</span>
                                            </CardDescription>
                                        </div>
                                        <Button
                                            variant="ghost"
                                            size="icon-sm"
                                            className="text-muted-foreground hover:text-primary shrink-0"
                                            asChild
                                        >
                                            <a href={`https://anidb.net/anime/${result.aid}`} target="_blank" rel="noopener noreferrer">
                                                <ExternalLink className="size-4" />
                                            </a>
                                        </Button>
                                    </div>
                                </CardHeader>
                                <CardFooter className="p-4 pt-2 mt-auto">
                                    <Button
                                        size="sm"
                                        variant="secondary"
                                        onClick={() => handleAdd(result)}
                                        disabled={adding === result.aid || result.added}
                                        className="w-full text-xs font-semibold"
                                    >
                                        {adding === result.aid ? "Adding..." : result.added ? "Added" : "Add to Library"}
                                    </Button>
                                </CardFooter>
                            </Card>
                        ))}
                    </div>

                    <Pagination className="mt-4">
                        <PaginationContent>
                            <PaginationItem>
                                <PaginationPrevious
                                    onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                                    className={currentPage === 1 ? "pointer-events-none opacity-50" : "cursor-pointer"}
                                />
                            </PaginationItem>

                            <PaginationItem>
                                <span className="text-sm text-muted-foreground px-4">
                                    Page {totalPages === 0 ? 0 : currentPage} of {totalPages}
                                </span>
                            </PaginationItem>

                            <PaginationItem>
                                <PaginationNext
                                    onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                                    className={currentPage === totalPages ? "pointer-events-none opacity-50" : "cursor-pointer"}
                                />
                            </PaginationItem>
                        </PaginationContent>
                    </Pagination>
                </div>
            )}

            {!searching && query && Medias.length === 0 && (
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
