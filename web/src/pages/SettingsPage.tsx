import { useEffect, useMemo, useState } from "react";
import {
  AlertTriangle,
  BellRing,
  CalendarClock,
  Lock,
  Radio,
  RefreshCw,
  Send,
  ShieldCheck,
  Sparkles,
  Webhook
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { useHealth, useSettings, useTestNotification, useUpdateSettings } from "../lib/queries";
import type { NotificationSettings } from "../lib/types";
import { Input } from "../components/ui/input";
import { useToast } from "../components/Toast";

function enabledChannels(form: NotificationSettings) {
  const channels = [];
  if (form.discord_enabled && form.discord_webhook) channels.push("Discord");
  if (form.slack_enabled && form.slack_webhook) channels.push("Slack");
  return channels;
}

function previewRows(form: NotificationSettings) {
  return [
    {
      title: "Discovery alert",
      caption: form.notify_on_find ? "Sent as soon as Bulwark finds new digests." : "Disabled",
      body: "3 updates detected across 2 targets. 2 allowed, 1 blocked by policy."
    },
    {
      title: "Apply result",
      caption: "Sent after each update attempt.",
      body: "web updated in 8.3s. Probes passed and no rollback was needed."
    },
    {
      title: "Digest summary",
      caption: form.digest_enabled ? `Scheduled with \`${form.digest_cron}\`` : "Disabled",
      body: "Scheduled rollup of outstanding updates, target counts, and blocked items."
    }
  ];
}

function schedulePresetLabel(cron: string) {
  switch (cron) {
    case "*/15 * * * *":
      return "Every 15 minutes";
    case "0 */12 * * *":
      return "Every 12 hours";
    case "0 9 * * *":
      return "Daily at 09:00";
    default:
      return "Custom";
  }
}

export function SettingsPage() {
  const { data: health } = useHealth();
  const { data: settingsData } = useSettings();
  const locked = settingsData?.locked;
  const updateSettings = useUpdateSettings();
  const testNotification = useTestNotification();
  const { toast } = useToast();
  const [unsafeWarningPending, setUnsafeWarningPending] = useState(false);
  const [form, setForm] = useState<NotificationSettings>({
    discord_webhook: "",
    slack_webhook: "",
    discord_enabled: false,
    slack_enabled: false,
    notify_on_find: false,
    digest_enabled: false,
    check_cron: "*/15 * * * *",
    digest_cron: "0 9 * * *",
    auto_update_enabled: false,
    auto_update_safe: false,
    auto_update_unsafe: false,
    auto_update_cron: "0 3 * * *"
  });

  useEffect(() => {
    if (settingsData?.notifications) {
      const merged = { ...settingsData.notifications };
      if (!merged.auto_update_cron) merged.auto_update_cron = "0 3 * * *";
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

  const channels = useMemo(() => enabledChannels(form), [form]);
  const previews = useMemo(() => previewRows(form), [form]);

  const handleSaveNotifications = async () => {
    try {
      await updateSettings.mutateAsync({ notifications: form });
      toast("Settings saved", "success");
    } catch (err) {
      toast(String(err), "error");
    }
  };

  const handleTestNotification = async () => {
    try {
      await testNotification.mutateAsync();
      toast("Test notification sent", "success");
    } catch (err) {
      toast(String(err), "error");
    }
  };

  return (
    <div className="space-y-6">
      <Card className="relative overflow-hidden border-signal-500/30 bg-gradient-to-br from-signal-500/10 via-ink-900/90 to-ink-900/70">
        <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-signal-400/60 to-transparent" />
        <CardHeader className="items-start gap-4 md:flex-row md:justify-between">
          <div className="space-y-3">
            <Badge variant="default" className="bg-signal-500/15 text-signal-400">
              Notification Control Room
            </Badge>
            <div>
              <CardTitle className="text-2xl">Settings</CardTitle>
              <CardDescription className="max-w-2xl text-ink-300">
                Tune delivery channels, schedules, and alert behavior. Discord and Slack now use richer layouts, so this page focuses on clarity and previewing what operators will actually receive.
              </CardDescription>
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            <Badge variant={health?.read_only ? "warning" : "success"}>
              {health?.read_only ? "Read-only" : "Write enabled"}
            </Badge>
            <Badge variant={channels.length > 0 ? "success" : "muted"}>
              {channels.length > 0 ? `${channels.length} channel${channels.length > 1 ? "s" : ""} live` : "No channels"}
            </Badge>
          </div>
        </CardHeader>
      </Card>

      <div className="grid gap-6 xl:grid-cols-[1.15fr,0.85fr]">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ShieldCheck className="h-5 w-5 text-signal-400" />
              System Posture
            </CardTitle>
            <CardDescription>Current UI state and how Bulwark can act right now.</CardDescription>
          </CardHeader>
          <div className="grid gap-3 px-6 pb-6 md:grid-cols-3">
            <div className="rounded-2xl border border-ink-800/70 bg-ink-950/50 p-4">
              <div className="text-xs uppercase tracking-[0.2em] text-ink-500">Web Console</div>
              <div className="mt-3 text-lg font-semibold text-ink-100">
                {health?.ui_enabled ? "Online" : "Disabled"}
              </div>
              <div className="mt-1 text-sm text-ink-400">Frontend + API availability</div>
            </div>
            <div className="rounded-2xl border border-ink-800/70 bg-ink-950/50 p-4">
              <div className="text-xs uppercase tracking-[0.2em] text-ink-500">Permissions</div>
              <div className="mt-3 text-lg font-semibold text-ink-100">
                {health?.read_only ? "Read-only" : "Write capable"}
              </div>
              <div className="mt-1 text-sm text-ink-400">Affects save, test, apply, and rollback</div>
            </div>
            <div className="rounded-2xl border border-ink-800/70 bg-ink-950/50 p-4">
              <div className="text-xs uppercase tracking-[0.2em] text-ink-500">Active Channels</div>
              <div className="mt-3 text-lg font-semibold text-ink-100">
                {channels.length > 0 ? channels.join(" + ") : "None"}
              </div>
              <div className="mt-1 text-sm text-ink-400">Based on enabled toggles and configured hooks</div>
            </div>
          </div>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Sparkles className="h-5 w-5 text-ember-400" />
              Delivery Preview
            </CardTitle>
            <CardDescription>What your team will see when Bulwark speaks up.</CardDescription>
          </CardHeader>
          <div className="space-y-4 px-6 pb-6">
            <div className="flex flex-wrap gap-2">
              {channels.length === 0 && <Badge variant="muted">No destinations enabled</Badge>}
              {channels.map((channel) => (
                <Badge key={channel} variant="success">
                  {channel}
                </Badge>
              ))}
              <Badge variant={form.notify_on_find ? "warning" : "muted"}>
                {form.notify_on_find ? "Immediate alerts on" : "Immediate alerts off"}
              </Badge>
              <Badge variant={form.digest_enabled ? "warning" : "muted"}>
                {form.digest_enabled ? "Digest on" : "Digest off"}
              </Badge>
            </div>
            {previews.map((preview) => (
              <div key={preview.title} className="rounded-2xl border border-ink-800/70 bg-ink-950/50 p-4">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <div className="text-sm font-semibold text-ink-100">{preview.title}</div>
                    <div className="mt-1 text-xs text-ink-400">{preview.caption}</div>
                  </div>
                  <Badge variant="muted">Sample</Badge>
                </div>
                <div className="mt-3 text-sm leading-6 text-ink-200">{preview.body}</div>
              </div>
            ))}
          </div>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BellRing className="h-5 w-5 text-signal-400" />
            Notifications
          </CardTitle>
          <CardDescription>Configure webhooks, delivery behavior, and operator-friendly schedules.</CardDescription>
        </CardHeader>
        <div className="space-y-6 px-6 pb-6">
          <div className="grid gap-4 lg:grid-cols-2">
            <div className="rounded-2xl border border-ink-800/70 bg-ink-950/50 p-4">
              <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-ink-100">
                <Webhook className="h-4 w-4 text-signal-400" />
                Discord
              </div>
              <label className="text-sm text-ink-400">Webhook URL</label>
              <Input
                value={form.discord_webhook}
                onChange={(event) => setForm({ ...form, discord_webhook: event.target.value })}
                placeholder="https://discord.com/api/webhooks/..."
                type="password"
                disabled={Boolean(locked?.discord_webhook)}
                className="mt-2"
              />
              <div className="mt-3 flex items-center justify-between rounded-xl border border-ink-800/70 bg-ink-900/50 px-3 py-2">
                <div className="text-sm text-ink-300">Enable Discord delivery</div>
                <input
                  type="checkbox"
                  checked={form.discord_enabled}
                  onChange={(event) => setForm({ ...form, discord_enabled: event.target.checked })}
                  disabled={Boolean(locked?.discord_webhook)}
                />
              </div>
              {locked?.discord_webhook && (
                <div className="mt-2 flex items-center gap-2 text-xs text-emerald-300">
                  <Lock className="h-3.5 w-3.5" />
                  Managed by environment variable
                </div>
              )}
            </div>

            <div className="rounded-2xl border border-ink-800/70 bg-ink-950/50 p-4">
              <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-ink-100">
                <Webhook className="h-4 w-4 text-ember-400" />
                Slack
              </div>
              <label className="text-sm text-ink-400">Webhook URL</label>
              <Input
                value={form.slack_webhook}
                onChange={(event) => setForm({ ...form, slack_webhook: event.target.value })}
                placeholder="https://hooks.slack.com/services/..."
                type="password"
                disabled={Boolean(locked?.slack_webhook)}
                className="mt-2"
              />
              <div className="mt-3 flex items-center justify-between rounded-xl border border-ink-800/70 bg-ink-900/50 px-3 py-2">
                <div className="text-sm text-ink-300">Enable Slack delivery</div>
                <input
                  type="checkbox"
                  checked={form.slack_enabled}
                  onChange={(event) => setForm({ ...form, slack_enabled: event.target.checked })}
                  disabled={Boolean(locked?.slack_webhook)}
                />
              </div>
              {locked?.slack_webhook && (
                <div className="mt-2 flex items-center gap-2 text-xs text-emerald-300">
                  <Lock className="h-3.5 w-3.5" />
                  Managed by environment variable
                </div>
              )}
            </div>
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <div className="rounded-2xl border border-ink-800/70 bg-ink-950/50 p-4">
              <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-ink-100">
                <Radio className="h-4 w-4 text-signal-400" />
                Immediate Alerts
              </div>
              <div className="flex items-center justify-between rounded-xl border border-ink-800/70 bg-ink-900/50 px-3 py-2">
                <div>
                  <div className="text-sm text-ink-100">Notify on discovery</div>
                  <div className="text-xs text-ink-400">Best for active ops channels and fast triage</div>
                </div>
                <input
                  type="checkbox"
                  checked={form.notify_on_find}
                  onChange={(event) => setForm({ ...form, notify_on_find: event.target.checked })}
                />
              </div>
              <div className="mt-3">
                <label className="text-sm text-ink-400">Check schedule</label>
                <Input
                  value={form.check_cron}
                  onChange={(event) => setForm({ ...form, check_cron: event.target.value })}
                  placeholder="*/15 * * * *"
                  className="mt-2"
                />
                <div className="mt-2 flex flex-wrap gap-2">
                  <Button variant="ghost" size="sm" onClick={() => setForm({ ...form, check_cron: "*/15 * * * *" })}>
                    Every 15 min
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => setForm({ ...form, check_cron: "0 */12 * * *" })}>
                    Every 12h
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => setForm({ ...form, check_cron: "0 9 * * *" })}>
                    Daily
                  </Button>
                </div>
                <div className="mt-2 text-xs text-ink-500">{schedulePresetLabel(form.check_cron)}</div>
              </div>
            </div>

            <div className="rounded-2xl border border-ink-800/70 bg-ink-950/50 p-4">
              <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-ink-100">
                <CalendarClock className="h-4 w-4 text-ember-400" />
                Digest Summaries
              </div>
              <div className="flex items-center justify-between rounded-xl border border-ink-800/70 bg-ink-900/50 px-3 py-2">
                <div>
                  <div className="text-sm text-ink-100">Send rollup digest</div>
                  <div className="text-xs text-ink-400">Best for quieter teams that prefer scheduled review</div>
                </div>
                <input
                  type="checkbox"
                  checked={form.digest_enabled}
                  onChange={(event) => setForm({ ...form, digest_enabled: event.target.checked })}
                />
              </div>
              <div className="mt-3">
                <label className="text-sm text-ink-400">Digest schedule</label>
                <Input
                  value={form.digest_cron}
                  onChange={(event) => setForm({ ...form, digest_cron: event.target.value })}
                  placeholder="0 9 * * *"
                  className="mt-2"
                />
                <div className="mt-2 flex flex-wrap gap-2">
                  <Button variant="ghost" size="sm" onClick={() => setForm({ ...form, digest_cron: "0 9 * * *" })}>
                    Daily
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => setForm({ ...form, digest_cron: "0 */12 * * *" })}>
                    Every 12h
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => setForm({ ...form, digest_cron: "0 17 * * 1-5" })}>
                    Weekdays 17:00
                  </Button>
                </div>
                <div className="mt-2 text-xs text-ink-500">{schedulePresetLabel(form.digest_cron)}</div>
              </div>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3">
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
              <Send className="mr-2 h-4 w-4" />
              {testNotification.isPending ? "Sending..." : "Send Test Notification"}
            </Button>
            {health?.read_only && (
              <span className="text-xs text-amber-200">Read-only mode enabled</span>
            )}
          </div>
        </div>
      </Card>

      {/* Unsafe-update confirmation modal */}
      {unsafeWarningPending && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="mx-4 w-full max-w-md rounded-2xl border border-amber-500/40 bg-ink-900 p-6 shadow-2xl">
            <div className="flex items-start gap-3">
              <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-amber-400" />
              <div>
                <div className="text-base font-semibold text-amber-300">Enable unsafe auto-updates?</div>
                <p className="mt-2 text-sm leading-6 text-ink-300">
                  This will automatically update containers that are <strong className="text-ink-100">stateful</strong>, have <strong className="text-ink-100">policy=notify</strong>, or are <strong className="text-ink-100">missing health probes</strong>. These containers may hold persistent data and have no automated rollback protection. Data loss or service disruption is possible.
                </p>
                <p className="mt-3 text-sm text-ink-400">
                  Only enable this if you understand the risk and have external backups or other safeguards in place.
                </p>
              </div>
            </div>
            <div className="mt-5 flex justify-end gap-3">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setUnsafeWarningPending(false)}
              >
                Cancel
              </Button>
              <Button
                variant="primary"
                size="sm"
                onClick={() => {
                  setForm((f) => ({ ...f, auto_update_unsafe: true }));
                  setUnsafeWarningPending(false);
                }}
                className="border-amber-500/40 bg-amber-500/10 text-amber-300 hover:bg-amber-500/20"
              >
                I understand, enable it
              </Button>
            </div>
          </div>
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <RefreshCw className="h-5 w-5 text-emerald-400" />
            Auto Update
          </CardTitle>
          <CardDescription>
            Automatically apply image updates on a schedule — similar to Watchtower. Safe containers use health probes and automatic rollback. Unsafe containers skip those guardrails.
          </CardDescription>
        </CardHeader>
        <div className="space-y-4 px-6 pb-6">
          {/* Master toggle */}
          <div className="flex items-center justify-between rounded-xl border border-ink-800/70 bg-ink-950/50 px-3 py-2">
            <div>
              <div className="text-sm text-ink-100">Enable automatic updates</div>
              <div className="text-xs text-ink-400">Run updates on the schedule below</div>
            </div>
            <input
              type="checkbox"
              checked={form.auto_update_enabled}
              onChange={(e) => setForm({ ...form, auto_update_enabled: e.target.checked })}
              disabled={health?.read_only}
            />
          </div>

          <div className={`space-y-4 transition-opacity ${form.auto_update_enabled ? "opacity-100" : "pointer-events-none opacity-40"}`}>
            {/* Schedule */}
            <div>
              <label className="text-sm text-ink-400">Update schedule</label>
              <div className="mt-2 flex flex-wrap gap-2">
                <Button variant="ghost" size="sm" onClick={() => setForm({ ...form, auto_update_cron: "0 3 * * *" })}>
                  Daily 03:00
                </Button>
                <Button variant="ghost" size="sm" onClick={() => setForm({ ...form, auto_update_cron: "0 */6 * * *" })}>
                  Every 6h
                </Button>
                <Button variant="ghost" size="sm" onClick={() => setForm({ ...form, auto_update_cron: "0 3 * * 0" })}>
                  Weekly Sun 03:00
                </Button>
              </div>
              <input
                type="text"
                value={form.auto_update_cron}
                onChange={(e) => setForm({ ...form, auto_update_cron: e.target.value })}
                placeholder="0 3 * * *"
                disabled={health?.read_only}
                className="mt-2 w-full rounded-lg border border-ink-700 bg-ink-900 px-3 py-1.5 text-sm text-ink-100 placeholder-ink-600 focus:border-signal-500 focus:outline-none"
              />
            </div>

            {/* Safe tier */}
            <div className="rounded-2xl border border-ink-800/70 bg-ink-950/50 p-4">
              <div className="flex items-center justify-between">
                <div>
                  <div className="flex items-center gap-2 text-sm font-semibold text-ink-100">
                    <span className="inline-block h-2 w-2 rounded-full bg-emerald-400" />
                    Safe containers
                  </div>
                  <div className="mt-1 text-xs text-ink-400">
                    Stateless services with health probes configured. Supports automatic rollback if probes fail.
                  </div>
                </div>
                <input
                  type="checkbox"
                  checked={form.auto_update_safe}
                  onChange={(e) => setForm({ ...form, auto_update_safe: e.target.checked })}
                  disabled={health?.read_only}
                />
              </div>
            </div>

            {/* Unsafe tier */}
            <div className="rounded-2xl border border-amber-500/20 bg-amber-500/5 p-4">
              <div className="flex items-center justify-between">
                <div>
                  <div className="flex items-center gap-2 text-sm font-semibold text-amber-300">
                    <AlertTriangle className="h-3.5 w-3.5" />
                    Unsafe containers
                  </div>
                  <div className="mt-1 text-xs text-ink-400">
                    Stateful, policy=notify, or probe-missing services. No rollback protection. Enable with caution.
                  </div>
                </div>
                <input
                  type="checkbox"
                  checked={form.auto_update_unsafe}
                  onChange={(e) => {
                    if (e.target.checked) {
                      setUnsafeWarningPending(true);
                    } else {
                      setForm({ ...form, auto_update_unsafe: false });
                    }
                  }}
                  disabled={health?.read_only}
                />
              </div>
              {form.auto_update_unsafe && (
                <div className="mt-3 flex items-center gap-2 text-xs text-amber-400">
                  <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
                  Updates will be forced — policy blocks and probe requirements are bypassed.
                </div>
              )}
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3 pt-1">
            <Button
              variant="primary"
              size="sm"
              onClick={handleSaveNotifications}
              disabled={updateSettings.isPending || health?.read_only}
            >
              {updateSettings.isPending ? "Saving..." : "Save Settings"}
            </Button>
            {health?.read_only && (
              <span className="text-xs text-amber-200">Read-only mode enabled</span>
            )}
          </div>
        </div>
      </Card>
    </div>
  );
}
