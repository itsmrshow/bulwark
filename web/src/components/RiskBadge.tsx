import { Badge } from "./ui/badge";
import { Tooltip, TooltipTrigger, TooltipContent } from "./ui/tooltip";
import { RiskLevel } from "../lib/types";

const labels: Record<RiskLevel, { label: string; variant: "success" | "warning" | "danger" | "muted"; description: string }> = {
  safe: { label: "Safe", variant: "success", description: "Stateless service with probes. Safe for automatic updates." },
  notify: { label: "Notify", variant: "muted", description: "Notify-only policy. Updates detected but not auto-applied." },
  stateful: { label: "Stateful", variant: "danger", description: "Stateful service. Blocked unless explicitly overridden." },
  probe_missing: { label: "Probe Missing", variant: "warning", description: "No health probe configured. Update safety unverified." }
};

export function RiskBadge({ risk }: { risk: RiskLevel }) {
  const info = labels[risk] ?? labels.safe;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Badge variant={info.variant}>{info.label}</Badge>
      </TooltipTrigger>
      <TooltipContent className="max-w-xs">{info.description}</TooltipContent>
    </Tooltip>
  );
}
