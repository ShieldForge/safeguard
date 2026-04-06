import { describe, it, expect } from "vitest";
import { computeSteps } from "../stores/steps";
import type { BuildFormData, EnvStatus } from "../types";

function makeFormData(overrides: Partial<BuildFormData> = {}): BuildFormData {
    return {
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
        target_os: "",
        target_arch: "",
        source_dir: "",
        work_dir: "",
        output_dir: "",
        output_filename: "",
        code_signing_enabled: "false",
        certificate_path: "",
        certificate_password: "",
        ...overrides,
    };
}

describe("computeSteps", () => {
    it("marks env as current when envStatus is null", () => {
        const steps = computeSteps(makeFormData(), null);
        const envStep = steps.find((s) => s.key === "env");
        expect(envStep?.status).toBe("current");
    });

    it("marks env as complete when envStatus is ready", () => {
        const env: EnvStatus = { status: "ready" };
        const form = makeFormData({ default_vault_addr: "http://localhost:8200" });
        const steps = computeSteps(form, env);
        const envStep = steps.find((s) => s.key === "env");
        expect(envStep?.status).toBe("complete");
    });

    it("advances to connection step after env is ready", () => {
        const env: EnvStatus = { status: "ready" };
        const steps = computeSteps(makeFormData(), env);
        const connStep = steps.find((s) => s.key === "connection");
        expect(connStep?.status).toBe("current");
    });

    it("marks connection complete when vault addr is set", () => {
        const env: EnvStatus = { status: "ready" };
        const form = makeFormData({ default_vault_addr: "http://localhost:8200" });
        const steps = computeSteps(form, env);
        const connStep = steps.find((s) => s.key === "connection");
        expect(connStep?.status).toBe("complete");
    });

    it("marks mount step as current after connection is complete", () => {
        const env: EnvStatus = { status: "ready" };
        const form = makeFormData({ default_vault_addr: "http://localhost:8200" });
        const steps = computeSteps(form, env);
        const mountStep = steps.find((s) => s.key === "mount");
        expect(mountStep?.status).toBe("current");
    });

    it("progresses to code_signing step when platform is set", () => {
        const env: EnvStatus = { status: "ready" };
        const form = makeFormData({
            default_vault_addr: "http://localhost:8200",
            default_mount_point: "V:",
            target_os: "windows",
            target_arch: "amd64",
        });
        const steps = computeSteps(form, env);
        const codeSigningStep = steps.find((s) => s.key === "code_signing");
        expect(codeSigningStep?.status).toBe("current");
    });

    it("returns exactly one current step", () => {
        const env: EnvStatus = { status: "ready" };
        const form = makeFormData({ default_vault_addr: "http://localhost:8200" });
        const steps = computeSteps(form, env);
        const currentSteps = steps.filter((s) => s.status === "current");
        expect(currentSteps).toHaveLength(1);
    });

    it("returns expected step keys in order", () => {
        const steps = computeSteps(makeFormData(), null);
        const keys = steps.map((s) => s.key);
        expect(keys).toEqual([
            "env",
            "connection",
            "mount",
            "customize",
            "platform",
            "code_signing",
            "build",
        ]);
    });
});
