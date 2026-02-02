export type RiskLevel = "safe" | "notify" | "stateful" | "probe_missing";

export interface HealthResponse {
  status: string;
  read_only: boolean;
  ui_enabled: boolean;
}

export interface OverviewResponse {
  generated_at: string;
  read_only: boolean;
  managed_targets: number;
  managed_services: number;
  updates_available: number;
  last_run?: {
    completed_at: string;
    status: string;
  };
  failures: number;
  rollbacks: number;
  activity: Array<{
    ts: string;
    action: string;
    target?: string;
    service?: string;
    message: string;
  }>;
}

export interface Target {
  id: string;
  type: string;
  name: string;
  path: string;
  services: Service[];
  labels: Labels;
  created_at: string;
  updated_at: string;
}

export interface Service {
  id: string;
  target_id: string;
  name: string;
  image: string;
  current_digest: string;
  labels: Labels;
}

export interface Labels {
  enabled: boolean;
  policy: string;
  tier: string;
  probe: ProbeConfig;
  definition?: string;
}

export interface ProbeConfig {
  type: string;
  http_url?: string;
  http_status?: number;
  tcp_host?: string;
  tcp_port?: number;
  log_pattern?: string;
  window_sec?: number;
  stability_sec?: number;
}

export interface Plan {
  generated_at: string;
  target_count: number;
  service_count: number;
  update_count: number;
  allowed_count: number;
  items: PlanItem[];
}

export interface PlanItem {
  target_id: string;
  target_name: string;
  target_type: string;
  service_id: string;
  service_name: string;
  image: string;
  current_digest: string;
  remote_digest: string;
  update_available: boolean;
  allowed: boolean;
  policy: string;
  tier: string;
  probe: ProbeConfig;
  reason: string;
  risk: RiskLevel;
  warnings?: string[];
}

export interface ApplyResponse {
  run_id: string;
}

export interface RunEvent {
  ts: string;
  level: string;
  target?: string;
  service?: string;
  step?: string;
  message: string;
  data?: Record<string, unknown>;
}

export interface Run {
  id: string;
  mode: string;
  status: string;
  created_at: string;
  started_at: string;
  completed_at?: string;
  summary: {
    updates_applied: number;
    updates_skipped: number;
    updates_failed: number;
    rollbacks: number;
  };
  events: RunEvent[];
}

export interface HistoryItem {
  target_id: string;
  service_id: string;
  service_name: string;
  old_digest: string;
  new_digest: string;
  success: boolean;
  rolled_back: boolean;
  error_message?: string;
  started_at: string;
  completed_at: string;
  probes_passed: number;
  probes_failed: number;
  duration_sec: number;
}

export interface HistoryResponse {
  page: number;
  page_size: number;
  has_more: boolean;
  items: HistoryItem[];
}

export interface NotificationSettings {
  discord_webhook: string;
  slack_webhook: string;
  discord_enabled: boolean;
  slack_enabled: boolean;
  notify_on_find: boolean;
  digest_enabled: boolean;
  check_cron: string;
  digest_cron: string;
}

export interface SettingsResponse {
  notifications: NotificationSettings;
  locked?: NotificationSettings;
}
