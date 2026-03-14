import { useMutation, useQuery } from "@tanstack/react-query";
import { apiFetch } from "./api";
import type {
  ApplyResponse,
  HealthResponse,
  HistoryResponse,
  OverviewResponse,
  Plan,
  Run,
  SettingsResponse,
  Target
} from "./types";

interface PlanQueryOptions {
  enabled?: boolean;
  refetchInterval?: number | false;
}

export function useHealth() {
  return useQuery({
    queryKey: ["health"],
    queryFn: () => apiFetch<HealthResponse>("/api/health"),
    refetchInterval: 30000
  });
}

export function useOverview() {
  return useQuery({
    queryKey: ["overview"],
    queryFn: () => apiFetch<OverviewResponse>("/api/overview"),
    refetchInterval: 60000
  });
}

export function useTargets() {
  return useQuery({
    queryKey: ["targets"],
    queryFn: async () => {
      const data = await apiFetch<{ targets: Target[] }>("/api/targets");
      return data.targets;
    }
  });
}

export function useTarget(id?: string) {
  return useQuery({
    queryKey: ["target", id],
    queryFn: () => apiFetch<Target>(`/api/targets/${id}`),
    enabled: Boolean(id)
  });
}

export function usePlan(options: PlanQueryOptions = {}) {
  return useQuery({
    queryKey: ["plan"],
    queryFn: () => apiFetch<Plan>("/api/plan", { method: "POST", body: JSON.stringify({}) }),
    enabled: options.enabled ?? true,
    refetchInterval: options.refetchInterval ?? 60000
  });
}

export function useRun(runId?: string) {
  return useQuery({
    queryKey: ["run", runId],
    queryFn: () => apiFetch<Run>(`/api/runs/${runId}`),
    enabled: Boolean(runId),
    refetchInterval: (query) => (query.state.data?.status === "running" ? 1000 : false),
    refetchIntervalInBackground: false,
    retry: true,
    retryDelay: 1000
  });
}

export function useHistory(page: number, pageSize: number, filters: Record<string, string>) {
  const params = new URLSearchParams({ page: String(page), page_size: String(pageSize), ...filters });
  return useQuery({
    queryKey: ["history", page, pageSize, filters],
    queryFn: () => apiFetch<HistoryResponse>(`/api/history?${params.toString()}`)
  });
}

export function useApply() {
  return useMutation({
    mutationFn: (payload: Record<string, unknown>) =>
      apiFetch<ApplyResponse>("/api/apply", { method: "POST", body: JSON.stringify(payload) })
  });
}

export function useSettings() {
  return useQuery({
    queryKey: ["settings"],
    queryFn: () => apiFetch<SettingsResponse>("/api/settings")
  });
}

export function useUpdateSettings() {
  return useMutation({
    mutationFn: (payload: SettingsResponse) =>
      apiFetch<SettingsResponse>("/api/settings", { method: "PUT", body: JSON.stringify(payload) })
  });
}

export function useTestNotification() {
  return useMutation({
    mutationFn: () => apiFetch("/api/notifications/test", { method: "POST" })
  });
}
