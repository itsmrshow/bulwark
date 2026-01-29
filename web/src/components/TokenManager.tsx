import { useState, useEffect } from "react";
import { useHealth } from "../lib/queries";

export function TokenManager() {
  const [writeMode, setWriteMode] = useState(false);
  const { data: health } = useHealth();

  // Check write mode status on mount
  useEffect(() => {
    const stored = sessionStorage.getItem("bulwark_write_enabled");
    if (stored === "true") {
      setWriteMode(true);
    }

    // Listen for write mode changes from Settings page
    const handleWriteModeChange = (event: Event) => {
      const customEvent = event as CustomEvent<{ enabled: boolean }>;
      setWriteMode(customEvent.detail.enabled);
    };

    window.addEventListener("writemode-changed", handleWriteModeChange);
    return () => window.removeEventListener("writemode-changed", handleWriteModeChange);
  }, []);

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
      {writeMode ? (
        <span className="text-emerald-400">✓ Write Mode Enabled</span>
      ) : (
        <span className="text-ink-400">Read-only (Enable in Settings)</span>
      )}
    </div>
  );
}
