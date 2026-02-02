import { useState, useEffect } from "react";
import { Card, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { useHealth, useSettings, useTestNotification, useUpdateSettings } from "../lib/queries";
import type { NotificationSettings } from "../lib/types";
import { Input } from "../components/ui/input";

export function SettingsPage() {
  const { data: health } = useHealth();
  const { data: settingsData } = useSettings();
  const locked = settingsData?.locked;
  const updateSettings = useUpdateSettings();
  const testNotification = useTestNotification();
  const [notificationError, setNotificationError] = useState("");
  const [notificationSuccess, setNotificationSuccess] = useState("");
  const [form, setForm] = useState<NotificationSettings>({
    discord_webhook: "",
    slack_webhook: "",
    discord_enabled: false,
    slack_enabled: false,
    notify_on_find: false,
    digest_enabled: false,
    check_cron: "*/15 * * * *",
    digest_cron: "0 9 * * *"
  });

  useEffect(() => {
    if (settingsData?.notifications) {
      const merged = { ...settingsData.notifications };
      if (settingsData.locked?.discord_webhook) {
        merged.discord_webhook = settingsData.notifications.discord_webhook;
        merged.discord_enabled = true;
      }
      if (settingsData.locked?.slack_webhook) {
        merged.slack_webhook = settingsData.notifications.slack_webhook;
        merged.slack_enabled = true;
      }
      setForm(merged);
    }
  }, [settingsData]);

  const handleSaveNotifications = async () => {
    setNotificationError("");
    setNotificationSuccess("");
    try {
      await updateSettings.mutateAsync({ notifications: form });
      setNotificationSuccess("Notification settings saved");
    } catch (err) {
      setNotificationError(String(err));
    }
  };

  const handleTestNotification = async () => {
    setNotificationError("");
    setNotificationSuccess("");
    try {
      await testNotification.mutateAsync();
      setNotificationSuccess("Test notification sent");
    } catch (err) {
      setNotificationError(String(err));
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
            <span className={`font-mono ${health?.read_only ? "text-ink-400" : "text-emerald-400"}`}>
              {health?.read_only ? "Read-Only" : "Write Enabled"}
            </span>
          </div>
        </div>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Notifications</CardTitle>
          <CardDescription>Send alerts to Discord or Slack when updates are found.</CardDescription>
        </CardHeader>
        <div className="px-6 pb-6 space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <label className="text-sm text-ink-400">Discord Webhook URL</label>
              <Input
                value={form.discord_webhook}
                onChange={(event) => setForm({ ...form, discord_webhook: event.target.value })}
                placeholder="https://discord.com/api/webhooks/..."
                type="password"
                disabled={Boolean(locked?.discord_webhook)}
              />
              {locked?.discord_webhook && (
                <div className="text-xs text-emerald-300 mt-1">Configured via environment</div>
              )}
              <div className="mt-2 flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={form.discord_enabled}
                  onChange={(event) => setForm({ ...form, discord_enabled: event.target.checked })}
                  disabled={Boolean(locked?.discord_webhook)}
                />
                <span className="text-sm text-ink-300">Enable Discord notifications</span>
              </div>
            </div>
            <div>
              <label className="text-sm text-ink-400">Slack Webhook URL</label>
              <Input
                value={form.slack_webhook}
                onChange={(event) => setForm({ ...form, slack_webhook: event.target.value })}
                placeholder="https://hooks.slack.com/services/..."
                type="password"
                disabled={Boolean(locked?.slack_webhook)}
              />
              {locked?.slack_webhook && (
                <div className="text-xs text-emerald-300 mt-1">Configured via environment</div>
              )}
              <div className="mt-2 flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={form.slack_enabled}
                  onChange={(event) => setForm({ ...form, slack_enabled: event.target.checked })}
                  disabled={Boolean(locked?.slack_webhook)}
                />
                <span className="text-sm text-ink-300">Enable Slack notifications</span>
              </div>
            </div>
          </div>

          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={form.notify_on_find}
                onChange={(event) => setForm({ ...form, notify_on_find: event.target.checked })}
              />
              <span className="text-sm text-ink-300">Notify immediately when updates are found</span>
            </div>
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={form.digest_enabled}
                onChange={(event) => setForm({ ...form, digest_enabled: event.target.checked })}
              />
              <span className="text-sm text-ink-300">Send digest summary</span>
            </div>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <label className="text-sm text-ink-400">Check schedule (cron)</label>
              <Input
                value={form.check_cron}
                onChange={(event) => setForm({ ...form, check_cron: event.target.value })}
                placeholder="*/15 * * * *"
              />
              <div className="mt-2 flex flex-wrap gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setForm({ ...form, check_cron: "0 */12 * * *" })}
                >
                  Every 12 hours
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setForm({ ...form, check_cron: "0 9 * * *" })}
                >
                  Once a day
                </Button>
              </div>
              <div className="text-xs text-ink-500 mt-1">Example: */15 * * * *</div>
            </div>
            <div>
              <label className="text-sm text-ink-400">Digest schedule (cron)</label>
              <Input
                value={form.digest_cron}
                onChange={(event) => setForm({ ...form, digest_cron: event.target.value })}
                placeholder="0 9 * * *"
              />
              <div className="mt-2 flex flex-wrap gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setForm({ ...form, digest_cron: "0 */12 * * *" })}
                >
                  Every 12 hours
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setForm({ ...form, digest_cron: "0 9 * * *" })}
                >
                  Once a day
                </Button>
              </div>
              <div className="text-xs text-ink-500 mt-1">Example: 0 9 * * *</div>
            </div>
          </div>

          <div className="flex flex-wrap gap-3">
            <Button
              variant="primary"
              size="sm"
              onClick={handleSaveNotifications}
              disabled={updateSettings.isPending || health?.read_only}
            >
              {updateSettings.isPending ? "Saving..." : "Save Settings"}
            </Button>
            <Button
              variant="secondary"
              size="sm"
              onClick={handleTestNotification}
              disabled={testNotification.isPending || health?.read_only}
            >
              {testNotification.isPending ? "Sending..." : "Send Test"}
            </Button>
            {health?.read_only && (
              <span className="text-xs text-amber-200">Read-only mode enabled</span>
            )}
          </div>

          {notificationError && (
            <div className="rounded-lg border border-rose-500/40 bg-rose-500/10 p-3">
              <div className="text-sm text-rose-200">{notificationError}</div>
            </div>
          )}

          {notificationSuccess && (
            <div className="rounded-lg border border-emerald-500/40 bg-emerald-500/10 p-3">
              <div className="text-sm text-emerald-200">{notificationSuccess}</div>
            </div>
          )}
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
