import { useState, useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { Card, CardHeader, CardTitle } from "@/components/ui/card"
import { PlayCircle } from "lucide-react"
import { API_URL, resolvePosterUrl } from "@/lib/api"

import {
    Pagination,
    PaginationContent,
    PaginationItem,
    PaginationNext,
    PaginationPrevious,
} from "@/components/ui/pagination"

import type { Media } from "@/types/media"


export function LibraryPage() {
    const [Medias, setMedias] = useState<Media[]>([])
    const [currentPage, setCurrentPage] = useState(1)
    const itemsPerPage = 18

    const totalPages = Math.ceil(Medias.length / itemsPerPage)
    const paginatedResults = Medias.slice(
        (currentPage - 1) * itemsPerPage,
        currentPage * itemsPerPage
    )

    const navigate = useNavigate()

    useEffect(() => {
        fetchList()
    }, [])

    const fetchList = async (): Promise<void> => {
        try {
            const res = await fetch(`${API_URL}/api/library`)
            const data = (await res.json()) as Media[]
            setMedias(data || [])
        } catch (err) {
            console.error("Failed to fetch media list:", err)
        }
    }

    return (
        <div className="space-y-6">

            {Medias.length === 0 ? (
                <Card className="border-dashed flex flex-col items-center justify-center p-12 text-center space-y-4">
                    <div className="bg-muted rounded-full p-4">
                        <PlayCircle className="h-8 w-8 text-muted-foreground" />
                    </div>
                    <div className="space-y-1">
                        <p className="font-medium">No anime added yet</p>
                        <p className="text-sm text-muted-foreground">Go to search and add some anime!</p>
                    </div>
                </Card>
            ) : (
                <div className="flex flex-col gap-6">
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                        {paginatedResults.map((media) => (
                            <Card
                                key={media.ID}
                                className="group cursor-pointer hover:shadow-lg transition-all duration-200 border-border/50 overflow-hidden"
                                onClick={() => navigate(`/media/${media.ID}`)}
                            >
                                {media.poster_url ? (
                                    <div className="relative w-full h-48 bg-muted overflow-hidden">
                                        <img
                                            src={resolvePosterUrl(media.poster_url)}
                                            alt={media.title}
                                            className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-200"
                                        />
                                    </div>
                                ) : (
                                    <div className="w-full h-48 bg-muted flex items-center justify-center">
                                        <PlayCircle className="size-8 text-muted-foreground" />
                                    </div>
                                )}
                                <CardHeader className="py-3 px-4">
                                    <CardTitle className="text-sm font-semibold line-clamp-2 leading-tight">
                                        {media.title}
                                    </CardTitle>
                                </CardHeader>
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
        </div>
    )
}
