import type { LucideIcon } from "lucide-react";
import { cn } from "../lib/utils";

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description: string;
  className?: string;
}

export function EmptyState({ icon: Icon, title, description, className }: EmptyStateProps) {
  return (
    <div className={cn("flex flex-col items-center gap-3 py-12", className)}>
      <Icon className="h-12 w-12 text-ink-500" />
      <h3 className="font-display text-lg text-ink-200">{title}</h3>
      <p className="max-w-sm text-center text-sm text-ink-400">{description}</p>
    </div>
  );
}
