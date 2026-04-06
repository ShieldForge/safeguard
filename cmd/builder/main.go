// Package main provides an HTTP server for building custom safeguard binaries
package main

//go:generate bash generate_ui.sh

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"safeguard/pkg/builder"
)

//go:embed static/*
var staticFiles embed.FS

var (
	port      string
	sourceDir string
	workDir   string
	outputDir string
)

func init() {
	flag.StringVar(&port, "port", "8082", "HTTP server port")
	flag.StringVar(&sourceDir, "source", "", "Source directory of safeguard project (auto-detected in debug mode)")
	flag.StringVar(&workDir, "work", "", "Work directory for builds (defaults to <source>/build-work)")
	flag.StringVar(&outputDir, "output", "", "Output directory for built binaries (defaults to <source>/build-output)")
}

func main() {
	flag.Parse()

	// Resolve source directory
	if sourceDir == "" {
		detected := findProjectRoot()
		if detected != "" {
			sourceDir = detected
			if isDebugMode() {
				log.Printf("[debug] Auto-detected source directory: %s", sourceDir)
			}
		} else {
			sourceDir = "."
			if isDebugMode() {
				log.Printf("[debug] Could not auto-detect project root, falling back to CWD")
			}
		}
	}

	// Resolve all directories to absolute paths
	absSourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path for source directory: %v", err)
	}
	sourceDir = absSourceDir

	// Default work/output dirs relative to source
	if workDir == "" {
		workDir = filepath.Join(sourceDir, "build-work")
	}
	workDir, _ = filepath.Abs(workDir)
	if outputDir == "" {
		outputDir = filepath.Join(sourceDir, "build-output")
	}
	outputDir, _ = filepath.Abs(outputDir)

	// Validate source directory
	goModPath := filepath.Join(sourceDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		log.Fatalf("Source directory %s does not contain go.mod. Use -source to specify the project root.", sourceDir)
	}

	log.Printf("Starting safeguard build server")
	log.Printf("Source directory: %s", sourceDir)
	log.Printf("Work directory: %s", workDir)
	log.Printf("Output directory: %s", outputDir)
	log.Printf("Server port: %s", port)

	// Create builder instance
	b, err := builder.NewBuilder(sourceDir, workDir, outputDir)
	if err != nil {
		log.Fatalf("Failed to create builder: %v", err)
	}

	// Serve embedded static files
	staticSub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem: %v", err)
	}
	staticHandler := http.FileServer(http.FS(staticSub))

	// Setup HTTP routes
	http.HandleFunc("/", handleIndex(staticSub))
	http.Handle("/static/", http.StripPrefix("/static/", staticHandler))
	http.HandleFunc("/api/build", handleBuild(b))
	http.HandleFunc("/api/build-stream", handleBuildStream(b))
	http.HandleFunc("/api/validate", handleValidate)
	http.HandleFunc("/api/download/", handleDownload)
	http.HandleFunc("/api/health", handleHealth)

	// Start server
	addr := ":" + port
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// handleIndex serves the React-based HTML interface from embedded files
func handleIndex(staticSub fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		data, err := fs.ReadFile(staticSub, "index.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	}
}

// handleBuild handles the build request
func handleBuild(defaultBuilder *builder.Builder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse request body
		var config builder.BuildConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("Invalid request body: %v", err),
			})
			return
		}

		log.Printf("Build request received: OS=%s, Arch=%s, Version=%s, Tag=%s",
			config.TargetOS, config.TargetArch, config.Version, config.BuildTag)

		// Use directory overrides if provided, otherwise use server defaults
		b := defaultBuilder
		if config.SourceDir != "" || config.WorkDir != "" || config.OutputDir != "" {
			src := sourceDir
			wrk := workDir
			out := outputDir
			if config.SourceDir != "" {
				src = config.SourceDir
			}
			if config.WorkDir != "" {
				wrk = config.WorkDir
			}
			if config.OutputDir != "" {
				out = config.OutputDir
			}
			var err error
			b, err = builder.NewBuilder(src, wrk, out)
			if err != nil {
				respondJSON(w, http.StatusInternalServerError, map[string]string{
					"error": fmt.Sprintf("Failed to configure build directories: %v", err),
				})
				return
			}
		}

		// Perform build
		result, err := b.Build(config)
		if err != nil {
			log.Printf("Build failed: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Build failed: %v", err),
			})
			return
		}

		log.Printf("Build successful: %s (size: %d bytes, checksum: %s)",
			result.BinaryPath, result.Size, result.Checksum)

		// Return build result
		respondJSON(w, http.StatusOK, result)
	}
}

