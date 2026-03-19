import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import { PlayCircle } from "lucide-react";
import { useRun } from "../lib/queries";
import { Skeleton } from "../components/ui/skeleton";
import { EmptyState } from "../components/EmptyState";
import { StatusPill } from "../components/StatusPill";
import { TimeAgo } from "../components/TimeAgo";

const LEVEL_STYLES: Record<string, { bar: string; text: string; bg: string }> = {
  info:  { bar: "bg-signal-500",  text: "text-signal-400",  bg: "" },
  warn:  { bar: "bg-amber-400",   text: "text-amber-300",   bg: "bg-amber-400/5" },
  error: { bar: "bg-rose-500",    text: "text-rose-300",    bg: "bg-rose-500/5" },
};

function levelStyle(level?: string) {
  return LEVEL_STYLES[level ?? ""] ?? LEVEL_STYLES.info;
}

function shortId(id: string) {
  return id.length > 8 ? id.slice(0, 8) : id;
}

export function ApplyPage() {
  const [params] = useSearchParams();
  const runId = params.get("run") ?? undefined;
  const { data: run } = useRun(runId);

  const events = useMemo(() => run?.events ?? [], [run?.events]);

  if (!runId) {
    return (
      <div className="rounded-2xl border border-ink-800/60 bg-ink-900/70 p-5">
        <EmptyState
          icon={PlayCircle}
          title="No active run"
          description="Start an apply from the Updates page to watch it here."
        />
      </div>
    );
  }

  if (!run) {
    return (
      <div className="space-y-4">
        <div className="rounded-2xl border border-ink-800/60 bg-ink-900/70 p-5">
          <Skeleton className="mb-3 h-6 w-48" />
          <Skeleton className="h-3 w-64" />
          <div className="mt-4 flex gap-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-6 w-20 rounded-full" />
            ))}
          </div>
        </div>
        <div className="rounded-2xl border border-ink-800/60 bg-ink-900/70 p-5">
          <Skeleton className="mb-4 h-5 w-28" />
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-14 w-full rounded-lg" />
            ))}
          </div>
        </div>
      </div>
    );
  }

  const summary = run.summary;

  return (
    <div className="space-y-4">

      {/* ── Run header ──────────────────────────────────── */}
      <div className="rounded-2xl border border-ink-800/60 bg-ink-900/70 p-5">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div className="flex items-center gap-3">
              <span className="font-display text-lg font-semibold text-ink-100">
                Apply Run
              </span>
              <code className="rounded-md bg-ink-800 px-2 py-0.5 font-mono text-xs text-ink-400 ring-1 ring-ink-700/60">
                {shortId(run.id)}
              </code>
              <StatusPill status={run.status} />
            </div>
            <p className="mt-1 text-xs text-ink-500">
              Started <TimeAgo date={run.started_at} />
              {run.completed_at && (
                <> · Completed <TimeAgo date={run.completed_at} /></>
              )}
            </p>
          </div>
        </div>

        {/* Summary metrics */}
        <div className="mt-4 grid grid-cols-2 gap-2 sm:grid-cols-4">
          <div className="rounded-xl bg-emerald-500/10 px-3 py-2 ring-1 ring-emerald-500/20">
            <div className="text-lg font-semibold text-emerald-300">{summary.updates_applied}</div>
            <div className="text-xs text-ink-500">Applied</div>
          </div>
          <div className="rounded-xl bg-ink-800/60 px-3 py-2 ring-1 ring-ink-700/40">
            <div className="text-lg font-semibold text-ink-300">{summary.updates_skipped}</div>
            <div className="text-xs text-ink-500">Skipped</div>
          </div>
          <div className={`rounded-xl px-3 py-2 ring-1 ${summary.updates_failed > 0 ? "bg-rose-500/10 ring-rose-500/20" : "bg-ink-800/60 ring-ink-700/40"}`}>
            <div className={`text-lg font-semibold ${summary.updates_failed > 0 ? "text-rose-300" : "text-ink-300"}`}>
              {summary.updates_failed}
            </div>
            <div className="text-xs text-ink-500">Failed</div>
          </div>
          <div className={`rounded-xl px-3 py-2 ring-1 ${summary.rollbacks > 0 ? "bg-amber-400/10 ring-amber-400/20" : "bg-ink-800/60 ring-ink-700/40"}`}>
            <div className={`text-lg font-semibold ${summary.rollbacks > 0 ? "text-amber-300" : "text-ink-300"}`}>
              {summary.rollbacks}
            </div>
            <div className="text-xs text-ink-500">Rollbacks</div>
          </div>
        </div>
      </div>

      {/* ── Event log ───────────────────────────────────── */}
      <div className="rounded-2xl border border-ink-800/60 bg-ink-900/70">
        <div className="flex items-center justify-between border-b border-ink-800/50 px-5 py-4">
          <div>
            <h2 className="font-display text-base font-semibold text-ink-100">Event Log</h2>
            <p className="text-xs text-ink-500">
              {run.status === "running" ? "Live — polling every second" : `${events.length} events`}
            </p>
          </div>
          {run.status === "running" && (
            <span className="flex items-center gap-1.5 text-xs text-signal-400">
              <span className="h-1.5 w-1.5 rounded-full bg-signal-400 animate-glow-pulse" />
              Live
            </span>
          )}
        </div>

        <div className="divide-y divide-ink-800/30 font-mono text-xs">
          {events.length === 0 && (
            <div className="px-5 py-6 text-center text-ink-500">No events yet.</div>
          )}
          {events.map((event, index) => {
            const ls = levelStyle(event.level);
            return (
              <div
                key={`${event.ts}-${index}`}
                className={`flex gap-3 px-5 py-2.5 ${ls.bg}`}
              >
                {/* Level bar */}
                <div className={`mt-0.5 w-0.5 shrink-0 rounded-full ${ls.bar} self-stretch`} />

                {/* Content */}
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
                    {event.step && (
                      <span className={`font-semibold uppercase tracking-wider ${ls.text}`}>
                        {event.step}
                      </span>
                    )}
                    {event.target && (
                      <span className="text-ink-500">
                        {event.target}{event.service ? `/${event.service}` : ""}
                      </span>
                    )}
                    <TimeAgo date={event.ts} className="ml-auto shrink-0 text-ink-700" />
                  </div>
                  <div className="mt-0.5 font-sans text-sm text-ink-200">{event.message}</div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
