import type { Step } from "../stores/steps";

interface StepProgressProps {
  steps: Step[];
  checkingEnv?: boolean;
  onEnvClick?: () => void;
}

export function StepProgress({
  steps,
  checkingEnv,
  onEnvClick,
}: StepProgressProps) {
  return (
    <nav className="step-progress" aria-label="Build progress">
      {steps.map((step, i) => {
        const isEnv = step.key === "env";
        const showSpinner = isEnv && checkingEnv;

        return (
          <div
            key={step.key}
            className={`step-item step-${step.status}${isEnv ? " step-clickable" : ""}`}
            onClick={isEnv && !checkingEnv ? onEnvClick : undefined}
            role={isEnv ? "button" : undefined}
            tabIndex={isEnv ? 0 : undefined}
            onKeyDown={
              isEnv && !checkingEnv
                ? (e) => {
                    if (e.key === "Enter") onEnvClick?.();
                  }
                : undefined
            }
          >
            <div className="step-indicator">
              {showSpinner ? (
                <span className="step-spinner" />
              ) : step.status === "complete" ? (
                <span className="step-check">✓</span>
              ) : (
                <span className="step-number">{i + 1}</span>
              )}
            </div>
            <div>
              <span className="step-label">
                {isEnv && step.status !== "complete"
                  ? "Check Environment"
                  : step.label}
              </span>
              {/* {isEnv && step.status !== 'complete' && !showSpinner && (
                <span className="step-action-hint">Click to run</span>
              )} */}
            </div>
            {i < steps.length - 1 && <div className="step-connector" />}
          </div>
        );
      })}
    </nav>
  );
}
