import { useState } from "react";
import { useBuildForm } from "../stores/useBuildForm";
import { EnvCheckModal } from "../components/EnvCheckModal";
import { BuildProgressModal } from "../components/BuildProgressModal";
import { StepProgress } from "../components/StepProgress";
import { computeSteps } from "../stores/steps";
import { ConnectionAuthSection } from "./sections/ConnectionAuthSection";
import { MountPathsSection } from "./sections/MountPathsSection";
import { AccessControlSection } from "./sections/AccessControlSection";
import { LoggingSection } from "./sections/LoggingSection";
import { BuildOptionsSection } from "./sections/BuildOptionsSection";
import { BuildMetadataSection } from "./sections/BuildMetadataSection";
import { TargetPlatformSection } from "./sections/TargetPlatformSection";
import { CheckEnvSection } from "./sections/CheckEnvSection";
import { CodeSigningSection } from "./sections/CodeSigningSection";

export function BuildPage() {
  const {
    formData,
    isBuilding,
    envStatus,
    checkingEnv,
    buildLog,
    buildModalStatus,
    buildErrorMessage,
    buildResult,
    showBuildModal,
    handleInputChange,
    checkEnvironment,
    handleSubmit,
    closeBuildModal,
  } = useBuildForm();

  const [showEnvModal, setShowEnvModal] = useState(false);
  const steps = computeSteps(formData, envStatus);

  const handleEnvCheck = () => {
    checkEnvironment();
    setShowEnvModal(true);
  };

  return (
    <div className="dashboard">
      <aside className="sidebar">
        <div className="sidebar-brand">
          <h1>Safeguard Mount</h1>
          <p>Custom Build Server</p>
        </div>
        <StepProgress
          steps={steps}
          checkingEnv={checkingEnv}
          onEnvClick={handleEnvCheck}
        />
        <div className="sidebar-footer">
          <button
            type="submit"
            form="build-form"
            className="sidebar-build-btn"
            disabled={isBuilding}
          >
            {isBuilding ? "Building…" : "Build Binary"}
          </button>
        </div>
      </aside>

      <main className="main-content">
        <header className="main-header">
          <h2>Configure Build</h2>
          <p>
            Set default configuration values that will be compiled into the
            binary.
          </p>
        </header>

        <CheckEnvSection
          envStatus={envStatus}
          isChecking={checkingEnv}
          formData={formData}
          onChange={handleInputChange}
          onCheck={handleEnvCheck}
        />

        <form id="build-form" onSubmit={handleSubmit}>
          <ConnectionAuthSection
            formData={formData}
            onChange={handleInputChange}
          />
          <MountPathsSection formData={formData} onChange={handleInputChange} />
          <AccessControlSection
            formData={formData}
            onChange={handleInputChange}
          />
          <LoggingSection formData={formData} onChange={handleInputChange} />

          <div className="section-row">
            <BuildOptionsSection
              formData={formData}
              onChange={handleInputChange}
            />
            <BuildMetadataSection
              formData={formData}
              onChange={handleInputChange}
            />
          </div>

          <TargetPlatformSection
            formData={formData}
            onChange={handleInputChange}
          />

          <CodeSigningSection
            formData={formData}
            onChange={handleInputChange}
          />

          <button
            type="submit"
            className="main-build-btn"
            disabled={isBuilding}
          >
            {isBuilding ? "Building…" : "Build Custom Binary"}
          </button>
        </form>
      </main>

      <BuildProgressModal
        isOpen={showBuildModal}
        logLines={buildLog}
        status={buildModalStatus}
        errorMessage={buildErrorMessage}
        result={buildResult}
        onClose={closeBuildModal}
      />

      {showEnvModal && (
        <EnvCheckModal
          status={envStatus}
          isChecking={checkingEnv}
          onClose={() => setShowEnvModal(false)}
          onRecheck={checkEnvironment}
        />
      )}
    </div>
  );
}
