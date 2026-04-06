import { useState } from "react";

interface TooltipProps {
  text: string;
  children?: React.ReactNode;
}

export function Tooltip({ text, children }: TooltipProps) {
  const [visible, setVisible] = useState(false);

  return (
    <span
      className="tooltip-wrapper"
      onMouseEnter={() => setVisible(true)}
      onMouseLeave={() => setVisible(false)}
    >
      {children ?? (
        <span className="tooltip-icon" aria-label="Help">
          ?
        </span>
      )}
      {visible && <span className="tooltip-bubble">{text}</span>}
    </span>
  );
}
