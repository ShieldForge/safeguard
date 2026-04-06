import type { SectionProps } from "../../types";
import { CollapsibleSection } from "../../components/CollapsibleSection";
import { FormGroup } from "../../components/FormGroup";

export function MountPathsSection({ formData, onChange }: SectionProps) {
  return (
    <CollapsibleSection
      title="Mount &amp; Paths"
      defaultOpen
      badge="important"
      description="Set where secrets appear on the filesystem and how Vault paths map to local directories."
    >
      <div className="form-row">
        <FormGroup
          label="Default Mount Point:"
          importance="important"
          tooltip="The drive letter (Windows, e.g. V:) or directory (Linux/macOS, e.g. /mnt/vault) where the virtual filesystem is mounted."
        >
          <input
            type="text"
            name="default_mount_point"
            value={formData.default_mount_point}
            onChange={onChange}
            placeholder="V: (Windows) or /mnt/vault (Linux)"
          />
        </FormGroup>

        <FormGroup
          label="Default Mapping Config:"
          tooltip="Path to a JSON file that maps Vault secret paths to local filesystem paths. See the path-mapping docs for the schema."
        >
          <input
            type="text"
            name="default_mapping_config"
            value={formData.default_mapping_config}
            onChange={onChange}
            placeholder="e.g., ./mapping.json"
          />
        </FormGroup>
      </div>
    </CollapsibleSection>
  );
}
