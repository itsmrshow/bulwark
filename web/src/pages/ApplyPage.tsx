import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import { useRun } from "../lib/queries";
import { Card, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { StatusPill } from "../components/StatusPill";

function formatDate(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}

export function ApplyPage() {
  const [params] = useSearchParams();
  const runId = params.get("run") ?? undefined;
  const { data: run } = useRun(runId);

  const events = useMemo(() => run?.events ?? [], [run?.events]);

  if (!runId) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>No active run</CardTitle>
          <CardDescription>Start an apply from the Plan page to watch it here.</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  if (!run) {
    return <div className="text-ink-300">Loading run...</div>;
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-3">
            Apply Run {run.id} <StatusPill status={run.status} />
          </CardTitle>
          <CardDescription>
            Started {formatDate(run.started_at)} · Completed {formatDate(run.completed_at)}
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
                <span>{formatDate(event.ts)}</span>
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
