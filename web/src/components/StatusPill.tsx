import { cn } from "../lib/utils";

const statusStyles: Record<string, string> = {
  running: "bg-signal-500/20 text-signal-500 border-signal-500/40",
  completed: "bg-emerald-400/20 text-emerald-200 border-emerald-400/40",
  failed: "bg-rose-400/20 text-rose-200 border-rose-400/40"
};

export function StatusPill({ status }: { status?: string }) {
  const style = statusStyles[status ?? ""] ?? "bg-ink-800 text-ink-200 border-ink-700";
  return (
    <span className={cn("inline-flex items-center rounded-full border px-2 py-1 text-xs font-semibold", style)}>
      {status ?? "unknown"}
    </span>
  );
}
