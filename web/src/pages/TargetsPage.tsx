import { useMemo, useState } from "react";
import { Target, X } from "lucide-react";
import { usePlan, useTargets } from "../lib/queries";
import type { PlanItem, Target as TargetType } from "../lib/types";
import { Card, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Skeleton } from "../components/ui/skeleton";
import { EmptyState } from "../components/EmptyState";
import { RiskBadge } from "../components/RiskBadge";

function buildPlanLookup(items: PlanItem[]) {
  const map = new Map<string, PlanItem>();
  items.forEach((item) => map.set(item.service_id, item));
  return map;
}

export function TargetsPage() {
  const { data: targets, isLoading } = useTargets();
  const { data: plan } = usePlan();
  const [selected, setSelected] = useState<TargetType | null>(null);
  const [search, setSearch] = useState("");

  const planLookup = useMemo(() => buildPlanLookup(plan?.items ?? []), [plan?.items]);

  const filtered = useMemo(() => {
    if (!targets) return [];
    if (!search.trim()) return targets;
    const q = search.toLowerCase();
    return targets.filter((t) => t.name.toLowerCase().includes(q));
  }, [targets, search]);

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-3 w-64" />
        </CardHeader>
        <div className="divide-y divide-ink-800/60">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex items-center justify-between px-2 py-4">
              <div className="space-y-2">
                <Skeleton className="h-4 w-48" />
                <Skeleton className="h-3 w-64" />
              </div>
              <div className="flex gap-2">
                <Skeleton className="h-6 w-16 rounded-full" />
                <Skeleton className="h-6 w-24 rounded-full" />
              </div>
            </div>
          ))}
        </div>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <CardTitle>Managed Targets</CardTitle>
              <CardDescription>Compose projects and containers tracked by Bulwark</CardDescription>
            </div>
            <Input
              placeholder="Filter targets..."
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              className="max-w-xs"
            />
          </div>
        </CardHeader>
        {filtered.length === 0 && !search && (
          <EmptyState
            icon={Target}
            title="No targets discovered"
            description="Add bulwark.enabled=true label to your containers."
          />
        )}
        {filtered.length === 0 && search && (
          <div className="px-6 pb-6 text-sm text-ink-400">
            No targets matching &ldquo;{search}&rdquo;
          </div>
        )}
        <div className="divide-y divide-ink-800/60">
          {filtered.map((target) => (
            <button
              key={target.id}
              className="flex w-full items-center justify-between px-2 py-4 text-left hover:bg-ink-800/40"
              onClick={() => setSelected(target)}
            >
              <div>
                <div className="text-sm font-semibold text-ink-100">{target.name}</div>
                <div className="text-xs text-ink-400">{target.path}</div>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant="muted">{target.type}</Badge>
                <Badge variant="default">{target.services.length} services</Badge>
              </div>
            </button>
          ))}
        </div>
      </Card>

      {selected && (
        <div className="fixed inset-0 z-40 bg-ink-950/70 backdrop-blur-sm">
          <aside className="absolute right-0 top-0 h-full w-full max-w-2xl overflow-y-auto border-l border-ink-800 bg-ink-900/95 p-6">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-display text-xl">{selected.name}</h3>
                <p className="text-sm text-ink-400">{selected.path}</p>
              </div>
              <button
                onClick={() => setSelected(null)}
                className="rounded-full border border-ink-700 p-2 text-ink-300 hover:text-ink-100"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="mt-6 space-y-4">
              {selected.services.map((service) => {
                const planItem = planLookup.get(service.id);
                return (
                  <Card key={service.id} className="bg-ink-900/60">
                    <CardHeader>
                      <CardTitle className="text-base">{service.name}</CardTitle>
                      <div className="flex items-center gap-2">
                        <RiskBadge risk={planItem?.risk ?? "safe"} />
                        <Badge variant="muted">{service.labels.policy}</Badge>
                        <Badge variant="muted">{service.labels.tier}</Badge>
                      </div>
                    </CardHeader>
                    <div className="space-y-2 text-sm text-ink-200">
                      <div className="flex justify-between">
                        <span>Image</span>
                        <span className="text-ink-100">{service.image}</span>
                      </div>
                      <div className="flex justify-between">
                        <span>Current Digest</span>
                        <span className="text-ink-100">{service.current_digest || "-"}</span>
                      </div>
                      <div className="flex justify-between">
                        <span>Remote Digest</span>
                        <span className="text-ink-100">{planItem?.remote_digest || "-"}</span>
                      </div>
                      <div className="flex justify-between">
                        <span>Probe</span>
                        <span className="text-ink-100">{service.labels.probe.type}</span>
                      </div>
                    </div>
                  </Card>
                );
              })}
            </div>
          </aside>
        </div>
      )}
    </div>
  );
}
