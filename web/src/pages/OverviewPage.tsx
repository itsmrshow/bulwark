import { useOverview } from "../lib/queries";
import { Card, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Skeleton } from "../components/ui/skeleton";
import { TimeAgo } from "../components/TimeAgo";

export function OverviewPage() {
  const { data, isLoading, error } = useOverview();

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-4 w-32" />
              </CardHeader>
              <Skeleton className="h-10 w-20" />
              <Skeleton className="mt-2 h-3 w-40" />
            </Card>
          ))}
        </div>
        <Card>
          <CardHeader>
            <Skeleton className="h-5 w-36" />
            <Skeleton className="h-3 w-28" />
          </CardHeader>
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        </Card>
      </div>
    );
  }

  if (error || !data) {
    return <div className="text-rose-200">Unable to load overview.</div>;
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Card>
          <CardHeader>
            <CardTitle>Managed Targets</CardTitle>
          </CardHeader>
          <div className="text-4xl font-display text-ink-100">{data.managed_targets}</div>
          <CardDescription>{data.managed_services} services tracked</CardDescription>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Updates Available</CardTitle>
          </CardHeader>
          <div className="text-4xl font-display text-ember-500">{data.updates_available}</div>
          <CardDescription>Plan before apply</CardDescription>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Last Run</CardTitle>
          </CardHeader>
          <div className="text-lg text-ink-100">
            {data.last_run ? (
              <TimeAgo date={data.last_run.completed_at} />
            ) : (
              "No recent runs"
            )}
          </div>
          <CardDescription>
            {data.last_run ? (
              <Badge variant={data.last_run.status === "failed" ? "danger" : "success"}>
                {data.last_run.status}
              </Badge>
            ) : (
              ""
            )}
          </CardDescription>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Failures / Rollbacks</CardTitle>
          </CardHeader>
          <div className="text-4xl font-display text-rose-300">
            {data.failures} / {data.rollbacks}
          </div>
          <CardDescription>From recent history</CardDescription>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Activity Stream</CardTitle>
          <CardDescription>Last 20 events</CardDescription>
        </CardHeader>
        <div className="space-y-3">
          {data.activity.length === 0 && (
            <div className="text-sm text-ink-400">No recent activity yet.</div>
          )}
          {data.activity.map((event, index) => (
            <div key={`${event.ts}-${index}`} className="flex items-start justify-between gap-4">
              <div>
                <div className="text-sm text-ink-100">
                  <span className="font-semibold">{event.action}</span> · {event.message}
                </div>
                <div className="text-xs text-ink-400">
                  {event.target ?? ""} {event.service ? `/${event.service}` : ""}
                </div>
              </div>
              <TimeAgo date={event.ts} className="shrink-0 text-xs text-ink-400" />
            </div>
          ))}
        </div>
      </Card>
    </div>
  );
}
