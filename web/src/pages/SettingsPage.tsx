import { useEffect, useState } from "react";
import {
  AlertTriangle,
  BellOff,
  BellRing,
  CalendarClock,
  Lock,
  Radio,
  RefreshCw,
  Send,
  Webhook
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { useHealth, useSettings, useTestNotification, useUpdateSettings } from "../lib/queries";
import type { NotificationSettings } from "../lib/types";
import { useToast } from "../components/Toast";

// ── Toggle switch ─────────────────────────────────────────
function Toggle({
  checked,
  onChange,
  disabled = false,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <button
      role="switch"
      aria-checked={checked}
      onClick={() => !disabled && onChange(!checked)}
      className={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-signal-500 focus-visible:ring-offset-2 focus-visible:ring-offset-ink-900 ${
        checked ? "bg-signal-500" : "bg-ink-700"
      } ${disabled ? "cursor-not-allowed opacity-50" : "cursor-pointer"}`}
    >
      <span
        className={`inline-block h-3.5 w-3.5 rounded-full bg-white shadow transition-transform ${
          checked ? "translate-x-[18px]" : "translate-x-[2px]"
        }`}
      />
    </button>
  );
}

// ── Setting row (label + description + toggle) ────────────
function SettingRow({
  label,
  description,
  checked,
  onChange,
  disabled = false,
}: {
  label: string;
  description?: string;
  checked: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-4 py-3">
      <div>
        <div className="text-sm font-medium text-ink-100">{label}</div>
        {description && <div className="mt-0.5 text-xs text-ink-500">{description}</div>}
      </div>
      <Toggle checked={checked} onChange={onChange} disabled={disabled} />
    </div>
  );
}

// ── Section wrapper ───────────────────────────────────────
function Section({
  icon: Icon,
  title,
  description,
  iconColor = "text-signal-400",
  children,
}: {
  icon: React.ElementType;
  title: string;
  description?: string;
  iconColor?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-2xl border border-ink-800/60 bg-ink-900/70">
      <div className="border-b border-ink-800/50 px-5 py-4">
        <div className="flex items-center gap-2">
          <Icon className={`h-4 w-4 ${iconColor}`} />
          <h2 className="font-display text-base font-semibold text-ink-100">{title}</h2>
        </div>
        {description && <p className="mt-0.5 text-xs text-ink-500">{description}</p>}
      </div>
      <div className="px-5 pb-5 pt-3">{children}</div>
    </div>
  );
}

// ── Cron preset picker ────────────────────────────────────
function CronPicker({
  label,
  value,
  onChange,
  presets,
  disabled = false,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  presets: { label: string; value: string }[];
  disabled?: boolean;
}) {
  return (
    <div className="mt-3">
      <label className="mb-1.5 block text-xs text-ink-500">{label}</label>
      <div className="flex flex-wrap gap-1.5">
        {presets.map((p) => (
          <button
            key={p.value}
            onClick={() => !disabled && onChange(p.value)}
            className={`rounded-lg px-2.5 py-1 text-xs font-medium transition-colors ${
              value === p.value
                ? "bg-signal-500/15 text-signal-400 ring-1 ring-signal-500/30"
                : "bg-ink-800/60 text-ink-400 hover:bg-ink-800 hover:text-ink-200"
            } ${disabled ? "cursor-not-allowed opacity-50" : ""}`}
          >
            {p.label}
          </button>
        ))}
      </div>
      <Input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="cron expression"
        disabled={disabled}
        className="mt-2 font-mono text-xs"
      />
    </div>
  );
}

// ─────────────────────────────────────────────────────────
export function SettingsPage() {
  const { data: health } = useHealth();
  const { data: settingsData } = useSettings();
  const locked = settingsData?.locked;
  const updateSettings = useUpdateSettings();
  const testNotification = useTestNotification();
  const { toast } = useToast();
  const readOnly = health?.read_only ?? true;

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
    auto_update_cron: "CRON_TZ=America/New_York 0 3 * * *",
  });

  useEffect(() => {
    if (settingsData?.notifications) {
      const merged = { ...settingsData.notifications };
      if (!merged.auto_update_cron) merged.auto_update_cron = "CRON_TZ=America/New_York 0 3 * * *";
      if (merged.auto_update_enabled) merged.auto_update_safe = true;
      if (settingsData.locked?.discord_webhook) merged.discord_enabled = true;
      if (settingsData.locked?.slack_webhook) merged.slack_enabled = true;
      setForm(merged);
    }
  }, [settingsData]);

  const set = <K extends keyof NotificationSettings>(key: K, value: NotificationSettings[K]) =>
    setForm((f) => {
      const next = { ...f, [key]: value };
      if (key === "auto_update_enabled" && value) {
        next.auto_update_safe = true;
      }
      return next;
    });

  const handleSave = async () => {
    try {
      await updateSettings.mutateAsync({ notifications: form });
      toast("Settings saved", "success");
    } catch (err) {
      toast(String(err), "error");
    }
  };

  const handleTest = async () => {
    try {
      await testNotification.mutateAsync();
      toast("Test notification sent", "success");
    } catch (err) {
      toast(String(err), "error");
    }
  };

  const activeChannels = [
    form.discord_enabled && form.discord_webhook && "Discord",
    form.slack_enabled && form.slack_webhook && "Slack",
  ].filter(Boolean) as string[];

  return (
    <div className="mx-auto max-w-2xl space-y-5">

      {/* ── Status bar ──────────────────────────────────── */}
      <div className="flex flex-wrap items-center gap-3 rounded-xl border border-ink-800/50 bg-ink-900/50 px-4 py-3 text-xs">
        <div className="flex items-center gap-1.5">
          <span className={`h-1.5 w-1.5 rounded-full ${readOnly ? "bg-amber-400" : "bg-emerald-400"}`} />
          <span className="text-ink-400">{readOnly ? "Read-only" : "Write enabled"}</span>
        </div>
        <span className="text-ink-700">·</span>
        <div className="flex items-center gap-1.5">
          {activeChannels.length > 0 ? (
            <>
              <span className="h-1.5 w-1.5 rounded-full bg-signal-400" />
              <span className="text-ink-400">{activeChannels.join(" + ")} active</span>
            </>
          ) : (
            <>
              <span className="h-1.5 w-1.5 rounded-full bg-ink-600" />
              <span className="text-ink-500">No notification channels configured</span>
            </>
          )}
        </div>
        {form.auto_update_enabled && (
          <>
            <span className="text-ink-700">·</span>
            <div className="flex items-center gap-1.5">
              <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-glow-pulse" />
              <span className="text-ink-400">Auto-update active</span>
            </div>
          </>
        )}
      </div>

      {/* ── Notification Channels ───────────────────────── */}
      <Section icon={Webhook} title="Notification Channels" description="Where Bulwark sends alerts and digests.">
        <div className="space-y-4">

          {/* Discord */}
          <div className={`rounded-xl border p-4 ${form.discord_enabled ? "border-signal-500/25 bg-signal-500/5" : "border-ink-800/60 bg-ink-950/40"}`}>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-sm font-medium text-ink-100">
                <Webhook className="h-4 w-4 text-[#5865F2]" />
                Discord
              </div>
              <Toggle
                checked={form.discord_enabled}
                onChange={(v) => set("discord_enabled", v)}
                disabled={readOnly || Boolean(locked?.discord_webhook)}
              />
            </div>
            <div className="mt-3">
              <label className="mb-1.5 block text-xs text-ink-500">Webhook URL</label>
              <Input
                value={form.discord_webhook}
                onChange={(e) => set("discord_webhook", e.target.value)}
                placeholder="https://discord.com/api/webhooks/…"
                type="password"
                disabled={readOnly || Boolean(locked?.discord_webhook)}
              />
            </div>
            {locked?.discord_webhook && (
              <div className="mt-2 flex items-center gap-1.5 text-xs text-emerald-400">
                <Lock className="h-3 w-3" />
                Managed via <code className="font-mono">DISCORD_WEBHOOK_URL</code>
              </div>
            )}
          </div>

          {/* Slack */}
          <div className={`rounded-xl border p-4 ${form.slack_enabled ? "border-signal-500/25 bg-signal-500/5" : "border-ink-800/60 bg-ink-950/40"}`}>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-sm font-medium text-ink-100">
                <Webhook className="h-4 w-4 text-[#4A154B]" />
                Slack
              </div>
              <Toggle
                checked={form.slack_enabled}
                onChange={(v) => set("slack_enabled", v)}
                disabled={readOnly || Boolean(locked?.slack_webhook)}
              />
            </div>
            <div className="mt-3">
              <label className="mb-1.5 block text-xs text-ink-500">Webhook URL</label>
              <Input
                value={form.slack_webhook}
                onChange={(e) => set("slack_webhook", e.target.value)}
                placeholder="https://hooks.slack.com/services/…"
                type="password"
                disabled={readOnly || Boolean(locked?.slack_webhook)}
              />
            </div>
            {locked?.slack_webhook && (
              <div className="mt-2 flex items-center gap-1.5 text-xs text-emerald-400">
                <Lock className="h-3 w-3" />
                Managed via <code className="font-mono">SLACK_WEBHOOK_URL</code>
              </div>
            )}
          </div>
        </div>
      </Section>

      {/* ── Alert Behavior ──────────────────────────────── */}
      <Section icon={BellRing} title="Alert Behavior" description="When and how often Bulwark sends notifications.">
        <div className="divide-y divide-ink-800/40">

          {/* Immediate alerts */}
          <div className="pb-4">
            <SettingRow
              label="Immediate alerts"
              description="Send a notification as soon as new image digests are detected."
              checked={form.notify_on_find}
              onChange={(v) => set("notify_on_find", v)}
              disabled={readOnly}
            />
            {form.notify_on_find && (
              <CronPicker
                label="Check schedule"
                value={form.check_cron}
                onChange={(v) => set("check_cron", v)}
                disabled={readOnly}
                presets={[
                  { label: "Every 15 min", value: "*/15 * * * *" },
                  { label: "Every hour",   value: "0 * * * *" },
                  { label: "Every 12h",    value: "0 */12 * * *" },
                  { label: "Daily 09:00",  value: "0 9 * * *" },
                ]}
              />
            )}
          </div>

          {/* Digest summaries */}
          <div className="pt-4">
            <SettingRow
              label="Digest summaries"
              description="Send a scheduled rollup of outstanding updates and blocked items."
              checked={form.digest_enabled}
              onChange={(v) => set("digest_enabled", v)}
              disabled={readOnly}
            />
            {form.digest_enabled && (
              <CronPicker
                label="Digest schedule"
                value={form.digest_cron}
                onChange={(v) => set("digest_cron", v)}
                disabled={readOnly}
                presets={[
                  { label: "Daily 09:00",      value: "0 9 * * *" },
                  { label: "Daily 17:00",      value: "0 17 * * *" },
                  { label: "Every 12h",        value: "0 */12 * * *" },
                  { label: "Weekdays 17:00",   value: "0 17 * * 1-5" },
                ]}
              />
            )}
          </div>
        </div>
      </Section>

      {/* ── Auto Update ─────────────────────────────────── */}
      <Section
        icon={RefreshCw}
        title="Auto Update"
        iconColor="text-emerald-400"
        description="Automatically apply image updates on a schedule — like Watchtower."
      >
        <div className="divide-y divide-ink-800/40">

          {/* Master toggle */}
          <div className="pb-4">
            <SettingRow
              label="Enable automatic updates"
              description="Activates scheduled safe updates for stateless services with health probes."
              checked={form.auto_update_enabled}
              onChange={(v) => set("auto_update_enabled", v)}
              disabled={readOnly}
            />
            {form.auto_update_enabled && (
              <CronPicker
                label="Update schedule"
                value={form.auto_update_cron}
                onChange={(v) => set("auto_update_cron", v)}
                disabled={readOnly}
                presets={[
                  { label: "Daily 03:00 ET",    value: "CRON_TZ=America/New_York 0 3 * * *" },
                  { label: "Every 6h",          value: "0 */6 * * *" },
                  { label: "Every 12h",         value: "0 */12 * * *" },
                  { label: "Weekly Sun 03:00",  value: "0 3 * * 0" },
                ]}
              />
            )}
          </div>

          {/* Tiers — only shown when enabled */}
          {form.auto_update_enabled && (
            <div className="space-y-3 pt-4">
              <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-3">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-sm font-medium text-ink-100">Safe containers</div>
                    <div className="mt-1 text-sm text-ink-400">
                      Included automatically. Stateless services with health probes are updated with rollback protection.
                    </div>
                  </div>
                  <div className="rounded-full border border-emerald-400/30 bg-emerald-400/10 px-2.5 py-1 text-xs font-medium text-emerald-300">
                    Always on
                  </div>
                </div>
              </div>

              {/* Unsafe tier */}
              <div className={`rounded-xl border p-3 ${form.auto_update_unsafe ? "border-amber-500/30 bg-amber-500/5" : "border-ink-800/60 bg-ink-950/40"}`}>
                <SettingRow
                  label="Unsafe containers"
                  description="Stateful, policy=notify, or probe-missing services. No rollback protection."
                  checked={form.auto_update_unsafe}
                  onChange={(v) => {
                    if (v) {
                      setUnsafeWarningPending(true);
                    } else {
                      set("auto_update_unsafe", false);
                    }
                  }}
                  disabled={readOnly}
                />
                {form.auto_update_unsafe && (
                  <div className="mt-2 flex items-start gap-2 rounded-lg bg-amber-400/8 px-3 py-2 text-xs text-amber-400">
                    <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                    Policy blocks and probe requirements are bypassed. All allowed updates are forced.
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </Section>

      {/* ── Disabled channels warning ────────────────────── */}
      {(form.notify_on_find || form.digest_enabled) && activeChannels.length === 0 && (
        <div className="flex items-start gap-2 rounded-xl border border-amber-500/25 bg-amber-500/8 px-4 py-3 text-sm text-amber-300">
          <BellOff className="mt-0.5 h-4 w-4 shrink-0" />
          Alert scheduling is on but no channels are enabled. Enable Discord or Slack above to receive notifications.
        </div>
      )}

      {/* ── Save bar ─────────────────────────────────────── */}
      <div className="flex flex-wrap items-center gap-3 rounded-xl border border-ink-800/50 bg-ink-900/50 px-4 py-3">
        <Button
          variant="primary"
          size="sm"
          onClick={handleSave}
          disabled={updateSettings.isPending || readOnly}
        >
          {updateSettings.isPending ? "Saving…" : "Save Settings"}
        </Button>
        <Button
          variant="secondary"
          size="sm"
          onClick={handleTest}
          disabled={testNotification.isPending || readOnly || activeChannels.length === 0}
        >
          <Send className="mr-1.5 h-3.5 w-3.5" />
          {testNotification.isPending ? "Sending…" : "Test Notification"}
        </Button>
        {readOnly && (
          <span className="ml-auto text-xs text-amber-400">
            Read-only mode — settings cannot be saved
          </span>
        )}
      </div>

      {/* ── Unsafe confirmation modal ─────────────────────── */}
      {unsafeWarningPending && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="mx-4 w-full max-w-md rounded-2xl border border-amber-500/40 bg-ink-900 p-6 shadow-2xl">
            <div className="flex items-start gap-3">
              <div className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-amber-400/15">
                <AlertTriangle className="h-4 w-4 text-amber-400" />
              </div>
              <div>
                <div className="font-semibold text-amber-300">Enable unsafe auto-updates?</div>
                <p className="mt-2 text-sm leading-6 text-ink-300">
                  This will automatically update containers that are{" "}
                  <strong className="text-ink-100">stateful</strong>,
                  have <strong className="text-ink-100">policy=notify</strong>, or are{" "}
                  <strong className="text-ink-100">missing health probes</strong>.
                  These containers have no automated rollback protection — data loss or service disruption is possible.
                </p>
              </div>
            </div>
            <div className="mt-5 flex justify-end gap-3">
              <Button variant="ghost" size="sm" onClick={() => setUnsafeWarningPending(false)}>
                Cancel
              </Button>
              <Button
                variant="secondary"
                size="sm"
                className="border-amber-500/30 bg-amber-500/10 text-amber-300 hover:bg-amber-500/20"
                onClick={() => {
                  set("auto_update_unsafe", true);
                  setUnsafeWarningPending(false);
                }}
              >
                I understand, enable it
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
