import { Badge } from "./ui/badge";
import { RiskLevel } from "../lib/types";

const labels: Record<RiskLevel, { label: string; variant: "success" | "warning" | "danger" | "muted" }> = {
  safe: { label: "Safe", variant: "success" },
  notify: { label: "Notify", variant: "muted" },
  stateful: { label: "Stateful", variant: "danger" },
  probe_missing: { label: "Probe Missing", variant: "warning" }
};

export function RiskBadge({ risk }: { risk: RiskLevel }) {
  const badge = labels[risk] ?? labels.safe;
  return <Badge variant={badge.variant}>{badge.label}</Badge>;
}
