import { useHealth } from "../lib/queries";

export function TokenManager() {
  const { data: health } = useHealth();

  if (health?.read_only) {
    return (
      <div className="flex items-center gap-1.5 rounded-lg border border-amber-400/30 bg-amber-400/10 px-2.5 py-1 text-xs font-medium text-amber-300">
        <span className="h-1.5 w-1.5 rounded-full bg-amber-400" />
        Read-only
      </div>
    );
  }

  // Write mode is the happy path — no badge needed, sidebar dot covers it.
  return null;
}
