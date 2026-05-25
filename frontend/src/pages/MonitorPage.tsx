import { useEffect, useState } from 'react';
import { API_URL } from "@/lib/api";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Trash2, ExternalLink } from "lucide-react";
import { toast } from "sonner";
import { Link } from 'react-router-dom';
import { ChevronLeft, ChevronRight } from "lucide-react";

interface MonitorEntry {
  ID: number;
  CreatedAt: string;
  library_id: number;
  title: string;
  episode_title: string;
  season: number;
  episode_number: number;
  is_episode: boolean;
  is_season: boolean;
  anidb_id: string;
  status: string;
}

export function MonitorPage() {
  const [monitors, setMonitors] = useState<MonitorEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [currentPage, setCurrentPage] = useState(1);
  const itemsPerPage = 10;

  const fetchMonitors = async () => {
    try {
      const response = await fetch(`${API_URL}/api/monitor`);
      if (!response.ok) throw new Error("Failed to fetch monitored items");
      const data = await response.json();
      setMonitors(data || []);
    } catch (error) {
      console.error(error);
      toast.error("Failed to load monitored items");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMonitors();
  }, []);

  const handleDelete = async (id: number) => {
    try {
      const response = await fetch(`${API_URL}/api/monitor/${id}`, {
        method: "DELETE",
      });
      if (!response.ok) throw new Error("Failed to delete monitor entry");
      setMonitors(monitors.filter((m) => m.ID !== id));
      toast.success("Removed from monitor");
    } catch (error) {
      console.error(error);
      toast.error("Failed to remove item");
    }
  };

  const totalPages = Math.ceil(monitors.length / itemsPerPage);
  const startIndex = (currentPage - 1) * itemsPerPage;
  const currentMonitors = monitors.slice(startIndex, startIndex + itemsPerPage);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Monitored Items</h1>
        <p className="text-muted-foreground">
          Tracked seasons and episodes across your library.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Currently Monitoring</CardTitle>
          <CardDescription>
            You have {monitors.length} item{monitors.length !== 1 ? 's' : ''} being monitored.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-muted/50 border-b font-medium">
                <tr>
                  <th className="px-4 py-3 text-left">Anime</th>
                  <th className="px-4 py-3 text-left">Type</th>
                  <th className="px-4 py-3 text-left">Details</th>
                  <th className="px-4 py-3 text-left w-24">Status</th>
                  <th className="px-4 py-3 text-left w-24">AniDB</th>
                  <th className="px-4 py-3 text-right w-12"></th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {loading ? (
                  <tr>
                    <td colSpan={5} className="text-center py-8">
                      Loading...
                    </td>
                  </tr>
                ) : monitors.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="text-center py-8 text-muted-foreground">
                      No items monitored yet.
                    </td>
                  </tr>
                ) : (
                  currentMonitors.map((entry) => (
                    <tr key={entry.ID} className="hover:bg-muted/30 transition-colors">
                      <td className="px-4 py-3 font-medium">
                        <Link 
                          to={`/media/${entry.library_id}`}
                          className="hover:underline text-primary"
                        >
                          {entry.title}
                        </Link>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`text-[10px] uppercase font-bold px-2 py-0.5 rounded-full ${
                          entry.is_season ? "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400" : "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
                        }`}>
                          {entry.is_season ? "Season" : "Episode"}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {entry.is_season ? (
                          null
                        ) : (
                          `E${entry.episode_number}: ${entry.episode_title}`
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <span className="capitalize text-xs font-medium bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400 px-2 py-0.5 rounded-full">
                          {entry.status}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        {entry.anidb_id && (
                          <a
                            href={entry.is_season ? `https://anidb.net/anime/${entry.anidb_id}` : `https://anidb.net/episode/${entry.anidb_id}`}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="hover:text-primary transition-colors"
                          >
                            <ExternalLink className="size-4" />
                          </a>
                        )}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
                          onClick={() => handleDelete(entry.ID)}
                        >
                          <Trash2 className="size-4" />
                        </Button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4">
              <p className="text-xs text-muted-foreground">
                Showing {startIndex + 1} to {Math.min(startIndex + itemsPerPage, monitors.length)} of {monitors.length} items
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage(prev => Math.max(1, prev - 1))}
                  disabled={currentPage === 1}
                >
                  <ChevronLeft className="size-4 mr-1" />
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage(prev => Math.min(totalPages, prev + 1))}
                  disabled={currentPage === totalPages}
                >
                  Next
                  <ChevronRight className="size-4 ml-1" />
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
