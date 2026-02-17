import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { StatusPill } from "../StatusPill";

describe("StatusPill", () => {
  it("renders running status", () => {
    render(<StatusPill status="running" />);
    expect(screen.getByText("running")).toBeInTheDocument();
  });

  it("renders completed status", () => {
    render(<StatusPill status="completed" />);
    expect(screen.getByText("completed")).toBeInTheDocument();
  });

  it("renders failed status", () => {
    render(<StatusPill status="failed" />);
    expect(screen.getByText("failed")).toBeInTheDocument();
  });

  it("renders unknown for undefined status", () => {
    render(<StatusPill />);
    expect(screen.getByText("unknown")).toBeInTheDocument();
  });
});
