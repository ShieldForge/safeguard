import type { EnvStatus } from "../types";

interface EnvironmentCheckProps {
  onCheck: () => void;
  status: EnvStatus | null;
  isChecking: boolean;
}

export function EnvironmentCheck({
  onCheck,
  status,
  isChecking,
}: EnvironmentCheckProps) {
  return (
    <div className="info-box">
      <strong>ℹ️ Prerequisites:</strong> Building requires Go and GCC (for
      CGO/cgofuse support).
      <br />
      <button className="check-btn" onClick={onCheck} disabled={isChecking}>
        {isChecking ? "Checking..." : "Check Build Environment"}
      </button>
      {status && (
        <div id="envStatus">
          {status.status === "ready" && (
            <p style={{ color: "green", marginTop: "10px" }}>
              ✅ Build environment is ready!
            </p>
          )}

          {status.status === "not_ready" && (
            <div style={{ marginTop: "10px" }}>
              <p style={{ color: "red" }}>❌ Build environment has issues:</p>
              <ul>
                {status.issues?.map((issue, idx) => (
                  <li key={idx} style={{ color: "red" }}>
                    {issue}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {status.status === "error" && (
            <p style={{ color: "red", marginTop: "10px" }}>
              ❌ Error: {status.issues?.[0]}
            </p>
          )}

          {status.info && status.info.length > 0 && (
            <div style={{ marginTop: "10px" }}>
              <p>
                <strong>Environment Info:</strong>
              </p>
              <ul>
                {status.info.map((info, idx) => (
                  <li key={idx}>{info}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
