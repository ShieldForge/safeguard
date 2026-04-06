import type { SectionProps } from "../../types";
import { CollapsibleSection } from "../../components/CollapsibleSection";
import { FormGroup } from "../../components/FormGroup";

export function LoggingSection({ formData, onChange }: SectionProps) {
  return (
    <CollapsibleSection
      title="Logging"
      description="Configure logging defaults baked into the binary. Users can override these at runtime unless CLI flags are disabled."
    >
      <div className="form-row">
        <FormGroup
          label="Default Debug Logging:"
          tooltip="Enables verbose debug output. Useful during development but noisy in production."
        >
          <input
            type="checkbox"
            name="default_debug"
            checked={formData.default_debug}
            onChange={onChange}
            style={{ width: "auto" }}
          />
        </FormGroup>

        <FormGroup
          label="Default Process Monitoring:"
          tooltip="Monitors which processes access the mounted filesystem and logs the details."
        >
          <input
            type="checkbox"
            name="default_monitor"
            checked={formData.default_monitor}
            onChange={onChange}
            style={{ width: "auto" }}
          />
        </FormGroup>
      </div>

      <div className="form-row">
        <FormGroup
          label="Default Audit Log:"
          tooltip="Path for a dedicated audit log that records all secret-access events. Leave empty to disable."
        >
          <input
            type="text"
            name="default_audit_log"
            value={formData.default_audit_log}
            onChange={onChange}
            placeholder="e.g., ./vault-audit.log"
          />
        </FormGroup>

        <FormGroup
          label="Default Log File:"
          tooltip="Path for the main application log file with automatic rotation. Leave empty to log to stdout only."
        >
          <input
            type="text"
            name="default_log_file"
            value={formData.default_log_file}
            onChange={onChange}
            placeholder="./logs/safeguard.log (empty to disable file logging)"
          />
        </FormGroup>
      </div>

      <div className="form-row form-row-3">
        <FormGroup
          label="Max Log Size (MB):"
          tooltip="Maximum size in megabytes before the log file is rotated. Default is 100 MB."
        >
          <input
            type="number"
            name="default_log_max_size"
            value={formData.default_log_max_size}
            onChange={onChange}
            placeholder="100"
            min={1}
          />
        </FormGroup>

        <FormGroup
          label="Max Rotated Backups:"
          tooltip="Number of old log files to keep after rotation. Set to 0 to keep all rotated files."
        >
          <input
            type="number"
            name="default_log_max_backups"
            value={formData.default_log_max_backups}
            onChange={onChange}
            placeholder="5"
            min={0}
          />
        </FormGroup>

        <FormGroup
          label="Max Log Age (Days):"
          tooltip="Maximum number of days to retain old log files. Set to 0 to never delete based on age."
        >
          <input
            type="number"
            name="default_log_max_age"
            value={formData.default_log_max_age}
            onChange={onChange}
            placeholder="30"
            min={0}
          />
        </FormGroup>
      </div>

      <FormGroup
        label="Compress Rotated Logs:"
        tooltip="Whether to gzip-compress rotated log files to save disk space."
      >
        <select
          name="default_log_compress"
          value={formData.default_log_compress}
          onChange={onChange}
        >
          <option value="">-- Use code default (true) --</option>
          <option value="true">Yes</option>
          <option value="false">No</option>
        </select>
      </FormGroup>
    </CollapsibleSection>
  );
}
