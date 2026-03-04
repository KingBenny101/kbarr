import { useState, useEffect } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger
} from "@/components/ui/alert-dialog"
import { Trash2 } from "lucide-react"
import { API_URL } from "@/lib/api"
import { toast } from "sonner"

interface Media {
    ID: number
    Title: string
    TitleJP?: string
    Episodes?: number
    Status: string
    AddedAt?: string
}


export function LibraryPage() {
    const [mediaList, setMediaList] = useState<Media[]>([])

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

    const handleDelete = async (id: number) => {
        try {
            const res = await fetch(`${API_URL}/api/library/${id}`, {
                method: 'DELETE',
            })
            if (res.ok) {
                toast.success("Media removed from library")
                fetchList()
            } else {
                toast.error("Failed to remove media")
            }
        } catch (err) {
            console.error("Failed to delete media:", err)
            toast.error("An error occurred while deleting")
        }
    }

    return (
        <div>
            {mediaList.length === 0 ? (
                <p className="text-center py-8">No media added yet</p>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {mediaList.map((media) => (
                        <Card key={media.ID} className="flex flex-col">
                            <CardHeader className="flex-row items-start justify-between space-y-0">
                                <div className="space-y-1">
                                    <CardTitle>{media.Title}</CardTitle>
                                    <CardDescription>ID: {media.ID}</CardDescription>
                                </div>
                                <AlertDialog>
                                    <AlertDialogTrigger asChild>
                                        <Button variant="ghost" size="icon" className="text-destructive hover:text-destructive hover:bg-destructive/10">
                                            <Trash2 className="h-4 w-4" />
                                        </Button>
                                    </AlertDialogTrigger>
                                    <AlertDialogContent>
                                        <AlertDialogHeader>
                                            <AlertDialogTitle>Are you sure?</AlertDialogTitle>
                                            <AlertDialogDescription>
                                                This will permanently remove <strong>{media.Title}</strong> from your library.
                                            </AlertDialogDescription>
                                        </AlertDialogHeader>
                                        <AlertDialogFooter>
                                            <AlertDialogCancel>Cancel</AlertDialogCancel>
                                            <AlertDialogAction
                                                onClick={() => handleDelete(media.ID)}
                                                className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                                            >
                                                Delete
                                            </AlertDialogAction>
                                        </AlertDialogFooter>
                                    </AlertDialogContent>
                                </AlertDialog>
                            </CardHeader>
                            <CardContent className="text-sm space-y-2 flex-1">
                                <div>Episodes: {media.Episodes || "—"}</div>
                                <div>Status: <Badge variant="secondary">{media.Status}</Badge></div>
                                <div>Added: {media.AddedAt ? new Date(media.AddedAt).toLocaleDateString() : "—"}</div>
                                {media.TitleJP && <div className="italic text-xs">{media.TitleJP}</div>}
                            </CardContent>
                        </Card>
                    ))}
                </div>
            )}
        </div>
    )
}
