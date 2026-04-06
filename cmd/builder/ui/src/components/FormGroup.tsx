import { Tooltip } from "./Tooltip";

interface FormGroupProps {
  label: string;
  importance?: "required" | "important";
  tooltip?: string;
  children: React.ReactNode;
}

export function FormGroup({
  label,
  importance,
  tooltip,
  children,
}: FormGroupProps) {
  return (
    <div className="form-group">
      <label>
        {label}
        {importance === "required" && (
          <span className="field-badge field-required">required</span>
        )}
        {importance === "important" && (
          <span className="field-badge field-important">important</span>
        )}
        {tooltip && <Tooltip text={tooltip} />}
      </label>
      {children}
    </div>
  );
}
