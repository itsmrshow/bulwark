import { useMemo, useState } from "react";
import { Boxes, ChevronRight, Container, Layers, X } from "lucide-react";
import { usePlan, useTargets } from "../lib/queries";
import type { PlanItem, Target as TargetType } from "../lib/types";
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

function shortDigest(digest?: string) {
  if (!digest) return null;
  const bare = digest.startsWith("sha256:") ? digest.slice(7) : digest;
  return bare.slice(0, 12);
}

function DigestPill({ digest, highlight = false }: { digest?: string; highlight?: boolean }) {
  const short = shortDigest(digest);
  if (!short) return <span className="text-ink-600">—</span>;
  return (
    <code className={`rounded-md px-2 py-0.5 font-mono text-xs ring-1 ${
      highlight
        ? "bg-signal-500/10 text-signal-400 ring-signal-500/30"
        : "bg-ink-800 text-ink-400 ring-ink-700/50"
    }`}>
      {short}
    </code>
  );
}

export function TargetsPage() {
  const { data: targets, isLoading } = useTargets();
  const [selected, setSelected] = useState<TargetType | null>(null);
  const [search, setSearch] = useState("");
  const { data: plan } = usePlan({ enabled: Boolean(selected), refetchInterval: false });

  const planLookup = useMemo(() => buildPlanLookup(plan?.items ?? []), [plan?.items]);

  const filtered = useMemo(() => {
    if (!targets) return [];
    if (!search.trim()) return targets;
    const q = search.toLowerCase();
    return targets.filter((t) => t.name.toLowerCase().includes(q));
  }, [targets, search]);

  if (isLoading) {
    return (
      <div className="rounded-2xl border border-ink-800/60 bg-ink-900/70 p-5">
        <div className="mb-4 flex items-center justify-between">
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-9 w-48 rounded-lg" />
        </div>
        <div className="divide-y divide-ink-800/40">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex items-center justify-between py-4">
              <div className="space-y-2">
                <Skeleton className="h-4 w-40" />
                <Skeleton className="h-3 w-56" />
              </div>
              <div className="flex gap-2">
                <Skeleton className="h-6 w-16 rounded-full" />
                <Skeleton className="h-6 w-20 rounded-full" />
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="rounded-2xl border border-ink-800/60 bg-ink-900/70">
        {/* Header */}
        <div className="flex flex-col gap-3 border-b border-ink-800/50 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 className="font-display text-base font-semibold text-ink-100">Managed Targets</h2>
            <p className="text-xs text-ink-500">Compose projects and containers tracked by Bulwark</p>
          </div>
          <Input
            placeholder="Filter targets…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="max-w-xs"
          />
        </div>

        {/* List */}
        {filtered.length === 0 && !search && (
          <EmptyState
            icon={Boxes}
            title="No targets discovered"
            description="Add bulwark.enabled=true label to your containers."
          />
        )}
        {filtered.length === 0 && search && (
          <div className="px-5 py-6 text-center text-sm text-ink-500">
            No targets matching &ldquo;{search}&rdquo;
          </div>
        )}
        <div className="divide-y divide-ink-800/40">
          {filtered.map((target) => {
            const Icon = target.type === "compose" ? Layers : Container;
            return (
              <button
                key={target.id}
                className="group flex w-full items-center gap-4 px-5 py-3.5 text-left transition-colors hover:bg-ink-800/30"
                onClick={() => setSelected(target)}
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-ink-800/80 text-ink-400 ring-1 ring-ink-700/50 group-hover:text-signal-400 transition-colors">
                  <Icon className="h-4 w-4" />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-semibold text-ink-100">{target.name}</div>
                  <div className="truncate font-mono text-xs text-ink-500">{target.path}</div>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <Badge variant="muted">{target.type}</Badge>
                  <Badge variant="default">{target.services.length} svc</Badge>
                  <ChevronRight className="h-3.5 w-3.5 text-ink-600 group-hover:text-ink-400 transition-colors" />
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* ── Detail panel ──────────────────────────────── */}
      {selected && (
        <div className="fixed inset-0 z-40 bg-ink-950/70 backdrop-blur-sm" onClick={() => setSelected(null)}>
          <aside
            className="absolute right-0 top-0 h-full w-full max-w-xl overflow-y-auto border-l border-ink-800/60 bg-ink-950 shadow-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Panel header */}
            <div className="sticky top-0 z-10 border-b border-ink-800/50 bg-ink-950/95 px-6 py-5 backdrop-blur-sm">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h3 className="font-display text-lg font-semibold text-ink-100">{selected.name}</h3>
                  <p className="mt-0.5 truncate font-mono text-xs text-ink-500">{selected.path}</p>
                </div>
                <button
                  onClick={() => setSelected(null)}
                  className="mt-0.5 rounded-lg border border-ink-700/60 p-1.5 text-ink-400 transition-colors hover:border-ink-600 hover:text-ink-200"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
              <div className="mt-3 flex items-center gap-2">
                <Badge variant="muted">{selected.type}</Badge>
                <Badge variant="default">{selected.services.length} service{selected.services.length !== 1 ? "s" : ""}</Badge>
              </div>
            </div>

            {/* Services */}
            <div className="space-y-3 p-6">
              {selected.services.map((service) => {
                const planItem = planLookup.get(service.id);
                const hasUpdate = planItem?.update_available;
                return (
                  <div
                    key={service.id}
                    className={`rounded-2xl border bg-ink-900/60 p-4 ${
                      hasUpdate ? "border-ember-500/30" : "border-ink-800/60"
                    }`}
                  >
                    {/* Service header */}
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="font-display text-sm font-semibold text-ink-100">{service.name}</span>
                          {hasUpdate && (
                            <span className="rounded-full bg-ember-500/15 px-2 py-0.5 text-xs font-medium text-ember-400 ring-1 ring-ember-500/25">
                              update available
                            </span>
                          )}
                        </div>
                        <p className="mt-0.5 font-mono text-xs text-ink-500">{service.image}</p>
                      </div>
                      <div className="flex shrink-0 items-center gap-1.5">
                        <RiskBadge risk={planItem?.risk ?? "safe"} />
                      </div>
                    </div>

                    {/* Labels row */}
                    <div className="mt-3 flex flex-wrap gap-1.5">
                      <Badge variant="muted">{service.labels.policy}</Badge>
                      <Badge variant="muted">{service.labels.tier}</Badge>
                      {service.labels.probe.type && (
                        <Badge variant="muted">probe: {service.labels.probe.type}</Badge>
                      )}
                    </div>

                    {/* Digest comparison */}
                    <div className="mt-3 flex items-center gap-2 text-xs text-ink-500">
                      <span className="w-14 shrink-0">Current</span>
                      <DigestPill digest={service.current_digest} />
                    </div>
                    {planItem?.remote_digest && (
                      <div className="mt-1.5 flex items-center gap-2 text-xs text-ink-500">
                        <span className="w-14 shrink-0">Remote</span>
                        <DigestPill digest={planItem.remote_digest} highlight={hasUpdate} />
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </aside>
        </div>
      )}
    </div>
  );
}
