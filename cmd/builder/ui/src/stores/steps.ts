import type { BuildFormData, EnvStatus } from "../types";

export interface Step {
  key: string;
  label: string;
  status: "complete" | "current" | "upcoming";
}

export function computeSteps(
  formData: BuildFormData,
  envStatus: EnvStatus | null,
): Step[] {
  const envDone = envStatus?.status === "ready";

  const connectionDone =
    formData.default_vault_addr !== "" ||
    formData.default_auth_method !== "" ||
    formData.default_vault_provider !== "";

  const mountDone = formData.default_mount_point !== "";

  const platformDone = formData.target_os !== "" && formData.target_arch !== "";

  const allRequiredDone = connectionDone && mountDone && platformDone;

  const steps: Step[] = [
    {
      key: "env",
      label: "Check Environment",
      status: envDone ? "complete" : "current",
    },
    {
      key: "connection",
      label: "Connection & Auth",
      status:
        envDone && connectionDone
          ? "complete"
          : envDone
            ? "current"
            : "upcoming",
    },
    {
      key: "mount",
      label: "Mount & Paths",
      status:
        connectionDone && mountDone
          ? "complete"
          : connectionDone
            ? "current"
            : "upcoming",
    },
    {
      key: "customize",
      label: "Customize Options",
      status: mountDone ? "complete" : "upcoming",
    },
    {
      key: "platform",
      label: "Target Platform",
      status:
        mountDone && platformDone
          ? "complete"
          : mountDone
            ? "current"
            : "upcoming",
    },
    {
      key: "code_signing",
      label: "Code Signing",
      status:
        platformDone && formData.code_signing_enabled === "true"
          ? "complete"
          : platformDone
            ? "current"
            : "upcoming",
    },
    {
      key: "build",
      label: "Build",
      status: allRequiredDone ? "current" : "upcoming",
    },
  ];

  // Ensure exactly one step is 'current': the first non-complete
  let foundCurrent = false;
  for (const step of steps) {
    if (step.status === "complete") continue;
    if (!foundCurrent) {
      step.status = "current";
      foundCurrent = true;
    } else {
      step.status = "upcoming";
    }
  }

  return steps;
}
