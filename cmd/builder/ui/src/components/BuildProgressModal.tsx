import { useEffect, useRef } from "react";
import type { BuildSuccess } from "../types";

interface BuildProgressModalProps {
  isOpen: boolean;
  logLines: string[];
  status: "building" | "success" | "error";
  errorMessage?: string;
  result?: BuildSuccess;
  onClose: () => void;
}

export function BuildProgressModal({
  isOpen,
  logLines,
  status,
  errorMessage,
  result,
  onClose,
}: BuildProgressModalProps) {
  const logEndRef = useRef<HTMLDivElement>(null);

  // Auto-scroll log to bottom
  useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logLines]);

  // Close on Escape (only when build is finished)
  useEffect(() => {
    if (!isOpen) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape" && status !== "building") onClose();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [isOpen, status, onClose]);

  if (!isOpen) return null;

  const downloadUrl = result
    ? `/api/download/${encodeURIComponent(result.binary_path.split(/[\\/]/).pop()!)}`
    : undefined;

  return (
    <div
      className="modal-overlay"
      onClick={status !== "building" ? onClose : undefined}
    >
      <div className="build-modal-panel" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h3>
            {status === "building" && "🔨 Building…"}
            {status === "success" && "✅ Build Complete"}
            {status === "error" && "❌ Build Failed"}
          </h3>
          {status !== "building" && (
            <button
              className="modal-close"
              onClick={onClose}
              aria-label="Close"
            >
              ×
            </button>
          )}
        </div>

        <div className="build-modal-body">
          <div className="build-log">
            {logLines.map((line, i) => (
              <div key={i} className="build-log-line">
                {line}
              </div>
            ))}
            {status === "building" && (
              <div className="build-log-line build-log-cursor">▌</div>
            )}
            <div ref={logEndRef} />
          </div>

          {status === "error" && errorMessage && (
            <div className="build-error-banner">{errorMessage}</div>
          )}

          {status === "success" && result && (
            <div className="build-success-banner">
              <div className="build-success-details">
                <span>
                  <strong>Binary:</strong>{" "}
                  {result.binary_path.split(/[\\/]/).pop()}
                </span>
                <span>
                  <strong>Size:</strong>{" "}
                  {(result.size / 1024 / 1024).toFixed(2)} MB
                </span>
                <span className="build-checksum">
                  <strong>SHA256:</strong> {result.checksum}
                </span>
              </div>
            </div>
          )}
        </div>

        <div className="modal-footer">
          {status === "building" ? (
            <span className="build-modal-status">
              <span className="env-checking-spinner" /> Compiling…
            </span>
          ) : (
            <>
              <button
                className="modal-btn modal-btn-secondary"
                onClick={onClose}
              >
                Close
              </button>
              {status === "success" && downloadUrl && (
                <a
                  href={downloadUrl}
                  className="modal-btn modal-btn-primary build-download-btn"
                  download
                >
                  Download Now
                </a>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}
