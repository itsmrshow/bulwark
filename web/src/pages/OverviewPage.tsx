import { useOverview } from "../lib/queries";
import { Card, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Badge } from "../components/ui/badge";

function formatDate(value?: string) {
  if (!value) return "Never";
  const date = new Date(value);
  return date.toLocaleString();
}

export function OverviewPage() {
  const { data, isLoading, error } = useOverview();

  if (isLoading) {
    return <div className="text-ink-300">Loading overview...</div>;
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
            {data.last_run ? formatDate(data.last_run.completed_at) : "No recent runs"}
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
              <div className="text-xs text-ink-400">{formatDate(event.ts)}</div>
            </div>
          ))}
        </div>
      </Card>
    </div>
  );
}
