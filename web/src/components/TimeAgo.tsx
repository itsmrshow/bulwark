import { useSyncExternalStore } from "react";
import { timeAgo } from "../lib/timeago";
import { Tooltip, TooltipTrigger, TooltipContent } from "./ui/tooltip";

interface TimeAgoProps {
  date: string;
  className?: string;
}

const listeners = new Set<() => void>();
let intervalId: ReturnType<typeof setInterval> | null = null;

function emitTick() {
  for (const listener of listeners) {
    listener();
  }
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  if (intervalId === null) {
    intervalId = setInterval(emitTick, 30000);
  }

  return () => {
    listeners.delete(listener);
    if (listeners.size === 0 && intervalId !== null) {
      clearInterval(intervalId);
      intervalId = null;
    }
  };
}

function getSnapshot() {
  return Math.floor(Date.now() / 30000);
}

export function TimeAgo({ date, className }: TimeAgoProps) {
  useSyncExternalStore(subscribe, getSnapshot, () => 0);

  const full = new Date(date).toLocaleString();

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={className} title={full}>
          {timeAgo(date)}
        </span>
      </TooltipTrigger>
      <TooltipContent>{full}</TooltipContent>
    </Tooltip>
  );
}
