// Package tools provides MCP tool handlers for reading and searching Ubuntu logs.
package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

const (
	defaultLines   = 100
	maxLines       = 10000
	logDir         = "/var/log"
	journalctlPath = "journalctl"
)

// ListLogFiles returns a list of readable log files under /var/log.
func ListLogFiles(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var files []string

	err := filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories/files we cannot access.
			return nil
		}
		if info.IsDir() {
			return nil
		}
		// Only include regular files that are readable.
		f, openErr := os.Open(path)
		if openErr != nil {
			return nil
		}
		f.Close()
		files = append(files, path)
		return nil
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to walk %s: %v", logDir, err)), nil
	}

	sort.Strings(files)

	if len(files) == 0 {
		return mcp.NewToolResultText("No readable log files found in " + logDir), nil
	}

	return mcp.NewToolResultText(strings.Join(files, "\n")), nil
}

// ReadLogFile reads the last N lines of a log file.
func ReadLogFile(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	lines := req.GetInt("lines", defaultLines)
	if lines <= 0 || lines > maxLines {
		lines = defaultLines
	}

	if err := validateLogPath(path); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	content, err := tailFile(path, lines)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read %s: %v", path, err)), nil
	}

	if content == "" {
		return mcp.NewToolResultText(fmt.Sprintf("File %s is empty or has no readable content.", path)), nil
	}

	return mcp.NewToolResultText(content), nil
}

// SearchLogFile searches a log file for lines matching a pattern.
func SearchLogFile(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	pattern, err := req.RequireString("pattern")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	maxResults := req.GetInt("max_results", defaultLines)
	if maxResults <= 0 || maxResults > maxLines {
		maxResults = defaultLines
	}

	if err := validateLogPath(path); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	results, err := grepFile(path, pattern, maxResults)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to search %s: %v", path, err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No matches found for %q in %s", pattern, path)), nil
	}

	return mcp.NewToolResultText(strings.Join(results, "\n")), nil
}

// ReadJournal reads systemd journal entries using journalctl.
func ReadJournal(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lines := req.GetInt("lines", defaultLines)
	if lines <= 0 || lines > maxLines {
		lines = defaultLines
	}

	args := []string{"--no-pager", "-n", fmt.Sprintf("%d", lines)}

	if unit := strings.TrimSpace(req.GetString("unit", "")); unit != "" {
		if err := validateIdentifier(unit); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		args = append(args, "-u", unit)
	}

	if since := strings.TrimSpace(req.GetString("since", "")); since != "" {
		if err := validateTimeString(since); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		args = append(args, "--since", since)
	}

	if boot := req.GetBool("current_boot", false); boot {
		args = append(args, "-b")
	}

	output, err := runJournalctl(args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("journalctl failed: %v", err)), nil
	}

	if strings.TrimSpace(output) == "" {
		return mcp.NewToolResultText("No journal entries found."), nil
	}

	return mcp.NewToolResultText(output), nil
}

// SearchJournal searches systemd journal entries using journalctl grep.
func SearchJournal(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pattern, err := req.RequireString("pattern")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	lines := req.GetInt("lines", defaultLines)
	if lines <= 0 || lines > maxLines {
		lines = defaultLines
	}

	args := []string{"--no-pager", "-n", fmt.Sprintf("%d", lines), "--grep", pattern}

	if unit := strings.TrimSpace(req.GetString("unit", "")); unit != "" {
		if err := validateIdentifier(unit); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		args = append(args, "-u", unit)
	}

	if since := strings.TrimSpace(req.GetString("since", "")); since != "" {
		if err := validateTimeString(since); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		args = append(args, "--since", since)
	}

	output, err := runJournalctl(args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("journalctl failed: %v", err)), nil
	}

	if strings.TrimSpace(output) == "" {
		return mcp.NewToolResultText(fmt.Sprintf("No journal entries found matching %q", pattern)), nil
	}

	return mcp.NewToolResultText(output), nil
}

// ---- helpers ----

// validateLogPath ensures the resolved path is inside /var/log and is not a symlink
// pointing outside it, preventing path traversal attacks.
func validateLogPath(path string) error {
	clean := filepath.Clean(path)
	logDirClean := filepath.Clean(logDir)
	if !strings.HasPrefix(clean, logDirClean+string(filepath.Separator)) {
		return fmt.Errorf("path must be inside %s", logDir)
	}
	// Resolve symlinks and re-check.
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return fmt.Errorf("cannot resolve path: %v", err)
	}
	if !strings.HasPrefix(resolved, logDirClean+string(filepath.Separator)) {
		return fmt.Errorf("resolved path %s is outside %s", resolved, logDir)
	}
	return nil
}

// validateIdentifier ensures a systemd unit name or similar string contains only
// safe characters (letters, digits, hyphens, underscores, dots, @, :).
func validateIdentifier(s string) error {
	for _, r := range s {
		if !isIdentRune(r) {
			return fmt.Errorf("invalid character %q in identifier %q", r, s)
		}
	}
	return nil
}

func isIdentRune(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_' || r == '.' || r == '@' || r == ':'
}

// validateTimeString allows common journalctl time formats such as
// "2024-01-01", "2024-01-01 12:00:00", "1 hour ago", "yesterday", "today".
func validateTimeString(s string) error {
	allowed := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 -:_"
	for _, r := range s {
		if !strings.ContainsRune(allowed, r) {
			return fmt.Errorf("invalid character %q in time string", r)
		}
	}
	return nil
}

// tailFile returns the last n lines of a file.
func tailFile(path string, n int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	// Use a larger buffer for long log lines.
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

// grepFile returns up to maxResults lines matching pattern (case-insensitive substring).
func grepFile(path, pattern string, maxResults int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	lowerPattern := strings.ToLower(pattern)
	var matches []string

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.ToLower(line), lowerPattern) {
			matches = append(matches, line)
			if len(matches) >= maxResults {
				break
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return matches, nil
}

// runJournalctl executes journalctl with the given arguments and returns its output.
func runJournalctl(args []string) (string, error) {
	cmd := exec.Command(journalctlPath, args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%v: %s", exitErr, string(exitErr.Stderr))
		}
		return "", err
	}
	return string(out), nil
}
