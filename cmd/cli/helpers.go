package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"safeguard/pkg/logger"
	"safeguard/pkg/vault"
	"safeguard/pkg/vault/adapter"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

func setupLogging(f *appFlags) *logger.Logger {
	var multi io.Writer

	if *f.logFile != "" {
		// Ensure parent directory exists
		logDir := filepath.Dir(*f.logFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not create log directory %s: %v\n", logDir, err)
		}

		rotator := &lumberjack.Logger{
			Filename:   *f.logFile,
			MaxSize:    *f.logMaxSize,
			MaxBackups: *f.logMaxBackups,
			MaxAge:     *f.logMaxAge,
			Compress:   *f.logCompress,
		}
		multi = zerolog.MultiLevelWriter(os.Stdout, rotator)
	} else {
		multi = os.Stdout
	}

	log := logger.New(multi, *f.debug)
	logger.SetDefault(log)

	if *f.logFile != "" {
		log.Info("File logging enabled", map[string]interface{}{
			"log_file":     *f.logFile,
			"max_size_mb":  *f.logMaxSize,
			"max_backups":  *f.logMaxBackups,
			"max_age_days": *f.logMaxAge,
			"compress":     *f.logCompress,
		})
	}

	return log
}

func connectVault(log *logger.Logger, f *appFlags, token string) vault.ClientInterface {
	cfg := adapter.Config{
		Provider: *f.vaultProvider,
		Address:  *f.vaultAddr,
		Token:    token,
		Debug:    *f.debug,
		Logger:   log,
	}
	vaultClient, err := adapter.New(cfg)
	if err != nil {
		log.Fatal("Failed to create Vault client", map[string]interface{}{
			"provider": *f.vaultProvider,
			"error":    err.Error(),
		})
	}

	if err := vaultClient.Ping(context.Background()); err != nil {
		log.Fatal("Failed to connect to Vault", map[string]interface{}{
			"vault_addr": *f.vaultAddr,
			"provider":   *f.vaultProvider,
			"error":      err.Error(),
		})
	}
	log.Info("Successfully connected to Vault", map[string]interface{}{
		"vault_addr": *f.vaultAddr,
		"provider":   *f.vaultProvider,
	})

	if *f.cacheEnabled {
		ttl := time.Duration(*f.cacheTTL) * time.Second
		vaultClient = vault.NewCachingClient(vaultClient, ttl)
		log.Info("Response cache enabled", map[string]interface{}{
			"cache_ttl_seconds": *f.cacheTTL,
		})
	}

	return vaultClient
}
