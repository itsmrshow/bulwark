import { Fragment, useState } from "react";
import { ChevronDown, ChevronRight, History, Search } from "lucide-react";
import { useHistory } from "../lib/queries";
import type { HistoryItem } from "../lib/types";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";
import { Skeleton } from "../components/ui/skeleton";
import { EmptyState } from "../components/EmptyState";
import { TimeAgo } from "../components/TimeAgo";

function shortDigest(digest?: string) {
  if (!digest) return "—";
  const bare = digest.startsWith("sha256:") ? digest.slice(7) : digest;
  return bare.slice(0, 12) || "—";
}

function resultBadge(item: HistoryItem) {
  if (item.rolled_back) return <Badge variant="warning">Rolled back</Badge>;
  if (item.success)     return <Badge variant="success">Success</Badge>;
  return <Badge variant="danger">Failed</Badge>;
}

export function HistoryPage() {
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState({ target_id: "", service_id: "", result: "" });
  const { data, isLoading } = useHistory(page, 20, filters);
  const [expanded, setExpanded] = useState<string | null>(null);

  const updateFilter = (key: keyof typeof filters, value: string) => {
    setPage(1);
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  return (
    <div className="space-y-4">

      {/* ── Filters ───────────────────────────────────── */}
      <div className="rounded-2xl border border-ink-800/60 bg-ink-900/70 p-5">
        <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-ink-300">
          <Search className="h-4 w-4 text-ink-500" />
          Filter
        </div>
        <div className="grid gap-3 md:grid-cols-3">
          <div>
            <label className="mb-1.5 block text-xs text-ink-500">Target</label>
            <Input
              placeholder="e.g. myapp"
              value={filters.target_id}
              onChange={(e) => updateFilter("target_id", e.target.value)}
            />
          </div>
          <div>
            <label className="mb-1.5 block text-xs text-ink-500">Service</label>
            <Input
              placeholder="e.g. web"
              value={filters.service_id}
              onChange={(e) => updateFilter("service_id", e.target.value)}
            />
          </div>
          <div>
            <label className="mb-1.5 block text-xs text-ink-500">Result</label>
            <Input
              placeholder="success · failed · rolled_back"
              value={filters.result}
              onChange={(e) => updateFilter("result", e.target.value)}
            />
          </div>
        </div>
      </div>

      {/* ── Table ─────────────────────────────────────── */}
      <div className="rounded-2xl border border-ink-800/60 bg-ink-900/70">
        <div className="flex items-center justify-between border-b border-ink-800/50 px-5 py-4">
          <div>
            <h2 className="font-display text-base font-semibold text-ink-100">Update History</h2>
            <p className="text-xs text-ink-500">Audit update outcomes and rollback decisions</p>
          </div>
          <History className="h-4 w-4 text-ink-600" />
        </div>

        {isLoading && (
          <div className="divide-y divide-ink-800/40">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex items-center gap-4 px-5 py-4">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-6 w-20 rounded-full" />
                <Skeleton className="ml-auto h-4 w-16" />
              </div>
            ))}
          </div>
        )}

        {!isLoading && (data?.items ?? []).length === 0 && (
          <EmptyState
            icon={History}
            title="No update history"
            description="History appears after your first apply run."
          />
        )}

        {!isLoading && (data?.items ?? []).length > 0 && (
          <>
            {/* Column headers */}
            <div className="grid grid-cols-[1fr_auto_auto_auto_auto] gap-4 border-b border-ink-800/40 px-5 py-2 text-xs font-semibold uppercase tracking-wider text-ink-500">
              <span>Service</span>
              <span>Result</span>
              <span className="hidden sm:block">Duration</span>
              <span className="hidden sm:block">Completed</span>
              <span />
            </div>

            <div className="divide-y divide-ink-800/30">
              {(data?.items ?? []).map((item) => {
                const isExpanded = expanded === item.service_id;
                return (
                  <Fragment key={item.service_id}>
                    <button
                      className="grid w-full grid-cols-[1fr_auto_auto_auto_auto] items-center gap-4 px-5 py-3.5 text-left transition-colors hover:bg-ink-800/20"
                      onClick={() => setExpanded(isExpanded ? null : item.service_id)}
                    >
                      <div>
                        <div className="text-sm font-semibold text-ink-100">{item.service_name}</div>
                        <div className="font-mono text-xs text-ink-500">{item.target_id}</div>
                      </div>
                      <div>{resultBadge(item)}</div>
                      <div className="hidden text-sm text-ink-400 sm:block">
                        {item.duration_sec.toFixed(1)}s
                      </div>
                      <div className="hidden text-sm text-ink-400 sm:block">
                        {item.completed_at ? <TimeAgo date={item.completed_at} /> : "—"}
                      </div>
                      <div className="text-ink-500">
                        {isExpanded
                          ? <ChevronDown className="h-4 w-4" />
                          : <ChevronRight className="h-4 w-4" />
                        }
                      </div>
                    </button>

                    {isExpanded && (
                      <div className="border-t border-ink-800/30 bg-ink-950/40 px-5 py-4">
                        <div className="grid gap-x-8 gap-y-3 text-xs sm:grid-cols-2">
                          <div>
                            <div className="mb-1 text-ink-500 uppercase tracking-wide">Old digest</div>
                            <code className="rounded-md bg-ink-800 px-2 py-1 font-mono text-ink-400 ring-1 ring-ink-700/50">
                              {shortDigest(item.old_digest)}
                            </code>
                          </div>
                          <div>
                            <div className="mb-1 text-ink-500 uppercase tracking-wide">New digest</div>
                            <code className="rounded-md bg-signal-500/10 px-2 py-1 font-mono text-signal-400 ring-1 ring-signal-500/25">
                              {shortDigest(item.new_digest)}
                            </code>
                          </div>
                          <div>
                            <div className="mb-1 text-ink-500 uppercase tracking-wide">Probes</div>
                            <span className="text-ink-300">
                              {item.probes_passed} passed
                              {item.probes_failed > 0 && (
                                <span className="ml-1 text-rose-400">· {item.probes_failed} failed</span>
                              )}
                            </span>
                          </div>
                          <div>
                            <div className="mb-1 text-ink-500 uppercase tracking-wide">Started</div>
                            <span className="text-ink-300">
                              {item.started_at ? new Date(item.started_at).toLocaleString() : "—"}
                            </span>
                          </div>
                          {item.error_message && (
                            <div className="col-span-2">
                              <div className="mb-1 text-ink-500 uppercase tracking-wide">Error</div>
                              <p className="text-rose-300">{item.error_message}</p>
                            </div>
                          )}
                        </div>
                      </div>
                    )}
                  </Fragment>
                );
              })}
            </div>
          </>
        )}

        {/* Pagination */}
        <div className="flex items-center justify-between border-t border-ink-800/40 px-5 py-3">
          <Button variant="secondary" size="sm" disabled={page === 1} onClick={() => setPage(page - 1)}>
            Previous
          </Button>
          <span className="text-xs text-ink-500">Page {page}</span>
          <Button variant="secondary" size="sm" disabled={!data?.has_more} onClick={() => setPage(page + 1)}>
            Next
          </Button>
        </div>
      </div>
    </div>
  );
}