// sseWriter wraps an http.ResponseWriter and flushes after every Write, sending
// each line as an SSE "data:" event.
type sseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func (s *sseWriter) Write(p []byte) (int, error) {
	lines := strings.Split(strings.TrimRight(string(p), "\n"), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		fmt.Fprintf(s.w, "data: %s\n\n", line)
	}
	s.flusher.Flush()
	return len(p), nil
}

// handleBuildStream handles build requests using Server-Sent Events to stream progress.
func handleBuildStream(defaultBuilder *builder.Builder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		// Parse request body
		var config builder.BuildConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		// SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher.Flush()

		logWriter := &sseWriter{w: w, flusher: flusher}

		log.Printf("Build-stream request received: OS=%s, Arch=%s, Version=%s, Tag=%s",
			config.TargetOS, config.TargetArch, config.Version, config.BuildTag)

		// Use directory overrides if provided
		b := defaultBuilder
		if config.SourceDir != "" || config.WorkDir != "" || config.OutputDir != "" {
			src := sourceDir
			wrk := workDir
			out := outputDir
			if config.SourceDir != "" {
				src = config.SourceDir
			}
			if config.WorkDir != "" {
				wrk = config.WorkDir
			}
			if config.OutputDir != "" {
				out = config.OutputDir
			}
			var err error
			b, err = builder.NewBuilder(src, wrk, out)
			if err != nil {
				fmt.Fprintf(logWriter, "Failed to configure build directories: %v", err)
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", fmt.Sprintf(`{"error":"Failed to configure build directories: %v"}`, err))
				flusher.Flush()
				return
			}
		}

		// Perform build with streaming log
		result, err := b.BuildWithLog(config, logWriter)
		if err != nil {
			log.Printf("Build failed: %v", err)
			errJSON, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("Build failed: %v", err)})
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", errJSON)
			flusher.Flush()
			return
		}

		log.Printf("Build-stream successful: %s (size: %d bytes, checksum: %s)",
			result.BinaryPath, result.Size, result.Checksum)

		// Send the final result as a "done" event
		resultJSON, _ := json.Marshal(result)
		fmt.Fprintf(w, "event: done\ndata: %s\n\n", resultJSON)
		flusher.Flush()
	}
}

// handleDownload serves the built binary for download
func handleDownload(w http.ResponseWriter, r *http.Request) {
	// Extract filename from URL path
	filename := filepath.Base(r.URL.Path)
	if filename == "." || filename == "/" {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Construct full path
	filePath := filepath.Join(outputDir, filename)

	// Security check: ensure the file is within outputDir
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}
	absOutputDir, _ := filepath.Abs(outputDir)
	if !filepath.HasPrefix(absFilePath, absOutputDir) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if file exists
	if _, err := os.Stat(absFilePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Open file
	file, err := os.Open(absFilePath)
	if err != nil {
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Failed to stat file", http.StatusInternalServerError)
		return
	}

	// Set headers
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))

	// Stream file to response
	io.Copy(w, file)
	log.Printf("Downloaded: %s", filename)
}

// handleHealth returns health status
func handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
	})
}

