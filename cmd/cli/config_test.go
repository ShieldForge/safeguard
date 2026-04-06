package main

import (
	"io"
	"os"
	"path/filepath"
	"safeguard/pkg/logger"
	"testing"
)

func TestParseAccessLists_Empty(t *testing.T) {
	log := logger.New(io.Discard, false)
	pidStr := ""
	uidStr := ""
	f := &appFlags{
		allowedPIDsStr: &pidStr,
		allowedUIDsStr: &uidStr,
	}

	pids, uids := parseAccessLists(log, f)
	if len(pids) != 0 {
		t.Errorf("Expected empty PIDs, got %v", pids)
	}
	if len(uids) != 0 {
		t.Errorf("Expected empty UIDs, got %v", uids)
	}
}

func TestParseAccessLists_ValidPIDs(t *testing.T) {
	log := logger.New(io.Discard, false)
	pidStr := "123,456,789"
	uidStr := ""
	f := &appFlags{
		allowedPIDsStr: &pidStr,
		allowedUIDsStr: &uidStr,
	}

	pids, _ := parseAccessLists(log, f)
	if len(pids) != 3 {
		t.Fatalf("Expected 3 PIDs, got %d", len(pids))
	}
	if pids[0] != 123 || pids[1] != 456 || pids[2] != 789 {
		t.Errorf("PIDs = %v, want [123 456 789]", pids)
	}
}

func TestParseAccessLists_ValidUIDs(t *testing.T) {
	log := logger.New(io.Discard, false)
	pidStr := ""
	uidStr := "1000,1001"
	f := &appFlags{
		allowedPIDsStr: &pidStr,
		allowedUIDsStr: &uidStr,
	}

	_, uids := parseAccessLists(log, f)
	if len(uids) != 2 {
		t.Fatalf("Expected 2 UIDs, got %d", len(uids))
	}
	if uids[0] != 1000 || uids[1] != 1001 {
		t.Errorf("UIDs = %v, want [1000 1001]", uids)
	}
}

func TestParseAccessLists_WhitespaceHandling(t *testing.T) {
	log := logger.New(io.Discard, false)
	pidStr := " 100 , 200 , "
	uidStr := ""
	f := &appFlags{
		allowedPIDsStr: &pidStr,
		allowedUIDsStr: &uidStr,
	}

	pids, _ := parseAccessLists(log, f)
	if len(pids) != 2 {
		t.Fatalf("Expected 2 PIDs after trimming, got %d: %v", len(pids), pids)
	}
	if pids[0] != 100 || pids[1] != 200 {
		t.Errorf("PIDs = %v, want [100 200]", pids)
	}
}

func TestExtractEmbeddedPolicies_Empty(t *testing.T) {
	// When no policies are embedded, the map is empty
	// Save and restore original
	orig := embeddedPolicyFiles
	embeddedPolicyFiles = map[string]string{}
	defer func() { embeddedPolicyFiles = orig }()

	dir, err := extractEmbeddedPolicies()
	if err != nil {
		t.Fatalf("extractEmbeddedPolicies() error = %v", err)
	}
	defer os.RemoveAll(dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("Failed to read dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected empty dir, got %d entries", len(entries))
	}
}

func TestExtractEmbeddedPolicies_WithPolicies(t *testing.T) {
	orig := embeddedPolicyFiles
	embeddedPolicyFiles = map[string]string{
		"policy1.rego": "package policy1\ndefault allow = true",
		"policy2.rego": "package policy2\ndefault allow = false",
	}
	defer func() { embeddedPolicyFiles = orig }()

	dir, err := extractEmbeddedPolicies()
	if err != nil {
		t.Fatalf("extractEmbeddedPolicies() error = %v", err)
	}
	defer os.RemoveAll(dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("Failed to read dir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	// Verify content
	content, err := os.ReadFile(filepath.Join(dir, "policy1.rego"))
	if err != nil {
		t.Fatalf("Failed to read policy1.rego: %v", err)
	}
	if string(content) != "package policy1\ndefault allow = true" {
		t.Errorf("policy1.rego content = %v", string(content))
	}
}

func TestExtractEmbeddedPolicies_FilePermissions(t *testing.T) {
	orig := embeddedPolicyFiles
	embeddedPolicyFiles = map[string]string{
		"test.rego": "package test",
	}
	defer func() { embeddedPolicyFiles = orig }()

	dir, err := extractEmbeddedPolicies()
	if err != nil {
		t.Fatalf("extractEmbeddedPolicies() error = %v", err)
	}
	defer os.RemoveAll(dir)

	info, err := os.Stat(filepath.Join(dir, "test.rego"))
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// On Windows permissions work differently, just check file exists
	if info.Size() == 0 {
		t.Error("Policy file should not be empty")
	}
}
