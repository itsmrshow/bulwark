import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AlertTriangle, ClipboardCheck } from "lucide-react";
import { useApply, usePlan } from "../lib/queries";
import type { PlanItem } from "../lib/types";
import { Card, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "../components/ui/dialog";
import { RiskBadge } from "../components/RiskBadge";

function groupPlan(items: PlanItem[]) {
  return items.filter((item) => item.update_available);
}

export function PlanPage({ readOnly }: { readOnly: boolean }) {
  const { data: plan, isLoading, error } = usePlan();
  const applyMutation = useApply();
  const navigate = useNavigate();
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

  const apply = async (mode: "safe" | "selected") => {
    const payload: Record<string, unknown> = { mode };
    if (mode === "selected") {
      payload.service_ids = Array.from(selected);
    }
    const response = await applyMutation.mutateAsync(payload);
    navigate(`/apply?run=${response.run_id}`);
  };

  const canApplySelected = selected.size > 0;

  if (isLoading) {
    return <div className="text-ink-300">Building plan...</div>;
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
        <div className="divide-y divide-ink-800/60">
          {updates.length === 0 && (
            <div className="py-6 text-sm text-ink-400">No updates available right now.</div>
          )}
          {updates.map((item) => (
            <div key={item.service_id} className="flex flex-col gap-3 py-4 lg:flex-row lg:items-center lg:justify-between">
              <div>
                <div className="flex items-center gap-2">
                  <button
                    className="h-4 w-4 rounded border border-ink-700"
                    aria-label="select"
                    onClick={() => toggleSelect(item.service_id)}
                  >
                    {selected.has(item.service_id) && <div className="h-full w-full bg-signal-500" />}
                  </button>
                  <div className="text-sm font-semibold text-ink-100">
                    {item.target_name}/{item.service_name}
                  </div>
                  <RiskBadge risk={item.risk} />
                  {!item.allowed && <Badge variant="danger">Blocked</Badge>}
                </div>
                <div className="mt-2 text-xs text-ink-400">{item.reason}</div>
              </div>
              <div className="flex items-center gap-3 text-xs text-ink-300">
                <span>{item.policy}</span>
                <span>{item.tier}</span>
                <span className="truncate">{item.image}</span>
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