// handleValidate checks if the build environment is properly configured.
// GET returns validation using server defaults. POST accepts directory overrides.
func handleValidate(w http.ResponseWriter, r *http.Request) {
	src := sourceDir
	wrk := workDir
	out := outputDir

	if r.Method == http.MethodPost {
		var req struct {
			SourceDir string `json:"source_dir"`
			WorkDir   string `json:"work_dir"`
			OutputDir string `json:"output_dir"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if req.SourceDir != "" {
				src = req.SourceDir
			}
			if req.WorkDir != "" {
				wrk = req.WorkDir
			}
			if req.OutputDir != "" {
				out = req.OutputDir
			}
		}
		// Resolve to absolute paths
		src, _ = filepath.Abs(src)
		wrk, _ = filepath.Abs(wrk)
		out, _ = filepath.Abs(out)
	}

	issues := []string{}
	warnings := []string{}

	// Check for Go compiler
	if _, err := exec.LookPath("go"); err != nil {
		issues = append(issues, "Go compiler not found in PATH")
	} else {
		// Get Go version
		cmd := exec.Command("go", "version")
		output, _ := cmd.Output()
		warnings = append(warnings, "Go: "+string(output))
	}

	// Check for GCC (required for CGO)
	if _, err := exec.LookPath("gcc"); err != nil {
		issues = append(issues, "GCC not found in PATH (required for CGO/cgofuse). Install MinGW-w64, TDM-GCC, or similar.")
	} else {
		warnings = append(warnings, "GCC: Available in PATH")
	}

	// Check source directory
	if _, err := os.Stat(filepath.Join(src, "go.mod")); os.IsNotExist(err) {
		issues = append(issues, "Source directory does not contain go.mod file")
	}

	// read go.mod to check for cgofuse dependency
	goModData, err := os.ReadFile(filepath.Join(src, "go.mod"))
	if err != nil {
		issues = append(issues, "Failed to read go.mod: "+err.Error())
	} else if !strings.Contains(string(goModData), "github.com/winfsp/cgofuse") {
		issues = append(issues, "go.mod does not include github.com/winfsp/cgofuse dependency")
	}

	// check goModData for go version requirement (go 1.18+ for CGO generics)
	if !strings.Contains(string(goModData), "go 1.18") && !strings.Contains(string(goModData), "go 1.19") && !strings.Contains(string(goModData), "go 1.2") {
		issues = append(issues, "go.mod should specify go 1.18 or higher for CGO generics support")
	}

	// check goModData for correct module name (should be main for builder)
	if !strings.Contains(string(goModData), "module safeguard") {
		issues = append(issues, "go.mod isn't using expected module name")
	}

	// Check output directory
	if _, err := os.Stat(out); os.IsNotExist(err) {
		warnings = append(warnings, "Output directory will be created on first build")
	}

	status := "ready"
	if len(issues) > 0 {
		status = "not_ready"
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": status,
		"issues": issues,
		"info":   warnings,
		"source": src,
		"work":   wrk,
		"output": out,
	})
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// findProjectRoot walks up from the current working directory looking for a
// directory that contains go.mod and the expected repo structure (cmd/builder/).
// Returns the absolute path to the project root, or "" if not found.
func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		goMod := filepath.Join(dir, "go.mod")
		cmdBuilder := filepath.Join(dir, "cmd", "builder")
		if _, err := os.Stat(goMod); err == nil {
			if info, err := os.Stat(cmdBuilder); err == nil && info.IsDir() {
				return dir
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// isDebugMode returns true if the process appears to be running in a debug context.
// It checks (in order): environment variables, build flags, and attached debugger.
func isDebugMode() bool {
	// 1. Explicit env var override
	for _, key := range []string{"BUILDER_DEBUG", "DEBUG"} {
		if v := os.Getenv(key); v == "1" || strings.EqualFold(v, "true") {
			log.Printf("[debug] Debug mode enabled via %s env var", key)
			return true
		}
	}

	// 2. Binary built with debug gcflags (-N -l), e.g. via Delve / VS Code debugger
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "-gcflags" && strings.Contains(setting.Value, "-N") && strings.Contains(setting.Value, "-l") {
				log.Printf("[debug] Debug mode detected from build flags: %s", setting.Value)
				return true
			}
		}
	}

	// 3. Debugger attached (Windows: kernel32 IsDebuggerPresent)
	if isDebuggerAttached() {
		log.Printf("[debug] Debug mode detected: debugger attached")
		return true
	}

	return false
}
