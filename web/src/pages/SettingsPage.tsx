import { useState, useEffect } from "react";
import { Card, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { useHealth } from "../lib/queries";
import { apiFetch } from "../lib/api";

export function SettingsPage() {
  const { data: health } = useHealth();
  const [writeMode, setWriteMode] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  // Verify session status on mount by checking if we can access a protected endpoint
  useEffect(() => {
    const checkSession = async () => {
      const stored = sessionStorage.getItem("bulwark_write_enabled");
      if (stored === "true") {
        // Verify the session is actually valid by testing a protected endpoint
        try {
          // Try to fetch the plan which requires auth
          await apiFetch("/api/plan", { method: "POST", body: JSON.stringify({}) });
          setWriteMode(true);
        } catch (err) {
          // Session expired or invalid, clear the flag
          sessionStorage.removeItem("bulwark_write_enabled");
          setWriteMode(false);
          setError("Session expired. Please enable write mode again.");
        }
      }
    };
    checkSession();
  }, []);

  const handleToggleWriteMode = async (enabled: boolean) => {
    setIsLoading(true);
    setError("");
    setSuccess("");

    try {
      if (enabled) {
        // Enable write mode by authenticating with backend token
        await apiFetch("/api/enable-writes", { method: "POST" });
        setWriteMode(true);
        sessionStorage.setItem("bulwark_write_enabled", "true");
        // Dispatch event so other components know write mode changed
        window.dispatchEvent(new CustomEvent("writemode-changed", { detail: { enabled: true } }));
        setSuccess("Write mode enabled - you can now apply updates");
      } else {
        // Disable by logging out
        await apiFetch("/api/logout", { method: "POST" });
        setWriteMode(false);
        sessionStorage.removeItem("bulwark_write_enabled");
        // Dispatch event so other components know write mode changed
        window.dispatchEvent(new CustomEvent("writemode-changed", { detail: { enabled: false } }));
        setSuccess("Write mode disabled - updates are now read-only");
      }
    } catch (err) {
      setError(String(err));
      setWriteMode(!enabled); // Revert toggle
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Settings</CardTitle>
          <CardDescription>
            Configure Bulwark behavior and permissions
          </CardDescription>
        </CardHeader>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Write Permissions</CardTitle>
          <CardDescription>
            Control whether this session can apply updates to containers
          </CardDescription>
        </CardHeader>
        <div className="px-6 pb-6 space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <div className="font-semibold text-ink-100">Enable Write Mode</div>
              <div className="text-sm text-ink-400 mt-1">
                When enabled, you can apply updates and perform write operations.
                {health?.read_only && " (Currently in read-only mode globally)"}
              </div>
            </div>
            <button
              onClick={() => handleToggleWriteMode(!writeMode)}
              disabled={isLoading || health?.read_only}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                writeMode ? "bg-emerald-500" : "bg-ink-700"
              } ${isLoading || health?.read_only ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}`}
            >
              <span
                className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                  writeMode ? "translate-x-6" : "translate-x-1"
                }`}
              />
            </button>
          </div>

          {writeMode && (
            <div className="rounded-lg border border-emerald-500/40 bg-emerald-500/10 p-3">
              <div className="text-sm text-emerald-200">
                ✓ Write mode is active - you can now apply updates from the Plan page
              </div>
            </div>
          )}

          {!writeMode && !health?.read_only && (
            <div className="rounded-lg border border-ink-700 bg-ink-800/40 p-3">
              <div className="text-sm text-ink-300">
                Write mode is disabled - toggle on to enable update operations
              </div>
            </div>
          )}

          {health?.read_only && (
            <div className="rounded-lg border border-amber-500/40 bg-amber-500/10 p-3">
              <div className="text-sm text-amber-200">
                Bulwark is in global read-only mode (BULWARK_UI_READONLY=true)
              </div>
            </div>
          )}

          {error && (
            <div className="rounded-lg border border-rose-500/40 bg-rose-500/10 p-3">
              <div className="text-sm text-rose-200">{error}</div>
            </div>
          )}

          {success && (
            <div className="rounded-lg border border-emerald-500/40 bg-emerald-500/10 p-3">
              <div className="text-sm text-emerald-200">{success}</div>
            </div>
          )}
        </div>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>System Information</CardTitle>
          <CardDescription>Current Bulwark configuration</CardDescription>
        </CardHeader>
        <div className="px-6 pb-6 space-y-3">
          <div className="flex justify-between text-sm">
            <span className="text-ink-400">Web UI Status</span>
            <span className="text-ink-100 font-mono">
              {health?.ui_enabled ? "Enabled" : "Disabled"}
            </span>
          </div>
          <div className="flex justify-between text-sm">
            <span className="text-ink-400">Global Mode</span>
            <span className="text-ink-100 font-mono">
              {health?.read_only ? "Read-Only" : "Write Enabled"}
            </span>
          </div>
          <div className="flex justify-between text-sm">
            <span className="text-ink-400">Session Status</span>
            <span className={`font-mono ${writeMode ? "text-emerald-400" : "text-ink-400"}`}>
              {writeMode ? "Write Enabled" : "Read-Only"}
            </span>
          </div>
        </div>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>About</CardTitle>
          <CardDescription>Bulwark Container Update Manager</CardDescription>
        </CardHeader>
        <div className="px-6 pb-6 space-y-2 text-sm text-ink-300">
          <p>
            Bulwark safely manages Docker container updates with policy-based automation,
            health probes, and automatic rollback.
          </p>
          <div className="mt-4 text-xs text-ink-500">
            Configure behavior using labels in your docker-compose.yml files
          </div>
        </div>
      </Card>
    </div>
  );
}
