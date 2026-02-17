import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import { PlayCircle } from "lucide-react";
import { useRun } from "../lib/queries";
import { Card, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Skeleton } from "../components/ui/skeleton";
import { EmptyState } from "../components/EmptyState";
import { StatusPill } from "../components/StatusPill";
import { TimeAgo } from "../components/TimeAgo";

export function ApplyPage() {
  const [params] = useSearchParams();
  const runId = params.get("run") ?? undefined;
  const { data: run } = useRun(runId);

  const events = useMemo(() => run?.events ?? [], [run?.events]);

  if (!runId) {
    return (
      <Card>
        <EmptyState
          icon={PlayCircle}
          title="No active run"
          description="Start an apply from the Plan page to watch it here."
        />
      </Card>
    );
  }

  if (!run) {
    return (
      <div className="space-y-6">
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-48" />
            <Skeleton className="h-3 w-64" />
          </CardHeader>
          <div className="flex flex-wrap gap-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-6 w-24 rounded-full" />
            ))}
          </div>
        </Card>
        <Card>
          <CardHeader>
            <Skeleton className="h-5 w-28" />
            <Skeleton className="h-3 w-56" />
          </CardHeader>
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-20 w-full rounded-lg" />
            ))}
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-3">
            Apply Run {run.id} <StatusPill status={run.status} />
          </CardTitle>
          <CardDescription>
            Started <TimeAgo date={run.started_at} /> · Completed{" "}
            {run.completed_at ? <TimeAgo date={run.completed_at} /> : "-"}
          </CardDescription>
        </CardHeader>
        <div className="flex flex-wrap gap-3 text-sm text-ink-200">
          <Badge variant="success">Applied: {run.summary.updates_applied}</Badge>
          <Badge variant="warning">Skipped: {run.summary.updates_skipped}</Badge>
          <Badge variant="danger">Failed: {run.summary.updates_failed}</Badge>
          <Badge variant="muted">Rollbacks: {run.summary.rollbacks}</Badge>
        </div>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Live Events</CardTitle>
          <CardDescription>Polling every second while the run is active.</CardDescription>
        </CardHeader>
        <div className="space-y-3">
          {events.map((event, index) => (
            <div key={`${event.ts}-${index}`} className="rounded-lg border border-ink-800/60 bg-ink-900/60 p-3">
              <div className="flex items-center justify-between text-xs text-ink-400">
                <span>{event.step ?? "event"}</span>
                <TimeAgo date={event.ts} className="text-xs text-ink-400" />
              </div>
              <div className="mt-2 text-sm text-ink-100">
                {event.message}
                {event.target && (
                  <span className="text-ink-400"> · {event.target}</span>
                )}
                {event.service && (
                  <span className="text-ink-400">/{event.service}</span>
                )}
              </div>
            </div>
          ))}
          {events.length === 0 && <div className="text-sm text-ink-400">No events yet.</div>}
        </div>
      </Card>
    </div>
  );
}
