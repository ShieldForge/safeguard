import type { SectionProps } from "../../types";
import { CollapsibleSection } from "../../components/CollapsibleSection";
import { FormGroup } from "../../components/FormGroup";

export function TargetPlatformSection({ formData, onChange }: SectionProps) {
  return (
    <CollapsibleSection
      title="Target Platform"
      defaultOpen
      badge="required"
      description="Select the operating system and CPU architecture for the compiled binary."
    >
      <div className="form-row">
        <FormGroup
          label="Target OS:"
          importance="required"
          tooltip="The operating system the binary will run on. Must match the target machine."
        >
          <select
            name="target_os"
            value={formData.target_os}
            onChange={onChange}
          >
            <option value="windows">Windows</option>
            <option value="linux">Linux</option>
            <option value="darwin">macOS</option>
          </select>
        </FormGroup>

        <FormGroup
          label="Target Architecture:"
          importance="required"
          tooltip="CPU architecture for the target machine. Use amd64 for most servers, arm64 for Apple Silicon or ARM boards."
        >
          <select
            name="target_arch"
            value={formData.target_arch}
            onChange={onChange}
          >
            <option value="amd64">amd64 (64-bit)</option>
            <option value="arm64">arm64 (Apple Silicon, ARM64)</option>
            <option value="386">386 (32-bit)</option>
          </select>
        </FormGroup>
      </div>
    </CollapsibleSection>
  );
}
