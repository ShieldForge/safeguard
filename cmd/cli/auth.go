package main

import (
	"bufio"
	"fmt"
	"os"
	"safeguard/pkg/auth"
	"safeguard/pkg/logger"
	"safeguard/pkg/vault/adapter"
	"strings"
	"syscall"

	"golang.org/x/term"
)

func promptCredentials(f *appFlags) {
	if *f.authMethod == "ldap" {
		if *f.ldapUsername == "" && defaultLdapUsername != "" {
			f.ldapUsername = &defaultLdapUsername
		}
		if *f.ldapUsername == "" {
			fmt.Print("LDAP Username: ")
			reader := bufio.NewReader(os.Stdin)
			username, err := reader.ReadString('\n')
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading username: %v\n", err)
				os.Exit(1)
			}
			username = strings.TrimSpace(username)
			f.ldapUsername = &username
		}
		if *f.ldapPassword == "" && defaultLdapPassword != "" {
			f.ldapPassword = &defaultLdapPassword
		}
		if *f.ldapPassword == "" {
			fmt.Print("LDAP Password: ")
			passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
				os.Exit(1)
			}
			password := string(passwordBytes)
			f.ldapPassword = &password
		}
	}

	if *f.authMethod == "token" && *f.vaultToken == "" {
		if defaultVaultToken != "" {
			f.vaultToken = &defaultVaultToken
		} else if token := os.Getenv("VAULT_TOKEN"); token != "" {
			f.vaultToken = &token
		} else {
			fmt.Print("Vault Token: ")
			tokenBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading vault token: %v\n", err)
				os.Exit(1)
			}
			token := string(tokenBytes)
			f.vaultToken = &token
		}
	}
}

func authenticate(log *logger.Logger, f *appFlags) (auth.AuthProvider, string) {
	log.Info("Authenticating with Vault", map[string]interface{}{
		"auth_method": *f.authMethod,
		"provider":    *f.vaultProvider,
	})

	cfg := adapter.Config{
		Provider: *f.vaultProvider,
		Address:  *f.vaultAddr,
		Token:    *f.vaultToken,
		Debug:    *f.debug,
		Logger:   log,
		Auth: adapter.AuthConfig{
			Method:    *f.authMethod,
			Username:  *f.ldapUsername,
			Password:  *f.ldapPassword,
			Role:      *f.authRole,
			MountPath: *f.authMount,
		},
	}

	authenticator, err := adapter.NewAuth(cfg)
	if err != nil {
		log.Fatal("Failed to create auth provider", map[string]interface{}{
			"provider": *f.vaultProvider,
			"error":    err.Error(),
		})
	}

	authResult, err := authenticator.Authenticate()
	if err != nil {
		log.Fatal("Authentication failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	if authResult.Renewable {
		log.Info("Token renewal enabled", map[string]interface{}{
			"lease_duration": authResult.LeaseDuration,
		})
		authenticator.StartRenewal()
	}

	return authenticator, authResult.Token
}
