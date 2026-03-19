interface BulwarkLogoProps {
  className?: string;
}

/**
 * Inline SVG logo — abstract battlements/wall silhouette.
 * Use `className` to set size (e.g. "h-8 w-8") and color (e.g. "text-signal-500").
 * color is driven by `currentColor` so it inherits from the text color class.
 */
export function BulwarkLogo({ className = "h-8 w-8 text-signal-500" }: BulwarkLogoProps) {
  return (
    <svg
      viewBox="0 0 36 30"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      aria-label="Bulwark"
    >
      {/* Left merlon */}
      <rect x="1" y="1" width="9" height="13" rx="1.5" fill="currentColor" />
      {/* Center merlon */}
      <rect x="13.5" y="1" width="9" height="13" rx="1.5" fill="currentColor" />
      {/* Right merlon */}
      <rect x="26" y="1" width="9" height="13" rx="1.5" fill="currentColor" />
      {/* Base wall */}
      <rect x="1" y="17" width="34" height="12" rx="1.5" fill="currentColor" />
    </svg>
  );
}
