import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AlertTriangle, ClipboardCheck } from "lucide-react";
import { useApply, usePlan } from "../lib/queries";
import type { PlanItem } from "../lib/types";
import { Card, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "../components/ui/dialog";
import { Skeleton } from "../components/ui/skeleton";
import { EmptyState } from "../components/EmptyState";
import { RiskBadge } from "../components/RiskBadge";
import { useToast } from "../components/Toast";

function groupPlan(items: PlanItem[]) {
  return items.filter((item) => item.update_available);
}

function formatDigest(digest: string): string {
  const bare = digest.startsWith("sha256:") ? digest.slice(7) : digest;
  return bare.slice(0, 12) || "unknown";
}

export function PlanPage({ readOnly }: { readOnly: boolean }) {
  const { data: plan, isLoading, error } = usePlan();
  const applyMutation = useApply();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [confirmMode, setConfirmMode] = useState<"safe" | "selected" | null>(null);
  const [confirmText, setConfirmText] = useState("");

  const updates = useMemo(() => groupPlan(plan?.items ?? []), [plan?.items]);

  const toggleSelect = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (selected.size === updates.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(updates.map(item => item.service_id)));
    }
  };

  const apply = async (mode: "safe" | "selected") => {
    const payload: Record<string, unknown> = { mode };
    if (mode === "selected") {
      payload.service_ids = Array.from(selected);
    }
    try {
      const response = await applyMutation.mutateAsync(payload);
      toast("Apply run started", "info");
      navigate(`/apply?run=${response.run_id}`);
    } catch {
      toast("Failed to start apply run", "error");
    }
  };

  const canApplySelected = selected.size > 0;

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Card className="border-ember-500/40 bg-ember-500/10">
          <CardHeader>
            <Skeleton className="h-5 w-40" />
            <Skeleton className="h-3 w-72" />
          </CardHeader>
        </Card>
        <Card>
          <CardHeader>
            <Skeleton className="h-5 w-32" />
            <Skeleton className="h-3 w-56" />
          </CardHeader>
          <div className="divide-y divide-ink-800/60">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex items-center justify-between py-4">
                <div className="space-y-2">
                  <Skeleton className="h-4 w-48" />
                  <Skeleton className="h-3 w-64" />
                </div>
                <Skeleton className="h-6 w-20 rounded-full" />
              </div>
            ))}
          </div>
        </Card>
      </div>
    );
  }

  if (error || !plan) {
    return <div className="text-rose-200">Unable to build plan. Check Bulwark API logs.</div>;
  }

  return (
    <div className="space-y-6">
      <Card className="border-ember-500/40 bg-ember-500/10">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-ember-200">
            <ClipboardCheck className="h-5 w-5" />
            Plan before apply
          </CardTitle>
          <CardDescription className="text-amber-100">
            Review every update, risk level, and policy decision before starting an apply run.
          </CardDescription>
        </CardHeader>
      </Card>

      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex items-center gap-3">
          <Badge variant="default">{plan.update_count} updates available</Badge>
          <Badge variant="success">{plan.allowed_count} allowed</Badge>
          <Badge variant="muted">{plan.service_count} services tracked</Badge>
        </div>
        <div className="flex flex-wrap gap-3">
          <Button
            variant="primary"
            size="sm"
            disabled={readOnly || plan.allowed_count === 0}
            onClick={() => {
              setConfirmMode("safe");
              setConfirmText("");
            }}
          >
            Apply Safe Updates
          </Button>
          <Button
            variant="secondary"
            size="sm"
            disabled={readOnly || !canApplySelected}
            onClick={() => {
              setConfirmMode("selected");
              setConfirmText("");
            }}
          >
            Apply Selected
          </Button>
          {readOnly && <span className="text-xs text-amber-200">Read-only mode enabled</span>}
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Update Plan</CardTitle>
          <CardDescription>Only services with available updates are shown.</CardDescription>
        </CardHeader>
        {updates.length > 0 && (
          <div className="border-b border-ink-800/60 px-6 py-3">
            <label className="flex items-center gap-2 text-sm text-ink-300 cursor-pointer">
              <button
                className="h-4 w-4 rounded border border-ink-700"
                aria-label="select all"
                onClick={toggleSelectAll}
              >
                {selected.size === updates.length && updates.length > 0 && (
                  <div className="h-full w-full bg-signal-500" />
                )}
              </button>
              <span>Select All ({updates.length} updates)</span>
            </label>
          </div>
        )}
        <div className="divide-y divide-ink-800/40">
          {updates.length === 0 && (
            <EmptyState
              icon={ClipboardCheck}
              title="All up to date"
              description="No digest changes detected."
            />
          )}
          {updates.map((item) => (
            <div
              key={item.service_id}
              className={`flex flex-col gap-3 px-1 py-4 transition-colors hover:bg-ink-800/20 lg:flex-row lg:items-center lg:justify-between ${
                selected.has(item.service_id) ? "bg-signal-500/5" : ""
              }`}
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2.5">
                  <button
                    className={`h-4 w-4 rounded border transition-colors ${
                      selected.has(item.service_id)
                        ? "border-signal-500 bg-signal-500"
                        : "border-ink-600 hover:border-signal-500/60"
                    }`}
                    aria-label="select"
                    onClick={() => toggleSelect(item.service_id)}
                  />
                  <span className="text-sm font-semibold text-ink-100">
                    <span className="text-ink-400">{item.target_name}</span>
                    <span className="text-ink-600">/</span>
                    {item.service_name}
                  </span>
                  <RiskBadge risk={item.risk} />
                  {!item.allowed && <Badge variant="danger">Blocked</Badge>}
                </div>
                <div className="mt-1.5 flex items-center gap-3 pl-[26px] text-xs text-ink-500">
                  <span className="font-medium uppercase tracking-wide">{item.policy}</span>
                  <span className="text-ink-700">·</span>
                  <span>{item.tier}</span>
                  <span className="text-ink-700">·</span>
                  <span className="truncate font-mono">{item.image}</span>
                </div>
                {item.reason && (
                  <div className="mt-1 pl-[26px] text-xs text-ink-500">{item.reason}</div>
                )}
              </div>
              <div className="flex items-center gap-1.5 font-mono text-xs shrink-0">
                <span className="rounded-md bg-ink-800 px-2 py-1 text-ink-400 ring-1 ring-ink-700/60">
                  {item.current_digest ? formatDigest(item.current_digest) : "—"}
                </span>
                <span className="text-ink-600">→</span>
                <span className="rounded-md bg-signal-500/10 px-2 py-1 text-signal-400 ring-1 ring-signal-500/30">
                  {formatDigest(item.remote_digest)}
                </span>
              </div>
            </div>
          ))}
        </div>
      </Card>

      <Dialog open={confirmMode !== null} onOpenChange={() => setConfirmMode(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm apply run</DialogTitle>
            <DialogDescription>
              {confirmMode === "safe"
                ? "Apply all safe/stateless updates now?"
                : "Apply selected updates now?"}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 text-sm text-ink-200">
            <div className="rounded-lg border border-ink-700 bg-ink-800/40 p-3">
              <div className="flex items-center gap-2">
                <AlertTriangle className="h-4 w-4 text-amber-200" />
                This action will restart services and may trigger rollbacks.
              </div>
            </div>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={confirmText === "confirm"}
                onChange={(event) => setConfirmText(event.target.checked ? "confirm" : "")}
              />
              I understand the impact of applying updates.
            </label>
          </div>
          <div className="mt-6 flex justify-end gap-2">
            <Button variant="ghost" onClick={() => setConfirmMode(null)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              disabled={confirmText !== "confirm" || applyMutation.isPending}
              onClick={() => confirmMode && apply(confirmMode)}
            >
              {applyMutation.isPending ? "Starting..." : "Start apply"}
            </Button>
          </div>
          {applyMutation.error && (
            <div className="mt-4 flex items-center gap-2 text-sm text-rose-200">
              <AlertTriangle className="h-4 w-4" />
              {String(applyMutation.error)}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
