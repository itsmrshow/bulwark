import {
  Activity,
  AlertTriangle,
  ArrowRight,
  Boxes,
  RefreshCw,
  RotateCcw
} from "lucide-react";
import { useOverview } from "../lib/queries";
import { Badge } from "../components/ui/badge";
import { Skeleton } from "../components/ui/skeleton";
import { TimeAgo } from "../components/TimeAgo";

const ACTION_COLORS: Record<string, string> = {
  updated:      "bg-emerald-500",
  failed:       "bg-rose-500",
  rolled_back:  "bg-amber-400",
  rollback:     "bg-amber-400",
  skip:         "bg-ink-600",
  complete:     "bg-emerald-500",
  start:        "bg-signal-500",
  plan:         "bg-ink-500",
};

function actionDot(action: string) {
  const color = ACTION_COLORS[action] ?? "bg-ink-600";
  return <span className={`mt-1 h-2 w-2 shrink-0 rounded-full ${color}`} />;
}

function StatCard({
  label,
  value,
  sub,
  icon: Icon,
  accentClass,
  glowClass,
}: {
  label: string;
  value: React.ReactNode;
  sub?: React.ReactNode;
  icon: React.ElementType;
  accentClass: string;
  glowClass: string;
}) {
  return (
    <div className={`relative overflow-hidden rounded-2xl border border-ink-800/60 bg-ink-900/70 p-5 ${glowClass}`}>
      {/* Background glow blob */}
      <div className="pointer-events-none absolute -right-4 -top-4 h-24 w-24 rounded-full opacity-10 blur-2xl" />
      {/* Icon badge */}
      <div className={`mb-4 inline-flex items-center justify-center rounded-xl p-2 ${accentClass}`}>
        <Icon className="h-4 w-4" />
      </div>
      {/* Value */}
      <div className="font-display text-4xl font-semibold tracking-tight text-ink-100">
        {value}
      </div>
      {/* Label */}
      <div className="mt-1 text-sm font-medium text-ink-400">{label}</div>
      {/* Sub */}
      {sub && <div className="mt-2">{sub}</div>}
    </div>
  );
}

export function OverviewPage() {
  const { data, isLoading, error } = useOverview();

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="rounded-2xl border border-ink-800/60 bg-ink-900/70 p-5">
              <Skeleton className="mb-4 h-9 w-9 rounded-xl" />
              <Skeleton className="h-10 w-16" />
              <Skeleton className="mt-1 h-4 w-28" />
            </div>
          ))}
        </div>
        <div className="rounded-2xl border border-ink-800/60 bg-ink-900/70 p-5">
          <Skeleton className="mb-4 h-5 w-32" />
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-9 w-full" />
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="flex items-center gap-2 rounded-xl border border-rose-500/30 bg-rose-500/10 p-4 text-sm text-rose-300">
        <AlertTriangle className="h-4 w-4 shrink-0" />
        Unable to load overview. Check the Bulwark API.
      </div>
    );
  }

  return (
    <div className="space-y-6">

      {/* ── Stat cards ───────────────────────────────────── */}
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label="Managed Targets"
          value={data.managed_targets}
          sub={
            <span className="text-xs text-ink-500">
              {data.managed_services} service{data.managed_services !== 1 ? "s" : ""} tracked
            </span>
          }
          icon={Boxes}
          accentClass="bg-signal-500/15 text-signal-400"
          glowClass=""
        />
        <StatCard
          label="Updates Available"
          value={
            <span className={data.updates_available > 0 ? "text-ember-500" : "text-ink-100"}>
              {data.updates_available}
            </span>
          }
          sub={
            data.updates_available > 0 ? (
              <span className="inline-flex items-center gap-1 text-xs text-ember-500/80">
                <ArrowRight className="h-3 w-3" />
                Plan before apply
              </span>
            ) : (
              <span className="text-xs text-emerald-400/80">All up to date</span>
            )
          }
          icon={RefreshCw}
          accentClass={
            data.updates_available > 0
              ? "bg-ember-500/15 text-ember-400"
              : "bg-emerald-500/15 text-emerald-400"
          }
          glowClass=""
        />
        <StatCard
          label="Last Run"
          value={
            data.last_run ? (
              <span className="text-2xl">
                <TimeAgo date={data.last_run.completed_at} />
              </span>
            ) : (
              <span className="text-2xl text-ink-500">—</span>
            )
          }
          sub={
            data.last_run ? (
              <Badge variant={data.last_run.status === "failed" ? "danger" : "success"}>
                {data.last_run.status}
              </Badge>
            ) : (
              <span className="text-xs text-ink-500">No runs yet</span>
            )
          }
          icon={Activity}
          accentClass="bg-ink-800/80 text-ink-400"
          glowClass=""
        />
        <StatCard
          label="Failures / Rollbacks"
          value={
            <span className={data.failures > 0 || data.rollbacks > 0 ? "text-rose-300" : "text-ink-100"}>
              {data.failures} / {data.rollbacks}
            </span>
          }
          sub={<span className="text-xs text-ink-500">From recent history</span>}
          icon={RotateCcw}
          accentClass={
            data.failures > 0
              ? "bg-rose-500/15 text-rose-400"
              : "bg-ink-800/80 text-ink-400"
          }
          glowClass=""
        />
      </div>

      {/* ── Activity stream ───────────────────────────────── */}
      <div className="rounded-2xl border border-ink-800/60 bg-ink-900/70">
        <div className="flex items-center justify-between border-b border-ink-800/50 px-5 py-4">
          <div>
            <h2 className="font-display text-base font-semibold text-ink-100">Activity</h2>
            <p className="text-xs text-ink-500">Last 20 events</p>
          </div>
          <Activity className="h-4 w-4 text-ink-600" />
        </div>

        <div className="divide-y divide-ink-800/40">
          {data.activity.length === 0 && (
            <div className="px-5 py-8 text-center text-sm text-ink-500">
              No recent activity yet. Run an apply to see events here.
            </div>
          )}
          {data.activity.map((event, index) => (
            <div
              key={`${event.ts}-${index}`}
              className="flex items-start gap-3 px-5 py-3 hover:bg-ink-800/20 transition-colors"
            >
              {actionDot(event.action)}
              <div className="min-w-0 flex-1">
                <div className="flex items-baseline gap-2">
                  <span className="text-xs font-semibold uppercase tracking-wide text-ink-400">
                    {event.action}
                  </span>
                  {event.target && (
                    <span className="truncate font-mono text-xs text-ink-500">
                      {event.target}
                      {event.service ? `/${event.service}` : ""}
                    </span>
                  )}
                </div>
                <div className="mt-0.5 text-sm text-ink-200">{event.message}</div>
              </div>
              <TimeAgo date={event.ts} className="shrink-0 text-xs text-ink-600" />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
