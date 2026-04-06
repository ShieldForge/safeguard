import { useState } from "react";

interface CollapsibleSectionProps {
  title: string;
  defaultOpen?: boolean;
  badge?: "required" | "important";
  description?: string;
  children: React.ReactNode;
}

export function CollapsibleSection({
  title,
  defaultOpen = false,
  badge,
  description,
  children,
}: CollapsibleSectionProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div className={`collapsible-section ${isOpen ? "open" : ""}`}>
      <button
        type="button"
        className="collapsible-header"
        onClick={() => setIsOpen(!isOpen)}
        aria-expanded={isOpen}
      >
        <span className="collapsible-chevron">{isOpen ? "▾" : "▸"}</span>
        <span className="collapsible-title">{title}</span>
        {badge && (
          <span className={`section-badge badge-${badge}`}>{badge}</span>
        )}
      </button>
      {isOpen && (
        <div className="collapsible-body">
          {description && <p className="section-description">{description}</p>}
          {children}
        </div>
      )}
    </div>
  );
}
