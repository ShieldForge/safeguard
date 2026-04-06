export interface BuildFormData {
  default_vault_addr: string;
  default_auth_method: string;
  default_vault_provider: string;
  default_mount_point: string;
  default_auth_role: string;
  default_auth_mount: string;
  default_policy_path: string;
  default_mapping_config: string;
  default_audit_log: string;
  default_allowed_pids: string;
  default_allowed_uids: string;
  default_debug: boolean;
  default_monitor: boolean;
  default_access_control: boolean;
  disable_cli_flags: boolean;
  default_log_file: string;
  default_log_max_size: string;
  default_log_max_backups: string;
  default_log_max_age: string;
  default_log_compress: string;
  default_ldap_username: string;
  default_ldap_password: string;
  default_vault_token: string;
  embed_policy_from_url: boolean;
  embed_policy_files: boolean;
  version: string;
  build_tag: string;
  target_os: string;
  target_arch: string;
  source_dir: string;
  work_dir: string;
  output_dir: string;
  output_filename: string;
  code_signing_enabled: string;
  certificate_path: string;
  certificate_password: string;
}

export interface BuildSuccess {
  binary_path: string;
  size: number;
  checksum: string;
}

export interface BuildResult {
  show: boolean;
  type: "success" | "error" | "loading" | "";
  message: string;
  data?: BuildSuccess;
}

export interface EnvStatus {
  status: "ready" | "not_ready" | "error";
  issues?: string[];
  info?: string[];
}

export interface SectionProps {
  formData: BuildFormData;
  onChange: (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
  ) => void;
}
