import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { API_URL, resolvePosterUrl } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ArrowLeft, Trash2, ExternalLink, ChevronLeft, ChevronRight, Bell, Info } from "lucide-react";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import type { MediaDetails, Episode } from "@/types/media";
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

export function MediaDetailPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const [media, setMedia] = useState<MediaDetails | null>(null);
    const [loading, setLoading] = useState(true);
    const [monitoredItems, setMonitoredItems] = useState<any[]>([]);
    const [rangeInput, setRangeInput] = useState("");
    const [monitorEntireSeason, setMonitorEntireSeason] = useState(false);

    useEffect(() => {
        fetchMedia();
    }, [id]);

    useEffect(() => {
        if (media) {
            fetchMonitoredItems();
        }
    }, [media]);

    const fetchMonitoredItems = async () => {
        try {
            const response = await fetch(`${API_URL}/api/library/${id}/monitored`);
            if (response.ok) {
                const data = await response.json();
                setMonitoredItems(data || []);
                const isSeasonMonitored = data.some((m: any) => m.is_season && m.season === 1);
                setMonitorEntireSeason(isSeasonMonitored);
            }
        } catch (error) {
            console.error("Failed to fetch monitored items", error);
        }
    };

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


    const deleteMedia = async () => {
        if (!media) return;
        try {
            const response = await fetch(`${API_URL}/api/library/${id}`, {
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

    const parseRange = (input: string): number[] => {
        const nums = new Set<number>();
        const parts = input.split(',').map(p => p.trim());

        for (const part of parts) {
            if (part.includes('-')) {
                const [start, end] = part.split('-').map(p => parseInt(p.trim()));
                if (!isNaN(start) && !isNaN(end)) {
                    for (let i = Math.min(start, end); i <= Math.max(start, end); i++) {
                        nums.add(i);
                    }
                }
            } else {
                const val = parseInt(part);
                if (!isNaN(val)) nums.add(val);
            }
        }
        return Array.from(nums).sort((a, b) => a - b);
    };

    const handleBulkMonitor = async () => {
        let epNumbers: number[] = [];
        if (monitorEntireSeason) {
            if (!media?.episodes) return;
            epNumbers = media.episodes.map(e => parseInt(e.ep_no)).filter(n => !isNaN(n));

            const bulk = epNumbers.map((num: number) => {
                const epInfo = media?.episodes?.find(e => parseInt(e.ep_no) === num);
                return {
                    library_id: Number(id),
                    title: media?.title,
                    episode_title: epInfo?.title || `Episode ${num}`,
                    season: 1,
                    episode_number: num,
                    is_episode: true,
                    anidb_id: epInfo?.anidb_id || "",
                };
            });

            bulk.push({
                library_id: Number(id),
                title: media?.title,
                episode_title: "",
                season: 1,
                episode_number: 0,
                is_episode: false,
                is_season: true,
                anidb_id: String(media?.aid || ""),
            } as any);

            try {
                const response = await fetch(`${API_URL}/api/monitor/bulk`, {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify(bulk),
                });
                if (response.ok) {
                    toast.success("Season monitoring applied");
                    fetchMonitoredItems();
                    setRangeInput("");
                }
            } catch (error) {
                toast.error("Failed to apply monitor settings");
            }
        } else {
            // Unmonitoring or specific range
            if (rangeInput.trim()) {
                epNumbers = parseRange(rangeInput);
                const bulk = epNumbers.map((num: number) => {
                    const epInfo = media?.episodes?.find(e => parseInt(e.ep_no) === num);
                    return {
                        library_id: Number(id),
                        title: media?.title,
                        episode_title: epInfo?.title || `Episode ${num}`,
                        season: 1,
                        episode_number: num,
                        is_episode: true,
                        anidb_id: epInfo?.anidb_id || "",
                    };
                });

                try {
                    const response = await fetch(`${API_URL}/api/monitor/bulk`, {
                        method: "POST",
                        headers: { "Content-Type": "application/json" },
                        body: JSON.stringify(bulk),
                    });
                    if (response.ok) {
                        toast.success("Episode monitoring applied");
                        fetchMonitoredItems();
                        setRangeInput("");
                    }
                } catch (error) {
                    toast.error("Failed to apply range monitor");
                }
            } else {
                // If checkbox is unchecked and range is empty, unmonitor the whole season?
                // This is safer to do via a specific unmonitor call if they previously had it checked.
                const wasSeasonMonitored = monitoredItems.some((m: any) => m.is_season && m.season === 1);
                if (wasSeasonMonitored) {
                    try {
                        const response = await fetch(`${API_URL}/api/unmonitor/season`, {
                            method: "POST",
                            headers: { "Content-Type": "application/json" },
                            body: JSON.stringify({
                                library_id: Number(id),
                                season: 1,
                            }),
                        });
                        if (response.ok) {
                            toast.success("Stopped monitoring entire season");
                            fetchMonitoredItems();
                        }
                    } catch (error) {
                        toast.error("Failed to unmonitor season");
                    }
                } else {
                    toast.error("Please enter a range or select entire season");
                }
            }
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
                    <p className="text-muted-foreground">{media.alternate_titles !== media.title ? media.alternate_titles : ""}</p>
                </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
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
                                <p className="text-sm text-muted-foreground font-medium">AniDB ID</p>
                                <p className="text-sm">{media.aid}</p>
                            </div>
                            {media.total_episodes > 0 && (
                                <div className="space-y-1">
                                    <p className="text-sm text-muted-foreground font-medium">Episodes</p>
                                    <p className="text-sm">{media.total_episodes}</p>
                                </div>
                            )}
                            {media.total_seasons > 0 && (
                                <div className="space-y-1">
                                    <p className="text-sm text-muted-foreground font-medium">Seasons</p>
                                    <p className="text-sm">{media.total_seasons}</p>
                                </div>
                            )}
                        </CardContent>
                    </Card>

                    <Card className="border-primary/20 bg-primary/5">
                        <CardHeader className="pb-3">
                            <CardTitle className="flex items-center gap-2">
                                <Bell className="size-5 text-primary" />
                                Monitor Anime
                            </CardTitle>
                        </CardHeader>
                        <CardContent className="space-y-6">
                            <div className="flex items-center justify-between space-x-2">
                                <div className="space-y-0.5">
                                    <Label className="text-sm font-semibold">Monitor Entire Season</Label>
                                    <p className="text-xs text-muted-foreground">Automatically track all episodes in this season</p>
                                </div>
                                <Checkbox
                                    id="monitor-season"
                                    checked={monitorEntireSeason}
                                    onCheckedChange={(val) => setMonitorEntireSeason(!!val)}
                                />
                            </div>

                            <div className="space-y-3 pt-2">
                                <Label className="text-sm font-semibold">Monitor Specific Episodes (Ranges)</Label>
                                <Input
                                    placeholder="ex: 1-5, 8, 10-12"
                                    value={rangeInput}
                                    onChange={(e) => setRangeInput(e.target.value)}
                                    className="bg-background"
                                />
                                <Button className="w-full" onClick={handleBulkMonitor}>Apply Changes</Button>
                                <p className="text-[10px] text-muted-foreground flex items-center gap-1">
                                    <Info className="size-3" />
                                    Use comma separated numbers or ranges (1-10, 15, 20-25)
                                </p>
                            </div>
                        </CardContent>
                    </Card>

                    {media.episodes && media.episodes.length > 0 && (
                        <Card>
                            <CardHeader>
                                <CardTitle>Episodes</CardTitle>
                            </CardHeader>
                            <CardContent>
                                <EpisodeTable
                                    episodes={media.episodes}
                                    monitoredItems={monitoredItems}
                                />
                            </CardContent>
                        </Card>
                    )}

                    <div className="flex gap-4">

                        <Button variant="outline" asChild>
                            <a href={`https://anidb.net/anime/${media.aid}`} target="_blank" rel="noopener noreferrer">
                                <ExternalLink className="mr-2 h-4 w-4" /> View on AniDB
                            </a>
                        </Button>
                        <AlertDialog>
                            <AlertDialogTrigger asChild>
                                <Button variant="destructive">
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


                </div>


                <div className="space-y-6">
                    <Card className="overflow-hidden border-none shadow-xl">
                        <img
                            src={resolvePosterUrl(media.poster_url)}
                            alt={media.title}
                            className="w-full aspect-[2/3] object-cover"
                        />
                    </Card>



                </div>


            </div>
        </div>
    );

}

function EpisodeTable({
    episodes,
    monitoredItems
}: {
    episodes: Episode[],
    monitoredItems: any[]
}) {
    const [currentPage, setCurrentPage] = useState(1);
    const itemsPerPage = 10;
    const totalPages = Math.ceil(episodes.length / itemsPerPage);

    const startIndex = (currentPage - 1) * itemsPerPage;
    const currentEpisodes = episodes.slice(startIndex, startIndex + itemsPerPage);

    return (
        <div className="space-y-4">
            <div className="rounded-md border overflow-hidden">
                <table className="w-full text-sm">
                    <thead className="bg-muted/50 border-b font-medium">
                        <tr>
                            <th className="px-4 py-3 text-left w-16">No.</th>
                            <th className="px-4 py-3 text-left">Title</th>
                            <th className="px-4 py-3 text-left w-24">Type</th>
                            <th className="px-4 py-3 text-left w-32">Availability</th>
                            <th className="px-4 py-3 text-left w-32">Status</th>
                            <th className="px-4 py-3 text-right w-12">AniDB</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y">
                        {currentEpisodes.map((episode) => (
                            <tr key={episode.ID} className="hover:bg-muted/30 transition-colors">
                                <td className="px-4 py-3 font-medium">{episode.ep_no}</td>
                                <td className="px-4 py-3">{episode.title}</td>
                                <td className="px-4 py-3">
                                    <span className={`text-[10px] uppercase font-bold px-2 py-0.5 rounded-full ${episode.type === 1 ? "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400" :
                                            episode.type === 2 ? "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400" :
                                                episode.type === 3 ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" :
                                                    episode.type === 4 ? "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400" :
                                                        episode.type === 5 ? "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400" :
                                                            "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400"
                                        }`}>
                                        {episode.type === 1 ? "Regular" :
                                            episode.type === 2 ? "Special" :
                                                episode.type === 3 ? "Credit" :
                                                    episode.type === 4 ? "Trailer" :
                                                        episode.type === 5 ? "Parody" : "Other"}
                                    </span>
                                </td>
                                <td className="px-4 py-3">
                                    <span className="text-[10px] uppercase font-bold px-2 py-0.5 rounded-full bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400">
                                        Unavailable
                                    </span>
                                </td>
                                <td className="px-4 py-3">
                                    {monitoredItems.some(m => m.anidb_id === episode.anidb_id && m.is_episode) ? (
                                        <span className="text-[10px] uppercase font-bold px-2 py-0.5 rounded-full bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400">
                                            Monitored
                                        </span>
                                    ) : (
                                        <span className="text-[10px] uppercase font-bold px-2 py-0.5 rounded-full bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400">
                                            Not Monitored
                                        </span>
                                    )}
                                </td>
                                <td className="px-4 py-3 text-right">
                                    <a
                                        href={`https://anidb.net/episode/${episode.anidb_id}`}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        className="text-muted-foreground hover:text-primary transition-colors inline-flex items-center"
                                        title="View on AniDB"
                                    >
                                        <ExternalLink className="size-4" />
                                        <span className="sr-only">View on AniDB</span>
                                    </a>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>

            {totalPages > 1 && (
                <div className="flex items-center justify-between">
                    <p className="text-sm text-muted-foreground">
                        Showing {startIndex + 1} to {Math.min(startIndex + itemsPerPage, episodes.length)} of {episodes.length} episodes
                    </p>
                    <div className="flex items-center space-x-2">
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setCurrentPage(prev => Math.max(1, prev - 1))}
                            disabled={currentPage === 1}
                        >
                            <ChevronLeft className="h-4 w-4 mr-1" /> Previous
                        </Button>
                        <span className="text-sm font-medium">
                            Page {currentPage} of {totalPages}
                        </span>
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setCurrentPage(prev => Math.min(totalPages, prev + 1))}
                            disabled={currentPage === totalPages}
                        >
                            Next <ChevronRight className="h-4 w-4 ml-1" />
                        </Button>
                    </div>
                </div>
            )}
        </div>
    );
}

