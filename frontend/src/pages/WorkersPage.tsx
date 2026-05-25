import { useEffect, useState } from 'react';
import { API_URL } from "@/lib/api";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Clock, Calendar } from "lucide-react";
import { toast } from "sonner";

interface WorkerStatus {
  name: string;
  last_run: string;
  next_run: string;
  running: boolean;
}

export function WorkersPage() {
  const [workers, setWorkers] = useState<WorkerStatus[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchWorkers = async () => {
    try {
      const response = await fetch(`${API_URL}/api/workers`);
      if (!response.ok) throw new Error("Failed to fetch worker statuses");
      const data = await response.json();
      setWorkers(data || []);
    } catch (error) {
      console.error(error);
      toast.error("Failed to load worker statuses");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchWorkers();
    const interval = setInterval(fetchWorkers, 10000); // refresh every 10s
    return () => clearInterval(interval);
  }, []);

  const formatDate = (dateStr: string) => {
    if (!dateStr || dateStr.startsWith('0001-01-01')) return "Never";
    const date = new Date(dateStr);
    return date.toLocaleString();
  };

  const getWorkerDisplayName = (name: string) => {
    switch (name) {
      case 'anidb': return 'AniDB Sync';
      default: return name;
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">System Workers</h1>
        <p className="text-muted-foreground">
          Monitor background tasks and scheduled synchronization.
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {loading ? (
          <div className="col-span-full text-center py-12 italic text-muted-foreground">
            Loading worker status...
          </div>
        ) : workers.length === 0 ? (
          <div className="col-span-full text-center py-12 text-muted-foreground border rounded-lg bg-muted/20">
            No workers active.
          </div>
        ) : (
          workers.map((worker) => (
            <Card key={worker.name} className="overflow-hidden">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  {getWorkerDisplayName(worker.name)}
                </CardTitle>
                <Badge variant={worker.running ? "default" : "secondary"}>
                  {worker.running ? "Active" : "Stopped"}
                </Badge>
              </CardHeader>
              <CardContent className="space-y-4 pt-4">
                <div className="flex items-center gap-2 text-sm">
                  <Clock className="size-4 text-muted-foreground" />
                  <div className="flex-1">
                    <p className="text-xs text-muted-foreground mb-1">Last Run</p>
                    <p className="font-medium">{formatDate(worker.last_run)}</p>
                  </div>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <Calendar className="size-4 text-muted-foreground" />
                  <div className="flex-1">
                    <p className="text-xs text-muted-foreground mb-1">Next Run</p>
                    <p className="font-medium">{formatDate(worker.next_run)}</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))
        )}
      </div>
    </div>
  );
}
