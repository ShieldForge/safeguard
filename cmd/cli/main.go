// Package main provides the safeguard virtual drive application.
//
// safeguard mounts HashiCorp Vault secrets as a FUSE filesystem, presenting
// secrets and mapped files/directories as a virtual drive. The application
// supports multiple authentication methods (OIDC, LDAP, token, AWS, AppRole),
// policy-based access control using OPA/REGO, and process monitoring for
// audit compliance.
//
// Usage:
//
//	safeguard -mount V: -vault-addr http://127.0.0.1:8200 -auth-method oidc
//
// Command-line flags:
//   - mount: Mount point for the virtual drive (V: on Windows, /mnt/vault on Linux)
//   - vault-addr: HashiCorp Vault address (default: http://127.0.0.1:8200)
//   - auth-method: Authentication method (oidc, ldap, token, aws, approle)
//   - auth-role: Role name for OIDC or AppRole authentication
//   - auth-mount: Custom auth mount path (defaults to auth method name)
//   - ldap-username: LDAP username (for ldap auth)
//   - ldap-password: LDAP password (for ldap auth)
//   - vault-token: Vault token (only for token auth method)
//   - debug: Enable debug logging
//   - monitor: Enable process monitoring (logs PID/UID for all operations)
//   - audit-log: Path to audit log file (e.g., vault-audit.log)
//   - access-control: Enable process-based access control
//   - policy-path: Path to REGO policy file or directory for fine-grained access control
//   - allowed-pids: Comma-separated list of allowed process IDs (legacy, use policy-path)
//   - allowed-uids: Comma-separated list of allowed user IDs (legacy, use policy-path)
//   - mapping-config: Path to JSON configuration file for mapping virtual paths to real files
//
// The application will:
//  1. Authenticate with Vault using the specified method
//  2. Test the Vault connection
//  3. Load and validate policy files (if configured)
//  4. Mount the virtual drive at the specified mount point
//  5. Wait for interrupt signal (Ctrl+C) to unmount
//
// Example:
//
//	# Windows with OIDC authentication
//	safeguard -mount V: -auth-method oidc -debug
//
//	# Linux with LDAP authentication and policy-based access control
//	safeguard -mount /mnt/vault -auth-method ldap -ldap-username john -policy-path ./policies
//
//	# With path mapping and audit logging
//	safeguard -mount V: -mapping-config mapping.json -audit-log audit.log
package main

import (
	"os"
	"os/signal"
	"safeguard/pkg/utils"
	"syscall"
)

// Build-time variables that can be overridden using -ldflags
var (
	defaultVaultAddr     = "http://127.0.0.1:8200"
	defaultAuthMethod    = "oidc"
	defaultMountPoint    = "V:" // Default to V: for Windows, overridden in getDefaultMountPoint() for other OSes
	defaultAuthRole      = ""
	defaultAuthMount     = ""
	defaultPolicyPath    = ""
	defaultMappingPath   = ""
	defaultAuditLog      = ""
	defaultAllowedPIDs   = ""
	defaultAllowedUIDs   = ""
	defaultLdapUsername  = ""
	defaultLdapPassword  = ""
	defaultVaultToken    = ""
	defaultVaultProvider = "hashicorp"
	defaultDebug         = ""
	defaultMonitor       = ""
	defaultAccessControl = ""
	defaultLogFile       = "./logs/safeguard.log"
	defaultLogMaxSize    = "100" // megabytes
	defaultLogMaxBackups = "5"
	defaultLogMaxAge     = "30" // days
	defaultLogCompress   = "true"
	defaultCacheEnabled  = "false"
	defaultCacheTTL      = "60"
	disableCliFlags      = ""
	version              = "dev"
	buildTag             = ""
)

// main is the entry point for the safeguard application.
//
// It performs the following steps:
//  1. Parses command-line flags
//  2. Initializes logging infrastructure
//  3. Authenticates with Vault using the configured method
//  4. Creates and tests the Vault client connection
//  5. Parses and validates access control lists (PIDs, UIDs)
//  6. Validates policy files (if configured)
//  7. Creates the VaultFS filesystem with all options
//  8. Mounts the filesystem at the specified mount point
//  9. Waits for interrupt signal to unmount and exit
//
// The function exits (via log.Fatal) on any critical error during initialization
// or mounting. Normal termination occurs when an interrupt signal (Ctrl+C or SIGTERM)
// is received.
func main() {
	// If build-time config disables CLI flags, ignore all arguments.
	if utils.ParseBoolDefault(disableCliFlags, false) {
		os.Args = os.Args[:1]
	}

	flags := parseFlags()

	promptCredentials(flags)

	log := setupLogging(flags)

	authenticator, token := authenticate(log, flags)
	defer authenticator.StopRenewal()

	vaultClient := connectVault(log, flags, token)

	// Wire up token renewal: when the authenticator renews, update the vault client
	authenticator.SetOnTokenRenewed(func(newToken string) {
		vaultClient.SetToken(newToken)
		log.Debug("Vault client token updated after renewal", nil)
	})

	allowedPIDs, allowedUIDs := parseAccessLists(log, flags)

	// If policies were embedded at build time and no policy path was specified, extract them
	if len(embeddedPolicyFiles) > 0 && *flags.policyPath == "" {
		tempDir, err := extractEmbeddedPolicies()
		if err != nil {
			log.Fatal("Failed to extract embedded policies", map[string]interface{}{
				"error": err.Error(),
			})
		}
		defer os.RemoveAll(tempDir)
		*flags.policyPath = tempDir
		log.Info("Using embedded policies", map[string]interface{}{
			"count":    len(embeddedPolicyFiles),
			"temp_dir": tempDir,
		})
	}

	validatePolicies(log, flags)

	fs := buildFilesystem(log, flags, vaultClient, allowedPIDs, allowedUIDs)
	host := mountFilesystem(log, fs, *flags.mountPoint)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Info("Unmounting filesystem", nil)
	if !host.Unmount() {
		log.Fatal("Failed to unmount", nil)
	}

	if err := fs.Close(); err != nil {
		log.Warn("Error closing filesystem resources", map[string]interface{}{
			"error": err.Error(),
		})
	}

	log.Info("Unmounted successfully", nil)
}
