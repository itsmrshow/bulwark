import { useHealth } from "../lib/queries";

export function TokenManager() {
  const { data: health } = useHealth();

  // Show read-only status if in read-only mode
  if (health?.read_only) {
    return (
      <div className="text-sm text-amber-300">
        Global Read-only Mode
      </div>
    );
  }

  return (
    <div className="text-sm">
      <span className="text-emerald-400">✓ Write Enabled</span>
    </div>
  );
}
