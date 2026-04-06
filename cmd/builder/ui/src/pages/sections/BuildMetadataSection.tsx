import type { SectionProps } from "../../types";
import { CollapsibleSection } from "../../components/CollapsibleSection";
import { FormGroup } from "../../components/FormGroup";

export function BuildMetadataSection({ formData, onChange }: SectionProps) {
  return (
    <CollapsibleSection
      title="Build Metadata"
      defaultOpen
      description="Metadata stamped into the binary at compile time. Useful for tracking deployments."
    >
      <div className="form-row">
        <FormGroup
          label="Version:"
          tooltip="Semantic version string (e.g. 1.0.0) embedded in the binary. Shown in --version output."
        >
          <input
            type="text"
            name="version"
            value={formData.version}
            onChange={onChange}
            placeholder="1.0.0"
          />
        </FormGroup>

        <FormGroup
          label="Build Tag:"
          tooltip="A custom tag (e.g. customer name or environment) stamped into the binary for identification."
        >
          <input
            type="text"
            name="build_tag"
            value={formData.build_tag}
            onChange={onChange}
            placeholder="customer-name or custom-tag"
          />
        </FormGroup>
      </div>
    </CollapsibleSection>
  );
}
