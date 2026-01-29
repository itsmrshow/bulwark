import { ShieldAlert } from "lucide-react";

export function ReadOnlyBanner({ readOnly }: { readOnly: boolean }) {
  if (!readOnly) return null;
  return (
    <div className="mb-4 flex items-center gap-3 rounded-xl border border-amber-400/30 bg-amber-400/10 px-4 py-3 text-sm text-amber-200">
      <ShieldAlert className="h-4 w-4" />
      Read-only mode is enabled. Apply actions are disabled until writes are explicitly enabled.
    </div>
  );
}
