import { NavLink, Route, Routes, useLocation } from "react-router-dom";
import {
  ClipboardList,
  History,
  LayoutDashboard,
  PlayCircle,
  Settings,
  Target
} from "lucide-react";
import { useEffect } from "react";
import { useHealth } from "./lib/queries";
import { apiFetch } from "./lib/api";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { ReadOnlyBanner } from "./components/ReadOnlyBanner";
import { TokenManager } from "./components/TokenManager";
import { BulwarkLogo } from "./components/BulwarkLogo";
import { OverviewPage } from "./pages/OverviewPage";
import { TargetsPage } from "./pages/TargetsPage";
import { PlanPage } from "./pages/PlanPage";
import { ApplyPage } from "./pages/ApplyPage";
import { HistoryPage } from "./pages/HistoryPage";
import { SettingsPage } from "./pages/SettingsPage";

const navItems = [
  { to: "/",         label: "Overview",      icon: LayoutDashboard },
  { to: "/targets",  label: "Targets",       icon: Target },
  { to: "/plan",     label: "Updates",       icon: ClipboardList },
  { to: "/apply",    label: "Apply",         icon: PlayCircle },
  { to: "/history",  label: "History",       icon: History },
  { to: "/settings", label: "Settings",      icon: Settings }
];

function usePageTitle() {
  const { pathname } = useLocation();
  const match = navItems.find((item) =>
    item.to === "/" ? pathname === "/" : pathname.startsWith(item.to)
  );
  return match?.label ?? "Bulwark";
}

export default function App() {
  const { data: health } = useHealth();
  const pageTitle = usePageTitle();

  useEffect(() => {
    if (health && !health.read_only) {
      apiFetch("/api/enable-writes", { method: "POST" }).catch(() => {});
    }
  }, [health]);

  return (
    <div className="min-h-screen bg-ink-950 text-ink-100">
      <div className="flex min-h-screen">

        {/* ── Desktop sidebar ─────────────────────────────────────── */}
        <aside className="hidden w-56 flex-col border-r border-ink-800/50 bg-ink-950 lg:flex">
          {/* Logo */}
          <div className="flex items-center gap-3 px-5 py-6">
            <BulwarkLogo className="h-7 w-7 shrink-0 text-signal-500 drop-shadow-[0_0_8px_rgba(45,212,191,0.5)]" />
            <div>
              <span className="font-display text-[15px] font-semibold tracking-wide text-ink-100">
                Bulwark
              </span>
              <div className="text-[10px] font-medium uppercase tracking-[0.15em] text-ink-500">
                Web Console
              </div>
            </div>
          </div>

          {/* Top separator */}
          <div className="mx-5 h-px bg-ink-800/60" />

          {/* Nav */}
          <nav className="mt-3 flex flex-1 flex-col gap-0.5 px-3">
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.to === "/"}
                  className={({ isActive }) =>
                    isActive
                      ? "nav-active flex items-center gap-3 rounded-r-lg rounded-l-none px-3 py-2.5 text-sm font-medium pl-[calc(0.75rem-2px)]"
                      : "flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-ink-400 transition-colors hover:bg-ink-800/40 hover:text-ink-200"
                  }
                >
                  <Icon className="h-4 w-4 shrink-0" />
                  {item.label}
                </NavLink>
              );
            })}
          </nav>

          {/* Bottom status */}
          <div className="mx-5 mb-5 mt-4">
            <div className="h-px bg-ink-800/60 mb-4" />
            <div className="flex items-center gap-2">
              <span
                className={`h-2 w-2 rounded-full ${
                  health?.read_only ? "bg-amber-400" : "bg-emerald-400 animate-glow-pulse"
                }`}
              />
              <span className="text-xs text-ink-500">
                {health?.read_only ? "Read-only" : "Write mode"}
              </span>
            </div>
          </div>
        </aside>

        {/* ── Main content ─────────────────────────────────────────── */}
        <main className="flex min-w-0 flex-1 flex-col">

          {/* Top bar */}
          <header className="flex items-center justify-between border-b border-ink-800/50 bg-ink-950/80 px-6 py-3.5 backdrop-blur-sm">
            {/* Left: mobile logo + desktop page title */}
            <div className="flex items-center gap-3">
              {/* Mobile only: logo */}
              <div className="flex items-center gap-2.5 lg:hidden">
                <BulwarkLogo className="h-6 w-6 text-signal-500" />
                <span className="font-display text-base font-semibold text-ink-100">Bulwark</span>
                <span className="text-ink-700">·</span>
              </div>
              {/* Page title */}
              <h1 className="font-display text-base font-semibold text-ink-200 lg:text-lg">
                {pageTitle}
              </h1>
            </div>
            {/* Right: token / auth */}
            <TokenManager />
          </header>

          {/* Mobile nav */}
          <div className="border-b border-ink-800/50 bg-ink-950/60 px-4 py-2 lg:hidden">
            <div className="flex gap-1 overflow-x-auto pb-0.5">
              {navItems.map((item) => {
                const Icon = item.icon;
                return (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    end={item.to === "/"}
                    className={({ isActive }) =>
                      `flex shrink-0 items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                        isActive
                          ? "bg-signal-500/10 text-signal-400 ring-1 ring-signal-500/30"
                          : "text-ink-400 hover:bg-ink-800/40 hover:text-ink-200"
                      }`
                    }
                  >
                    <Icon className="h-3 w-3" />
                    {item.label}
                  </NavLink>
                );
              })}
            </div>
          </div>

          {/* Page content */}
          <div className="flex-1 overflow-auto px-6 py-6">
            <ReadOnlyBanner readOnly={health?.read_only ?? true} />
            <ErrorBoundary>
              <Routes>
                <Route path="/"         element={<OverviewPage />} />
                <Route path="/targets"  element={<TargetsPage />} />
                <Route path="/plan"     element={<PlanPage readOnly={health?.read_only ?? true} />} />
                <Route path="/apply"    element={<ApplyPage />} />
                <Route path="/history"  element={<HistoryPage />} />
                <Route path="/settings" element={<SettingsPage />} />
              </Routes>
            </ErrorBoundary>
          </div>
        </main>
      </div>
    </div>
  );
}
