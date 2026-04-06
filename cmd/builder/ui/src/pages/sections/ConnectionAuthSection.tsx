import type { SectionProps } from "../../types";
import { CollapsibleSection } from "../../components/CollapsibleSection";
import { FormGroup } from "../../components/FormGroup";

export function ConnectionAuthSection({ formData, onChange }: SectionProps) {
  const showLdapFields = formData.default_auth_method === "ldap";
  const showTokenFields = formData.default_auth_method === "token";

  return (
    <CollapsibleSection
      title="Connection &amp; Authentication"
      defaultOpen
      badge="required"
      description="Configure how the binary connects to your secrets vault and authenticates. These values become the compiled-in defaults — users can still override them at runtime unless CLI flags are disabled."
    >
      <FormGroup
        label="Default Vault Address:"
        importance="important"
        tooltip="The URL of your Vault server (e.g. http://127.0.0.1:8200). Users can override with -vault-addr or VAULT_ADDR at runtime."
      >
        <input
          type="text"
          name="default_vault_addr"
          value={formData.default_vault_addr}
          onChange={onChange}
          placeholder="http://127.0.0.1:8200"
        />
      </FormGroup>

      <div className="form-row">
        <FormGroup
          label="Default Auth Method:"
          importance="important"
          tooltip="How users authenticate to Vault. OIDC opens a browser flow, LDAP prompts for credentials, Token uses a static token."
        >
          <select
            name="default_auth_method"
            value={formData.default_auth_method}
            onChange={onChange}
          >
            <option value="">-- Use code default --</option>
            <option value="oidc">OIDC</option>
            <option value="ldap">LDAP</option>
            <option value="token">Token</option>
            <option value="aws">AWS</option>
            <option value="approle">AppRole</option>
          </select>
        </FormGroup>

        <FormGroup
          label="Default Vault Provider:"
          importance="important"
          tooltip="Which secrets backend to use. HashiCorp Vault is the default. Cloud providers use their native SDK for authentication."
        >
          <select
            name="default_vault_provider"
            value={formData.default_vault_provider}
            onChange={onChange}
          >
            <option value="">-- Use code default (hashicorp) --</option>
            <option value="hashicorp">HashiCorp Vault</option>
            <option value="aws-secrets-manager">AWS Secrets Manager</option>
            <option value="gcp-secret-manager">GCP Secret Manager</option>
            <option value="azure-keyvault">Azure Key Vault</option>
          </select>
        </FormGroup>
      </div>

      {showLdapFields && (
        <div className="conditional-field visible">
          <div className="security-warning">
            ⚠️ <strong>Security Note:</strong> These credentials will be
            embedded in the binary. Only use this for testing or if you
            understand the security implications. For production, leave empty
            and users will be prompted at runtime.
          </div>
          <FormGroup label="LDAP Username (Optional):">
            <input
              type="text"
              name="default_ldap_username"
              value={formData.default_ldap_username}
              onChange={onChange}
              placeholder="Leave empty for runtime prompt"
            />
          </FormGroup>
          <FormGroup label="LDAP Password (Optional):">
            <input
              type="password"
              name="default_ldap_password"
              value={formData.default_ldap_password}
              onChange={onChange}
              placeholder="Leave empty for security"
            />
          </FormGroup>
        </div>
      )}

      {showTokenFields && (
        <div className="conditional-field visible">
          <div className="security-warning">
            ⚠️ <strong>Security Note:</strong> Embedding a token in the binary
            is not recommended for production. Leave empty and users will be
            prompted at runtime or can use the VAULT_TOKEN environment variable.
          </div>
          <FormGroup label="Vault Token (Optional):">
            <input
              type="password"
              name="default_vault_token"
              value={formData.default_vault_token}
              onChange={onChange}
              placeholder="Leave empty for security"
            />
          </FormGroup>
        </div>
      )}

      <div className="form-row">
        <FormGroup
          label="Default Auth Role:"
          tooltip="The Vault auth role to request. Only needed if your auth method requires a specific role name."
        >
          <input
            type="text"
            name="default_auth_role"
            value={formData.default_auth_role}
            onChange={onChange}
            placeholder="Optional"
          />
        </FormGroup>

        <FormGroup
          label="Default Auth Mount:"
          tooltip="The Vault mount path for the auth method (e.g. 'auth/ldap'). Defaults to the method name if not set."
        >
          <input
            type="text"
            name="default_auth_mount"
            value={formData.default_auth_mount}
            onChange={onChange}
            placeholder="Defaults to auth method name"
          />
        </FormGroup>
      </div>
    </CollapsibleSection>
  );
}
