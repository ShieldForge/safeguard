import type { EnvStatus, BuildFormData } from "../../types";
import { CollapsibleSection } from "../../components/CollapsibleSection";
import { FormGroup } from "../../components/FormGroup";

interface CheckEnvSectionProps {
  envStatus: EnvStatus | null;
  isChecking: boolean;
  formData: BuildFormData;
  onChange: (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
  ) => void;
  onCheck: () => void;
}

export function CheckEnvSection({
  envStatus,
  isChecking,
  formData,
  onChange,
  onCheck,
}: CheckEnvSectionProps) {
  const isReady = envStatus?.status === "ready";

  return (
    <CollapsibleSection
      title="Check Environment"
      defaultOpen
      badge={isReady ? undefined : "required"}
      description="Building requires Go and GCC (for CGO/cgofuse support) to be installed and available in your PATH. Run the check below to verify your environment before configuring the build."
    >
      <div className="env-section-requirements">
        <ul className="env-req-list">
          <li>
            <strong>Go compiler</strong> — 1.21 or later recommended
          </li>
          <li>
            <strong>GCC / CGO toolchain</strong> — MinGW-w64 on Windows,
            build-essential on Linux, Xcode CLI tools on macOS
          </li>
          <li>
            <strong>Source directory</strong> — must contain a valid go.mod
          </li>
        </ul>
      </div>

      <div className="env-dir-fields">
        <FormGroup
          label="Source Directory:"
          tooltip="Path to the safeguard project root containing go.mod."
        >
          <input
            type="text"
            name="source_dir"
            value={formData.source_dir}
            onChange={onChange}
            placeholder="Auto-detected"
          />
        </FormGroup>
        <FormGroup
          label="Work Directory:"
          tooltip="Temporary directory used during builds. Defaults to build-work inside the source directory."
        >
          <input
            type="text"
            name="work_dir"
            value={formData.work_dir}
            onChange={onChange}
            placeholder="Default: <source>/build-work"
          />
        </FormGroup>
        <FormGroup
          label="Output Directory:"
          tooltip="Where built binaries are placed. Defaults to build-output inside the source directory."
        >
          <input
            type="text"
            name="output_dir"
            value={formData.output_dir}
            onChange={onChange}
            placeholder="Default: <source>/build-output"
          />
        </FormGroup>
      </div>

      <button
        type="button"
        className="env-check-btn"
        onClick={onCheck}
        disabled={isChecking}
      >
        {isChecking
          ? "Checking…"
          : isReady
            ? "✅ Re-check Environment"
            : "Check Build Environment"}
      </button>

      {isReady && <span className="env-ready-badge">Environment ready</span>}
    </CollapsibleSection>
  );
}
