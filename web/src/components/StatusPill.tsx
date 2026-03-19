import { cn } from "../lib/utils";

const statusStyles: Record<string, string> = {
  running:   "bg-signal-500/15 text-signal-400 border-signal-500/40",
  completed: "bg-emerald-400/15 text-emerald-300 border-emerald-400/30",
  failed:    "bg-rose-400/15 text-rose-300 border-rose-400/30"
};

export function StatusPill({ status }: { status?: string }) {
  const style = statusStyles[status ?? ""] ?? "bg-ink-800 text-ink-400 border-ink-700";
  const isRunning = status === "running";

  return (
    <span className={cn("inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-semibold", style)}>
      {isRunning ? (
        <span className="h-1.5 w-1.5 rounded-full bg-signal-400 animate-glow-pulse" />
      ) : (
        <span className={cn(
          "h-1.5 w-1.5 rounded-full",
          status === "completed" ? "bg-emerald-400" : status === "failed" ? "bg-rose-400" : "bg-ink-500"
        )} />
      )}
      {status ?? "unknown"}
    </span>
  );
}
