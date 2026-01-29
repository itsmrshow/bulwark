import * as React from "react";
import { cn } from "../../lib/utils";

const Input = React.forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(
  ({ className, ...props }, ref) => (
    <input
      ref={ref}
      className={cn(
        "h-10 w-full rounded-md border border-ink-700 bg-ink-900/80 px-3 text-sm text-ink-100 placeholder:text-ink-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-signal-500",
        className
      )}
      {...props}
    />
  )
);
Input.displayName = "Input";

export { Input };
