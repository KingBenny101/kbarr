import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { API_URL } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { ArrowLeft, Trash2, ExternalLink, Calendar, Monitor, Layers, Search as SearchIcon } from "lucide-react";
import { toast } from "sonner";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from "@/components/ui/alert-dialog";

interface Media {
    id: number;
    title: string;
    title_original: string;
    description: string;
    status: string;
    type: string;
    episodes: number;
    seasons: number;
    year: number;
    cover_image: string;
    banner_image: string;
    external_id: string;
    source: string;
    monitored: boolean;
    added_at: string;
    updated_at: string;
}

export function MediaDetailPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const [media, setMedia] = useState<Media | null>(null);
    const [loading, setLoading] = useState(true);
    const [searching, setSearching] = useState(false);

    useEffect(() => {
        fetchMedia();
    }, [id]);

    const fetchMedia = async () => {
        try {
            const response = await fetch(`${API_URL}/api/library/${id}`);
            if (!response.ok) throw new Error("Failed to fetch media details");
            const data = await response.json();
            setMedia(data);
        } catch (error) {
            toast.error("Error loading media details");
            console.error(error);
        } finally {
            setLoading(false);
        }
    };

    const triggerSearch = async () => {
        if (!media) return;
        setSearching(true);
        try {
            const response = await fetch(`${API_URL}/api/library/${media.id}/search`, {
                method: "POST",
            });
            if (response.ok) {
                const data = await response.json();
                toast.success(`Found ${data.results_count} results for ${data.media_title}`);
                console.log("Search results:", data.results);
            } else if (response.status === 400) {
                toast.error("Prowlarr is not configured");
            } else {
                toast.error("Failed to trigger search");
            }
        } catch (error) {
            toast.error("Error triggering search");
            console.error("Search error:", error);
        } finally {
            setSearching(false);
        }
    };

    const toggleMonitor = async () => {
        if (!media) return;
        const newStatus = !media.monitored;
        try {
            const response = await fetch(`${API_URL}/api/library/${media.id}/monitor`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ monitored: newStatus }),
            });
            if (response.ok) {
                setMedia({ ...media, monitored: newStatus });
                toast.success(`Monitoring ${newStatus ? "enabled" : "disabled"}`);
            }
        } catch (error) {
            toast.error("Failed to update monitoring status");
        }
    };

    const deleteMedia = async () => {
        if (!media) return;
        try {
            const response = await fetch(`${API_URL}/api/library/${media.id}`, {
                method: "DELETE",
            });
            if (response.ok) {
                toast.success("Media deleted");
                navigate("/library");
            }
        } catch (error) {
            toast.error("Failed to delete media");
        }
    };

    if (loading) return <div className="p-8 text-center text-muted-foreground">Loading...</div>;
    if (!media) return <div className="p-8 text-center text-destructive font-bold">Media not found</div>;

    return (
        <div className="container mx-auto p-4 lg:p-8 space-y-8 animate-in fade-in duration-500">
            <div className="flex items-center space-x-4">
                <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
                    <ArrowLeft className="h-5 w-5" />
                </Button>
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">{media.title}</h1>
                    <p className="text-muted-foreground">{media.title_original !== media.title ? media.title_original : ""}</p>
                </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
                {/* Left Column: Poster & Quick Info */}
                <div className="space-y-6">
                    <Card className="overflow-hidden border-none shadow-xl">
                        <img
                            src={media.cover_image || "/placeholder.svg?height=600&width=400"}
                            alt={media.title}
                            className="w-full aspect-[2/3] object-cover"
                        />
                    </Card>

                    <Card>
                        <CardHeader>
                            <CardTitle className="text-sm font-medium">Monitoring & Status</CardTitle>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            <div className="flex items-center justify-between">
                                <div className="space-y-0.5">
                                    <Label htmlFor="monitored">Monitored</Label>
                                    <p className="text-xs text-muted-foreground">Automated scan for releases</p>
                                </div>
                                <Switch
                                    id="monitored"
                                    checked={media.monitored}
                                    onCheckedChange={toggleMonitor}
                                />
                            </div>
                            <Separator />
                            <div className="flex justify-between items-center">
                                <span className="text-sm text-muted-foreground flex items-center gap-2">
                                    <Layers className="h-4 w-4" /> Status
                                </span>
                                <Badge variant="outline">{media.status}</Badge>
                            </div>
                            <div className="flex justify-between items-center">
                                <span className="text-sm text-muted-foreground flex items-center gap-2">
                                    <Monitor className="h-4 w-4" /> Type
                                </span>
                                <Badge variant="secondary" className="capitalize">{media.type}</Badge>
                            </div>
                            {media.year > 0 && (
                                <div className="flex justify-between items-center">
                                    <span className="text-sm text-muted-foreground flex items-center gap-2">
                                        <Calendar className="h-4 w-4" /> Year
                                    </span>
                                    <span className="text-sm font-medium">{media.year}</span>
                                </div>
                            )}
                        </CardContent>
                    </Card>

                    <AlertDialog>
                        <AlertDialogTrigger asChild>
                            <Button variant="destructive" className="w-full">
                                <Trash2 className="mr-2 h-4 w-4" /> Delete Media
                            </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                            <AlertDialogHeader>
                                <AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
                                <AlertDialogDescription>
                                    This will remove <strong>{media.title}</strong> from your library.
                                    This action cannot be undone.
                                </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction onClick={deleteMedia} className="bg-destructive hover:bg-destructive/90">
                                    Delete
                                </AlertDialogAction>
                            </AlertDialogFooter>
                        </AlertDialogContent>
                    </AlertDialog>
                </div>

                {/* Right Column: Details & Description */}
                <div className="md:col-span-2 space-y-6">
                    <Card>
                        <CardHeader>
                            <CardTitle>Overview</CardTitle>
                        </CardHeader>
                        <CardContent>
                            <p className="text-lg leading-relaxed text-muted-foreground">
                                {media.description || "No description available."}
                            </p>
                        </CardContent>
                    </Card>

                    <Card>
                        <CardHeader>
                            <CardTitle>Information</CardTitle>
                        </CardHeader>
                        <CardContent className="grid grid-cols-2 md:grid-cols-4 gap-4">
                            <div className="space-y-1">
                                <p className="text-sm text-muted-foreground font-medium">Source</p>
                                <p className="text-sm uppercase">{media.source}</p>
                            </div>
                            <div className="space-y-1">
                                <p className="text-sm text-muted-foreground font-medium">External ID</p>
                                <p className="text-sm">{media.external_id}</p>
                            </div>
                            {media.episodes > 0 && (
                                <div className="space-y-1">
                                    <p className="text-sm text-muted-foreground font-medium">Episodes</p>
                                    <p className="text-sm">{media.episodes}</p>
                                </div>
                            )}
                            {media.seasons > 0 && (
                                <div className="space-y-1">
                                    <p className="text-sm text-muted-foreground font-medium">Seasons</p>
                                    <p className="text-sm">{media.seasons}</p>
                                </div>
                            )}
                        </CardContent>
                    </Card>

                    <div className="flex gap-4 flex-wrap">
                        <Button
                            variant="default"
                            onClick={triggerSearch}
                            disabled={searching || !media.monitored}
                        >
                            <SearchIcon className="mr-2 h-4 w-4" />
                            {searching ? "Searching..." : "Search Now"}
                        </Button>
                        {media.source === 'anidb' && (
                            <Button variant="outline" asChild>
                                <a href={`https://anidb.net/anime/${media.external_id}`} target="_blank" rel="noopener noreferrer">
                                    <ExternalLink className="mr-2 h-4 w-4" /> View on AniDB
                                </a>
                            </Button>
                        )}
                        {media.source === 'tmdb' && (
                            <Button variant="outline" asChild>
                                <a href={`https://www.themoviedb.org/${media.type}/${media.external_id}`} target="_blank" rel="noopener noreferrer">
                                    <ExternalLink className="mr-2 h-4 w-4" /> View on TMDB
                                </a>
                            </Button>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}

