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
import { Trash2, Info } from "lucide-react";
import { toast } from "sonner";
import { Link } from 'react-router-dom';

interface QueueItem {
  ID: number;
  CreatedAt: string;
  library_id: number;
  title: string;
  episode_title: string;
  season: number;
  episode_number: number;
  is_episode: boolean;
  is_season: boolean;
  status: string;
}

export function SearchQueuePage() {
  const [queue, setQueue] = useState<QueueItem[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchQueue = async () => {
    try {
      const response = await fetch(`${API_URL}/api/search-queue`);
      if (!response.ok) throw new Error("Failed to fetch search queue");
      const data = await response.json();
      setQueue(data || []);
    } catch (error) {
      console.error(error);
      toast.error("Failed to load search queue");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchQueue();
    const interval = setInterval(fetchQueue, 5000); // refresh every 5s
    return () => clearInterval(interval);
  }, []);

  const handleDelete = async (id: number) => {
    try {
      const response = await fetch(`${API_URL}/api/search-queue/${id}`, {
        method: "DELETE",
      });
      if (!response.ok) throw new Error("Failed to delete queue entry");
      setQueue(queue.filter((item) => item.ID !== id));
      toast.success("Removed from queue");
    } catch (error) {
      console.error(error);
      toast.error("Failed to remove item");
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Search Queue</h1>
        <p className="text-muted-foreground">
          Pending Prowlarr searches for monitored content.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Queue Status</CardTitle>
          <CardDescription>
            {queue.length} item{queue.length !== 1 ? 's' : ' '} waiting for search.
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
                  <th className="px-4 py-3 text-left">Status</th>
                  <th className="px-4 py-3 text-right w-12"></th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {loading ? (
                  <tr>
                    <td colSpan={5} className="text-center py-8">
                      Loading queue...
                    </td>
                  </tr>
                ) : queue.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="text-center py-8 text-muted-foreground">
                      Search queue is empty.
                    </td>
                  </tr>
                ) : (
                  queue.map((item) => (
                    <tr key={item.ID} className="hover:bg-muted/30 transition-colors">
                      <td className="px-4 py-3 font-medium">
                        <Link 
                          to={`/media/${item.library_id}`}
                          className="hover:underline text-primary"
                        >
                          {item.title}
                        </Link>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`text-[10px] uppercase font-bold px-2 py-0.5 rounded-full ${
                          item.is_season ? "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400" : "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
                        }`}>
                          {item.is_season ? "Season" : "Episode"}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {item.is_season ? "" : `E${item.episode_number}: ${item.episode_title}`}
                      </td>
                      <td className="px-4 py-3">
                        <span className="capitalize text-xs font-medium bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400 px-2 py-0.5 rounded-full">
                          {item.status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
                          onClick={() => handleDelete(item.ID)}
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
        </CardContent>
      </Card>

      <div className="flex items-start gap-2 text-sm text-muted-foreground bg-muted/30 p-4 rounded-lg border border-dashed">
        <Info className="size-4 mt-0.5 shrink-0" />
        <p>
          Items are added to this queue when they are monitored. A background worker will pick up these items 
          and search for them on Prowlarr sequentially.
        </p>
      </div>
    </div>
  );
}
