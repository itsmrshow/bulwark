import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { TooltipProvider } from "../ui/tooltip";
import { RiskBadge } from "../RiskBadge";

function renderWithProviders(ui: React.ReactElement) {
  return render(<TooltipProvider>{ui}</TooltipProvider>);
}

describe("RiskBadge", () => {
  it("renders Safe for safe risk", () => {
    renderWithProviders(<RiskBadge risk="safe" />);
    expect(screen.getByText("Safe")).toBeInTheDocument();
  });

  it("renders Notify for notify risk", () => {
    renderWithProviders(<RiskBadge risk="notify" />);
    expect(screen.getByText("Notify")).toBeInTheDocument();
  });

  it("renders Stateful for stateful risk", () => {
    renderWithProviders(<RiskBadge risk="stateful" />);
    expect(screen.getByText("Stateful")).toBeInTheDocument();
  });

  it("renders Probe Missing for probe_missing risk", () => {
    renderWithProviders(<RiskBadge risk="probe_missing" />);
    expect(screen.getByText("Probe Missing")).toBeInTheDocument();
  });
});
