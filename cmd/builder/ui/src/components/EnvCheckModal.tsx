import { useEffect } from "react";
import type { EnvStatus } from "../types";

interface EnvCheckModalProps {
  status: EnvStatus | null;
  isChecking: boolean;
  onClose: () => void;
  onRecheck: () => void;
}

interface CheckItem {
  label: string;
  passed: boolean;
  detail: string;
}

function deriveChecks(status: EnvStatus): CheckItem[] {
  const checks: CheckItem[] = [];
  const infoMap = new Map((status.info ?? []).map((s) => [s, true]));
  const issueSet = new Set(status.issues ?? []);

  // Go compiler
  const goInfo = [...infoMap.keys()].find((s) => s.startsWith("Go:"));
  const goIssue = [...issueSet].find((s) =>
    s.toLowerCase().includes("go compiler"),
  );
  if (goIssue) {
    checks.push({ label: "Go Compiler", passed: false, detail: goIssue });
  } else if (goInfo) {
    checks.push({ label: "Go Compiler", passed: true, detail: goInfo });
  }

  // GCC
  const gccInfo = [...infoMap.keys()].find((s) => s.startsWith("GCC:"));
  const gccIssue = [...issueSet].find((s) => s.toLowerCase().includes("gcc"));
  if (gccIssue) {
    checks.push({ label: "GCC (CGO)", passed: false, detail: gccIssue });
  } else if (gccInfo) {
    checks.push({ label: "GCC (CGO)", passed: true, detail: gccInfo });
  }

  // Source directory / go.mod
  const srcIssue = [...issueSet].find((s) =>
    s.toLowerCase().includes("go.mod"),
  );
  checks.push({
    label: "Source Directory",
    passed: !srcIssue,
    detail: srcIssue ?? "go.mod found in source directory",
  });

  // Output directory
  const outInfo = [...infoMap.keys()].find((s) =>
    s.toLowerCase().includes("output"),
  );
  checks.push({
    label: "Output Directory",
    passed: true,
    detail: outInfo ?? "Output directory is available",
  });

  // Catch any remaining issues not already categorised
  const handled = new Set([goIssue, gccIssue, srcIssue].filter(Boolean));
  for (const issue of issueSet) {
    if (!handled.has(issue)) {
      checks.push({ label: "Other", passed: false, detail: issue });
    }
  }

  return checks;
}

export function EnvCheckModal({
  status,
  isChecking,
  onClose,
  onRecheck,
}: EnvCheckModalProps) {
  // Close on Escape
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [onClose]);

  const checks = status ? deriveChecks(status) : [];
  const passed = checks.filter((c) => c.passed).length;
  const total = checks.length;
  const allPassed = status?.status === "ready";

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-panel" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h3>Build Environment Check</h3>
          <button className="modal-close" onClick={onClose} aria-label="Close">
            ×
          </button>
        </div>

        <div className="modal-body">
          {isChecking && (
            <div className="env-checking">
              <span className="env-checking-spinner" />
              <span>Running environment checks…</span>
            </div>
          )}

          {!isChecking && !status && (
            <p className="env-prompt">
              Click the button below to verify your build environment.
            </p>
          )}

          {!isChecking && status && (
            <>
              <div
                className={`env-summary ${allPassed ? "env-summary-pass" : "env-summary-fail"}`}
              >
                {allPassed
                  ? `✅ All checks passed (${passed}/${total})`
                  : `⚠️ ${passed} of ${total} checks passed`}
              </div>

              <ul className="env-check-list">
                {checks.map((check, idx) => (
                  <li
                    key={idx}
                    className={`env-check-item ${check.passed ? "check-pass" : "check-fail"}`}
                  >
                    <span className="check-icon">
                      {check.passed ? "✓" : "✗"}
                    </span>
                    <div className="check-content">
                      <strong>{check.label}</strong>
                      <span className="check-detail">{check.detail}</span>
                    </div>
                  </li>
                ))}
              </ul>
            </>
          )}
        </div>

        <div className="modal-footer">
          <button className="modal-btn modal-btn-secondary" onClick={onClose}>
            Close
          </button>
          <button
            className="modal-btn modal-btn-primary"
            onClick={onRecheck}
            disabled={isChecking}
          >
            {isChecking ? "Checking…" : status ? "Re-check" : "Run Check"}
          </button>
        </div>
      </div>
    </div>
  );
}
