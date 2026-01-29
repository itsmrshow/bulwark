import { NavLink, Route, Routes } from "react-router-dom";
import {
  Activity,
  ClipboardList,
  History,
  LayoutDashboard,
  PlayCircle,
  Target
} from "lucide-react";
import { useHealth } from "./lib/queries";
import { ReadOnlyBanner } from "./components/ReadOnlyBanner";
import { TokenManager } from "./components/TokenManager";
import { OverviewPage } from "./pages/OverviewPage";
import { TargetsPage } from "./pages/TargetsPage";
import { PlanPage } from "./pages/PlanPage";
import { ApplyPage } from "./pages/ApplyPage";
import { HistoryPage } from "./pages/HistoryPage";

const navItems = [
  { to: "/", label: "Overview", icon: LayoutDashboard },
  { to: "/targets", label: "Targets", icon: Target },
  { to: "/plan", label: "Updates / Plan", icon: ClipboardList },
  { to: "/apply", label: "Apply", icon: PlayCircle },
  { to: "/history", label: "History", icon: History }
];

export default function App() {
  const { data: health } = useHealth();

  return (
    <div className="min-h-screen bg-ink-950 text-ink-100">
      <div className="flex min-h-screen">
        <aside className="hidden w-64 flex-col border-r border-ink-800 bg-ink-900/60 p-6 lg:flex">
          <div className="mb-10 flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-signal-500/20 text-signal-500">
              <Activity className="h-5 w-5" />
            </div>
            <div>
              <h1 className="font-display text-lg">Bulwark</h1>
              <p className="text-xs text-ink-400">Web Console</p>
            </div>
          </div>
          <nav className="flex flex-1 flex-col gap-2">
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <NavLink
                  key={item.to}
                  to={item.to}
                  className={({ isActive }) =>
                    `flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-semibold transition ${
                      isActive
                        ? "bg-ink-800 text-ink-100 shadow-glow"
                        : "text-ink-300 hover:bg-ink-800/60 hover:text-ink-100"
                    }`
                  }
                >
                  <Icon className="h-4 w-4" />
                  {item.label}
                </NavLink>
              );
            })}
          </nav>
          <div className="mt-10 text-xs text-ink-500">
            {health?.read_only ? "Read-only mode" : "Write mode enabled"}
          </div>
        </aside>

        <main className="flex-1">
          <div className="border-b border-ink-800/60 bg-ink-900/40 px-6 py-4">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
              <div>
                <h2 className="font-display text-2xl">Bulwark Web Console</h2>
                <p className="text-sm text-ink-300">Plan before apply. Observe everything.</p>
              </div>
              <TokenManager />
            </div>
          </div>

          <div className="border-b border-ink-800/60 bg-ink-900/40 px-6 py-3 lg:hidden">
            <div className="flex gap-2 overflow-x-auto">
              {navItems.map((item) => {
                const Icon = item.icon;
                return (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    className={({ isActive }) =>
                      `flex items-center gap-2 rounded-full px-3 py-2 text-xs font-semibold ${
                        isActive ? "bg-ink-800 text-ink-100" : "text-ink-300"
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

          <div className="px-6 py-6">
            <ReadOnlyBanner readOnly={health?.read_only ?? true} />
            <Routes>
              <Route path="/" element={<OverviewPage />} />
              <Route path="/targets" element={<TargetsPage />} />
              <Route path="/plan" element={<PlanPage readOnly={health?.read_only ?? true} />} />
              <Route path="/apply" element={<ApplyPage />} />
              <Route path="/history" element={<HistoryPage />} />
            </Routes>
          </div>
        </main>
      </div>
    </div>
  );
}
