import type { SectionProps } from "../../types";
import { CollapsibleSection } from "../../components/CollapsibleSection";
import { FormGroup } from "../../components/FormGroup";
import { isURL } from "../../utils/helpers";

export function AccessControlSection({ formData, onChange }: SectionProps) {
  return (
    <CollapsibleSection
      title="Access Control &amp; Policies"
      description="Restrict which processes or users can read secrets from the mounted filesystem. OPA/Rego policies provide the most flexible access control."
    >
      <FormGroup
        label="Default Access Control:"
        tooltip="When enabled, the filesystem checks each file access against process identity and policy rules before allowing reads."
      >
        <input
          type="checkbox"
          name="default_access_control"
          checked={formData.default_access_control}
          onChange={onChange}
          style={{ width: "auto" }}
        />
      </FormGroup>

      <FormGroup
        label="Default Policy Path:"
        tooltip="Path to an OPA/Rego policy file or directory. Can be a local path or HTTPS URL. See the policy docs for examples."
      >
        <input
          type="text"
          name="default_policy_path"
          value={formData.default_policy_path}
          onChange={onChange}
          placeholder="e.g., ./policies or https://example.com/policy.rego"
        />
        {formData.default_policy_path &&
          isURL(formData.default_policy_path) && (
            <div style={{ marginTop: "10px" }}>
              <label
                style={{
                  display: "flex",
                  alignItems: "center",
                  fontWeight: "normal",
                }}
              >
                <input
                  type="checkbox"
                  name="embed_policy_from_url"
                  checked={formData.embed_policy_from_url}
                  onChange={onChange}
                  style={{ width: "auto", marginRight: "8px" }}
                />
                Embed policy from URL into binary
              </label>
              <div
                className="info-box"
                style={{ marginTop: "10px", fontSize: "0.9em" }}
              >
                ℹ️ When checked, the policy will be downloaded and embedded into
                the binary at build time. When unchecked, the binary will
                download and cache the policy at runtime.
              </div>
            </div>
          )}
      </FormGroup>

      <div className="form-row">
        <FormGroup
          label="Default Allowed PIDs:"
          tooltip="Legacy: comma-separated list of process IDs allowed to access secrets. Prefer OPA policies instead."
        >
          <input
            type="text"
            name="default_allowed_pids"
            value={formData.default_allowed_pids}
            onChange={onChange}
            placeholder="Comma-separated list (legacy)"
          />
        </FormGroup>

        <FormGroup
          label="Default Allowed UIDs:"
          tooltip="Legacy: comma-separated list of Unix user IDs allowed to access secrets. Prefer OPA policies instead."
        >
          <input
            type="text"
            name="default_allowed_uids"
            value={formData.default_allowed_uids}
            onChange={onChange}
            placeholder="Comma-separated list (legacy)"
          />
        </FormGroup>
      </div>
    </CollapsibleSection>
  );
}
