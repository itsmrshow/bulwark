import { useEffect, useState } from "react";
import { timeAgo } from "../lib/timeago";
import { Tooltip, TooltipTrigger, TooltipContent } from "./ui/tooltip";

interface TimeAgoProps {
  date: string;
  className?: string;
}

export function TimeAgo({ date, className }: TimeAgoProps) {
  const [, setTick] = useState(0);

  useEffect(() => {
    const id = setInterval(() => setTick((t) => t + 1), 30000);
    return () => clearInterval(id);
  }, []);

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
