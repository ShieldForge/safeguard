import type { SectionProps } from "../../types";
import { CollapsibleSection } from "../../components/CollapsibleSection";
import { FormGroup } from "../../components/FormGroup";

export function CodeSigningSection({ formData, onChange }: SectionProps) {
  return (
    <CollapsibleSection
      title="Code Signing"
      defaultOpen
      badge="required"
      description="Configure code signing options for the compiled binary. (Coming Soon)"
    >
      <div className="form-row disabled">
        <FormGroup
          label="Enable Code Signing:"
          importance="required"
          tooltip="Toggle to enable or disable code signing for the compiled binary."
        >
          <select
            name="code_signing_enabled"
            disabled
            value={formData.code_signing_enabled}
            onChange={onChange}
          >
            <option value="true">Enabled</option>
            <option value="false">Disabled</option>
          </select>
        </FormGroup>
      </div>

      {formData.code_signing_enabled === "true" && (
        <div className="form-row disabled">
          <FormGroup
            label="Certificate Path:"
            importance="required"
            tooltip="File path to the code signing certificate (e.g., .pfx or .pem file)."
          >
            <input
              type="text"
              name="certificate_path"
              disabled
              value={formData.certificate_path}
              onChange={onChange}
            />
          </FormGroup>
          <FormGroup
            label="Certificate Password:"
            importance="required"
            tooltip="Password for the code signing certificate."
          >
            <input
              type="password"
              name="certificate_password"
              value={formData.certificate_password}
              onChange={onChange}
            />
          </FormGroup>
        </div>
      )}
    </CollapsibleSection>
  );
}
