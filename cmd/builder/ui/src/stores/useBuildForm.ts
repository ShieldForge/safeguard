import { useState, useEffect, useCallback } from "react";
import type {
  BuildFormData,
  BuildResult,
  BuildSuccess,
  EnvStatus,
} from "../types";

const initialFormData: BuildFormData = {
  default_vault_addr: "",
  default_auth_method: "",
  default_vault_provider: "",
  default_mount_point: "",
  default_auth_role: "",
  default_auth_mount: "",
  default_policy_path: "",
  default_mapping_config: "",
  default_audit_log: "",
  default_allowed_pids: "",
  default_allowed_uids: "",
  default_debug: false,
  default_monitor: false,
  default_access_control: false,
  disable_cli_flags: false,
  default_log_file: "",
  default_log_max_size: "",
  default_log_max_backups: "",
  default_log_max_age: "",
  default_log_compress: "",
  default_ldap_username: "",
  default_ldap_password: "",
  default_vault_token: "",
  embed_policy_from_url: false,
  embed_policy_files: false,
  version: "",
  build_tag: "",
  target_os: "windows",
  target_arch: "amd64",
  source_dir: "",
  work_dir: "",
  output_dir: "",
  output_filename: "",
  code_signing_enabled: "false",
  certificate_path: "",
  certificate_password: "",
};

export function useBuildForm() {
  const [formData, setFormData] = useState<BuildFormData>(initialFormData);
  const [result, setResult] = useState<BuildResult>({
    show: false,
    type: "",
    message: "",
  });
  const [isBuilding, setIsBuilding] = useState(false);
  const [envStatus, setEnvStatus] = useState<EnvStatus | null>(null);
  const [checkingEnv, setCheckingEnv] = useState(false);
  const [buildLog, setBuildLog] = useState<string[]>([]);
  const [buildModalStatus, setBuildModalStatus] = useState<
    "building" | "success" | "error"
  >("building");
  const [buildErrorMessage, setBuildErrorMessage] = useState("");
  const [buildResult, setBuildResult] = useState<BuildSuccess | undefined>();
  const [showBuildModal, setShowBuildModal] = useState(false);

  // Load server defaults for directory paths on mount
  useEffect(() => {
    fetch("/api/validate")
      .then((res) => res.json())
      .then((data) => {
        setEnvStatus(data);
        setFormData((prev) => ({
          ...prev,
          source_dir: prev.source_dir || data.source || "",
          work_dir: prev.work_dir || data.work || "",
          output_dir: prev.output_dir || data.output || "",
        }));
      })
      .catch(() => { });
  }, []);

  const handleInputChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
  ) => {
    const target = e.target;
    const name = target.name;
    const value =
      target instanceof HTMLInputElement && target.type === "checkbox"
        ? target.checked
        : target.value;
    setFormData((prev) => {
      const next = { ...prev, [name]: value };
      return next;
    });
  };

  const checkEnvironment = async () => {
    setCheckingEnv(true);
    try {
      const response = await fetch("/api/validate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          source_dir: formData.source_dir,
          work_dir: formData.work_dir,
          output_dir: formData.output_dir,
        }),
      });
      const data = await response.json();
      setEnvStatus(data);
      // Update form fields from validated/resolved paths
      setFormData((prev) => ({
        ...prev,
        source_dir: data.source || prev.source_dir,
        work_dir: data.work || prev.work_dir,
        output_dir: data.output || prev.output_dir,
      }));
    } catch (error) {
      setEnvStatus({ status: "error", issues: [(error as Error).message] });
    } finally {
      setCheckingEnv(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsBuilding(true);
    setBuildLog([]);
    setBuildModalStatus("building");
    setBuildErrorMessage("");
    setBuildResult(undefined);
    setShowBuildModal(true);
    setResult({ show: false, type: "", message: "" });

    try {
      const response = await fetch("/api/build-stream", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(formData),
      });

      if (!response.ok) {
        const data = await response
          .json()
          .catch(() => ({ error: response.statusText }));
        setBuildErrorMessage(data.error || "Unknown error");
        setBuildModalStatus("error");
        setIsBuilding(false);
        return;
      }

      const reader = response.body?.getReader();
      const decoder = new TextDecoder();
      if (!reader) throw new Error("No response body");

      let buffer = "";
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const parts = buffer.split("\n\n");
        buffer = parts.pop() ?? "";

        for (const part of parts) {
          const lines = part.split("\n");
          let eventType = "message";
          let data = "";
          for (const line of lines) {
            if (line.startsWith("event: ")) eventType = line.slice(7);
            else if (line.startsWith("data: ")) data += line.slice(6);
          }

          if (eventType === "done") {
            const result = JSON.parse(data);
            setBuildResult(result);
            setBuildModalStatus("success");
            setResult({
              show: true,
              type: "success",
              message: "",
              data: result,
            });
          } else if (eventType === "error") {
            const errData = JSON.parse(data);
            setBuildErrorMessage(errData.error || "Build failed");
            setBuildModalStatus("error");
            setResult({
              show: true,
              type: "error",
              message: errData.error || "Build failed",
            });
          } else if (data) {
            setBuildLog((prev) => [...prev, data]);
          }
        }
      }
    } catch (error) {
      setBuildErrorMessage((error as Error).message);
      setBuildModalStatus("error");
    } finally {
      setIsBuilding(false);
    }
  };

  const closeBuildModal = useCallback(() => setShowBuildModal(false), []);

  return {
    formData,
    result,
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
  };
}
