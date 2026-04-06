package filesystem

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/open-policy-agent/opa/rego"
)

// PolicyEvaluator evaluates access control policies using Open Policy Agent (OPA) and REGO.
//
// It enables fine-grained access control based on:
//   - Process information (PID, process name, executable path)
//   - User information (UID, GID, username)
//   - Path being accessed
//   - Operation being performed (GETATTR, READ, READDIR, etc.)
//   - Time of access
//   - Custom metadata
//
// Policies are written in REGO and can be loaded from a single file or a directory
// of .rego files for modular policy management. Policies can also be loaded from
// HTTP/HTTPS URLs and will be cached in memory.
type PolicyEvaluator struct {
	policyPath   string
	isDirectory  bool
	isURL        bool
	cachedPolicy string // for URL-based policies
	query        *rego.PreparedEvalQuery
	mu           sync.RWMutex
}

// AccessRequest contains comprehensive information about a filesystem access request.
//
// This structure is passed to the REGO policy engine as input, allowing policies
// to make decisions based on all available context about the access attempt.
//
// Fields are automatically populated with process and user information when available.
type AccessRequest struct {
	Path        string            `json:"path"`
	Operation   string            `json:"operation"`
	PID         int               `json:"pid"`
	UID         uint32            `json:"uid"`
	GID         uint32            `json:"gid"`
	ProcessName string            `json:"process_name"`
	ProcessPath string            `json:"process_path"`
	Username    string            `json:"username"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewPolicyEvaluator creates a new policy evaluator from a REGO policy file, directory, or URL.
//
// If policyPath is a directory, all .rego files in the directory are loaded and combined.
// This allows for modular policy management where different aspects of access control
// can be defined in separate files.
//
// If policyPath is an HTTP/HTTPS URL, the policy will be downloaded and cached in memory.
//
// The policy must define a rule at data.vault.result that returns an object with
// "allow" (bool) and "reason" (string) fields.
//
// Example policy:
//
//	package vault
//
//	default result = {"allow": false, "reason": ""}
//
//	result = {"allow": true, "reason": "Allowed read"} {
//	    input.operation == "READ"
//	    startswith(input.path, "secret/")
//	}
//
// Returns an error if the policy path doesn't exist or if the policy cannot be compiled.
func NewPolicyEvaluator(policyPath string) (*PolicyEvaluator, error) {
	// Check if policyPath is a URL
	if isURL(policyPath) {
		// Download and cache the policy
		policyContent, err := downloadPolicy(policyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to download policy from URL: %w", err)
		}

		// Compile the downloaded policy
		query, err := loadPolicyFromString(policyPath, policyContent)
		if err != nil {
			return nil, err
		}

		return &PolicyEvaluator{
			policyPath:   policyPath,
			isURL:        true,
			cachedPolicy: policyContent,
			query:        &query,
		}, nil
	}

	// Check if path is a file or directory
	info, err := os.Stat(policyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat policy path: %w", err)
	}

	isDir := info.IsDir()
	var query rego.PreparedEvalQuery

	if isDir {
		// Load all .rego files from directory
		query, err = loadPoliciesFromDirectory(policyPath)
		if err != nil {
			return nil, err
		}
	} else {
		// Load single policy file
		query, err = loadPolicyFile(policyPath)
		if err != nil {
			return nil, err
		}
	}

	return &PolicyEvaluator{
		policyPath:  policyPath,
		isDirectory: isDir,
		query:       &query,
	}, nil
}

// Evaluate evaluates the policy against an access request
func (pe *PolicyEvaluator) Evaluate(ctx context.Context, request *AccessRequest) (bool, string, error) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	if pe.query == nil {
		return false, "policy not loaded", fmt.Errorf("policy evaluator not initialized")
	}

	// Prepare input for the policy evaluation
	input := map[string]interface{}{
		"path":         request.Path,
		"operation":    request.Operation,
		"pid":          request.PID,
		"uid":          request.UID,
		"gid":          request.GID,
		"process_name": request.ProcessName,
		"process_path": request.ProcessPath,
		"username":     request.Username,
		"metadata":     request.Metadata,
	}

	// Evaluate the policy
	results, err := pe.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, "", fmt.Errorf("policy evaluation failed: %w", err)
	}

	// Check if the policy returned results
	if len(results) == 0 {
		return false, "policy returned no results", nil
	}

	// Extract the result object
	resultValue := results[0].Expressions[0].Value
	resultMap, ok := resultValue.(map[string]interface{})
	if !ok {
		return false, "policy did not return a result object", nil
	}

	// Extract allow field
	allowed := false
	if allowVal, ok := resultMap["allow"].(bool); ok {
		allowed = allowVal
	}

	// Extract reason field
	reason := "policy decision"
	if reasonVal, ok := resultMap["reason"].(string); ok && reasonVal != "" {
		reason = reasonVal
	}

	return allowed, reason, nil
}

// Reload reloads the policy from disk or URL
func (pe *PolicyEvaluator) Reload() error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	var query rego.PreparedEvalQuery
	var err error

	if pe.isURL {
		// Re-download policy from URL
		policyContent, err := downloadPolicy(pe.policyPath)
		if err != nil {
			return fmt.Errorf("failed to re-download policy: %w", err)
		}
		pe.cachedPolicy = policyContent

		query, err = loadPolicyFromString(pe.policyPath, policyContent)
		if err != nil {
			return err
		}
	} else if pe.isDirectory {
		// Reload all policies from directory
		query, err = loadPoliciesFromDirectory(pe.policyPath)
		if err != nil {
			return err
		}
	} else {
		// Reload single policy file
		query, err = loadPolicyFile(pe.policyPath)
		if err != nil {
			return err
		}
	}

	pe.query = &query
	return nil
}

// ValidatePolicy validates a REGO policy file, directory, or URL without creating an evaluator
func ValidatePolicy(policyPath string) error {
	// Check if it's a URL
	if isURL(policyPath) {
		policyContent, err := downloadPolicy(policyPath)
		if err != nil {
			return fmt.Errorf("failed to download policy from URL: %w", err)
		}
		_, err = loadPolicyFromString(policyPath, policyContent)
		return err
	}

	info, err := os.Stat(policyPath)
	if err != nil {
		return fmt.Errorf("failed to stat policy path: %w", err)
	}

	if info.IsDir() {
		// Validate all policies in directory
		_, err = loadPoliciesFromDirectory(policyPath)
		if err != nil {
			return err
		}
	} else {
		// Validate single policy file
		_, err = loadPolicyFile(policyPath)
		if err != nil {
			return err
		}
	}

	return nil
}

// loadPolicyFile loads and compiles a single REGO policy file
func loadPolicyFile(policyPath string) (rego.PreparedEvalQuery, error) {
	policyContent, err := os.ReadFile(policyPath)
	if err != nil {
		return rego.PreparedEvalQuery{}, fmt.Errorf("failed to read policy file: %w", err)
	}

	// Create query for result object
	query, err := rego.New(
		rego.Query("data.vault.result"),
		rego.Module(policyPath, string(policyContent)),
	).PrepareForEval(context.Background())
	if err != nil {
		return rego.PreparedEvalQuery{}, fmt.Errorf("failed to compile policy: %w", err)
	}

	return query, nil
}

// loadPoliciesFromDirectory loads all .rego files from a directory
func loadPoliciesFromDirectory(dirPath string) (rego.PreparedEvalQuery, error) {
	// Find all .rego files in the directory
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return rego.PreparedEvalQuery{}, fmt.Errorf("failed to read policy directory: %w", err)
	}

	// Collect all modules
	var modules []func(*rego.Rego)
	policyCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only load .rego files
		if !strings.HasSuffix(entry.Name(), ".rego") {
			continue
		}

		fullPath := filepath.Join(dirPath, entry.Name())
		policyContent, err := os.ReadFile(fullPath)
		if err != nil {
			return rego.PreparedEvalQuery{}, fmt.Errorf("failed to read policy file %s: %w", entry.Name(), err)
		}

		modules = append(modules, rego.Module(fullPath, string(policyContent)))
		policyCount++
	}

	if policyCount == 0 {
		return rego.PreparedEvalQuery{}, fmt.Errorf("no .rego policy files found in directory: %s", dirPath)
	}

	// Create query for result object with all modules
	opts := append([]func(*rego.Rego){rego.Query("data.vault.result")}, modules...)
	query, err := rego.New(opts...).PrepareForEval(context.Background())
	if err != nil {
		return rego.PreparedEvalQuery{}, fmt.Errorf("failed to compile policies: %w", err)
	}

	return query, nil
}

// BuildAccessRequest builds an AccessRequest from ProcessInfo and operation details
func BuildAccessRequest(procInfo *ProcessInfo, path, operation string) *AccessRequest {
	if procInfo == nil {
		return &AccessRequest{
			Path:      path,
			Operation: operation,
		}
	}

	details := resolveProcessDetails(procInfo)

	return &AccessRequest{
		Path:        cleanPath(path),
		Operation:   strings.ToUpper(operation),
		PID:         procInfo.PID,
		UID:         procInfo.UID,
		GID:         procInfo.GID,
		ProcessName: details.ProcessName,
		ProcessPath: details.ExecutablePath,
		Username:    details.Username,
	}
}

// cleanPath normalizes the path for policy evaluation
func cleanPath(path string) string {
	// Replace backslashes with forward slashes
	path = strings.ReplaceAll(path, "\\", "/")
	// Replace multiple slashes with single slash
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	// Remove leading/trailing slashes
	path = strings.Trim(path, "/")
	return path
}

// isURL checks if a path is an HTTP or HTTPS URL
func isURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// downloadPolicy downloads a policy from a URL and returns its content
func downloadPolicy(url string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read policy content: %w", err)
	}

	return string(body), nil
}

// loadPolicyFromString loads and compiles a REGO policy from a string
func loadPolicyFromString(name, policyContent string) (rego.PreparedEvalQuery, error) {
	// Create query for result object
	query, err := rego.New(
		rego.Query("data.vault.result"),
		rego.Module(name, policyContent),
	).PrepareForEval(context.Background())
	if err != nil {
		return rego.PreparedEvalQuery{}, fmt.Errorf("failed to compile policy: %w", err)
	}

	return query, nil
}
