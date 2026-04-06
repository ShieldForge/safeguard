package main

import (
	"flag"
	"safeguard/pkg/utils"
	"safeguard/pkg/vault/adapter"
	"strings"
)

// appFlags holds all parsed CLI flags.
type appFlags struct {
	mountPoint     *string
	vaultAddr      *string
	vaultProvider  *string
	authMethod     *string
	authRole       *string
	authMount      *string
	ldapUsername   *string
	ldapPassword   *string
	vaultToken     *string
	debug          *bool
	monitor        *bool
	auditLog       *string
	accessControl  *bool
	policyPath     *string
	allowedPIDsStr *string
	allowedUIDsStr *string
	mappingConfig  *string
	logFile        *string
	logMaxSize     *int
	logMaxBackups  *int
	logMaxAge      *int
	logCompress    *bool
	cacheEnabled   *bool
	cacheTTL       *int
}

func parseFlags() *appFlags {
	mountDefault := defaultMountPoint
	if mountDefault == "" {
		mountDefault = getDefaultMountPoint()
	}

	debugDefault := utils.ParseBoolDefault(defaultDebug, false)
	monitorDefault := utils.ParseBoolDefault(defaultMonitor, false)
	accessControlDefault := utils.ParseBoolDefault(defaultAccessControl, false)

	f := &appFlags{
		mountPoint:     flag.String("mount", mountDefault, "Mount point for the virtual drive (e.g., V: on Windows, /mnt/vault on Linux)"),
		vaultAddr:      flag.String("vault-addr", defaultVaultAddr, "Vault service address"),
		vaultProvider:  flag.String("vault-provider", defaultVaultProvider, "Vault backend provider ("+strings.Join(adapter.ListProviders(), ", ")+")"),
		authMethod:     flag.String("auth-method", defaultAuthMethod, "Authentication method (oidc, ldap, token, aws, approle)"),
		authRole:       flag.String("auth-role", defaultAuthRole, "Role name for OIDC or AppRole auth"),
		authMount:      flag.String("auth-mount", defaultAuthMount, "Custom auth mount path (defaults to auth method name)"),
		ldapUsername:   flag.String("ldap-username", "", "LDAP username (for ldap auth). If not provided, will prompt for input"),
		ldapPassword:   flag.String("ldap-password", "", "LDAP password (for ldap auth). If not provided, will prompt for input"),
		vaultToken:     flag.String("vault-token", "", "Vault token (only for token auth method). If not provided, will check VAULT_TOKEN env var or prompt for input"),
		debug:          flag.Bool("debug", debugDefault, "Enable debug logging"),
		monitor:        flag.Bool("monitor", monitorDefault, "Enable process monitoring (logs PID/UID for all operations)"),
		auditLog:       flag.String("audit-log", defaultAuditLog, "Path to audit log file (e.g., vault-audit.log)"),
		accessControl:  flag.Bool("access-control", accessControlDefault, "Enable process-based access control"),
		policyPath:     flag.String("policy-path", defaultPolicyPath, "Path to REGO policy file or directory for fine-grained access control"),
		allowedPIDsStr: flag.String("allowed-pids", defaultAllowedPIDs, "Comma-separated list of allowed process IDs (legacy, use policy-path for advanced rules)"),
		allowedUIDsStr: flag.String("allowed-uids", defaultAllowedUIDs, "Comma-separated list of allowed user IDs (legacy, use policy-path for advanced rules)"),
		mappingConfig:  flag.String("mapping-config", defaultMappingPath, "Path to JSON configuration file for mapping virtual paths to real files"),
		logFile:        flag.String("log-file", defaultLogFile, "Path to application log file (empty to disable file logging)"),
		logMaxSize:     flag.Int("log-max-size", utils.ParseIntDefault(defaultLogMaxSize, 100), "Maximum size of a log file in megabytes before rotation"),
		logMaxBackups:  flag.Int("log-max-backups", utils.ParseIntDefault(defaultLogMaxBackups, 5), "Maximum number of rotated log files to retain"),
		logMaxAge:      flag.Int("log-max-age", utils.ParseIntDefault(defaultLogMaxAge, 30), "Maximum number of days to retain rotated log files"),
		logCompress:    flag.Bool("log-compress", utils.ParseBoolDefault(defaultLogCompress, true), "Compress rotated log files with gzip"),
		cacheEnabled:   flag.Bool("cache", utils.ParseBoolDefault(defaultCacheEnabled, false), "Enable in-memory response cache for vault operations"),
		cacheTTL:       flag.Int("cache-ttl", utils.ParseIntDefault(defaultCacheTTL, 300), "Cache time-to-live in seconds"),
	}
	flag.Parse()
	return f
}
