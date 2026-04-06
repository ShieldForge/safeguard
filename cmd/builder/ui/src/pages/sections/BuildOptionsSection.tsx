import type { SectionProps } from "../../types";
import { CollapsibleSection } from "../../components/CollapsibleSection";
import { FormGroup } from "../../components/FormGroup";

export function BuildOptionsSection({ formData, onChange }: SectionProps) {
  return (
    <CollapsibleSection
      title="Build Options"
      defaultOpen
      description="Options that affect the compiled binary's behaviour."
    >
      <FormGroup
        label="Output Filename:"
        tooltip="Custom name for the built binary. Leave blank to auto-generate from version, OS, and architecture."
      >
        <input
          type="text"
          name="output_filename"
          value={formData.output_filename}
          onChange={onChange}
          placeholder="e.g. safeguard-custom"
        />
      </FormGroup>
      <FormGroup
        label="Disable CLI Flags (Build-only):"
        tooltip="When enabled, the built binary ignores all command-line arguments and uses only the defaults you set here. Useful for locked-down deployments."
      >
        <input
          type="checkbox"
          name="disable_cli_flags"
          checked={formData.disable_cli_flags}
          onChange={onChange}
          style={{ width: "auto" }}
        />
        <span className="field-hint">
          When enabled, the binary ignores all CLI arguments and uses only
          embedded defaults.
        </span>
      </FormGroup>
      <FormGroup
        label="Embed Policy Files:"
        tooltip="When enabled and a policy path is set, all .rego files are embedded into the binary. The binary will use them automatically without needing a policy path at runtime."
      >
        <input
          type="checkbox"
          name="embed_policy_files"
          checked={formData.embed_policy_files}
          onChange={onChange}
          style={{ width: "auto" }}
        />
        <span className="field-hint">
          Embed policy .rego files into the binary so no external policy path is
          needed at runtime.
        </span>
      </FormGroup>
    </CollapsibleSection>
  );
}
