// loggerMCP is a Model Context Protocol (MCP) server that exposes tools for
// reading and searching Ubuntu system logs via the AI assistant interface.
package main

import (
	"fmt"
	"os"

	"github.com/AlexeySpiridonov/loggerMCP/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer(
		"loggerMCP",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	registerTools(s)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func registerTools(s *server.MCPServer) {
	// list_log_files
	s.AddTool(
		mcp.NewTool("list_log_files",
			mcp.WithDescription("List all readable log files available under /var/log on the Ubuntu system."),
		),
		tools.ListLogFiles,
	)

	// read_log_file
	s.AddTool(
		mcp.NewTool("read_log_file",
			mcp.WithDescription("Read the last N lines of a log file located under /var/log."),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("Absolute path to the log file (must be inside /var/log)."),
			),
			mcp.WithNumber("lines",
				mcp.Description("Number of lines to return from the end of the file (default: 100, max: 10000)."),
			),
		),
		tools.ReadLogFile,
	)

	// search_log_file
	s.AddTool(
		mcp.NewTool("search_log_file",
			mcp.WithDescription("Search a log file for lines containing a given pattern (case-insensitive substring match)."),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("Absolute path to the log file (must be inside /var/log)."),
			),
			mcp.WithString("pattern",
				mcp.Required(),
				mcp.Description("Text pattern to search for (case-insensitive)."),
			),
			mcp.WithNumber("max_results",
				mcp.Description("Maximum number of matching lines to return (default: 100, max: 10000)."),
			),
		),
		tools.SearchLogFile,
	)

	// read_journal
	s.AddTool(
		mcp.NewTool("read_journal",
			mcp.WithDescription("Read systemd journal entries using journalctl. Optionally filter by unit and time range."),
			mcp.WithNumber("lines",
				mcp.Description("Number of most recent journal entries to return (default: 100, max: 10000)."),
			),
			mcp.WithString("unit",
				mcp.Description("Systemd unit name to filter by (e.g. \"ssh.service\", \"nginx\"). Leave empty for all units."),
			),
			mcp.WithString("since",
				mcp.Description("Show entries since this time. Accepts formats like \"2024-01-15\", \"2024-01-15 12:00:00\", \"1 hour ago\", \"yesterday\", \"today\"."),
			),
			mcp.WithBoolean("current_boot",
				mcp.Description("If true, show entries from the current boot only."),
			),
		),
		tools.ReadJournal,
	)

	// search_journal
	s.AddTool(
		mcp.NewTool("search_journal",
			mcp.WithDescription("Search systemd journal entries for lines matching a pattern using journalctl --grep."),
			mcp.WithString("pattern",
				mcp.Required(),
				mcp.Description("Regular expression pattern to search for in journal messages."),
			),
			mcp.WithNumber("lines",
				mcp.Description("Maximum number of matching entries to return (default: 100, max: 10000)."),
			),
			mcp.WithString("unit",
				mcp.Description("Systemd unit name to filter by (e.g. \"ssh.service\"). Leave empty for all units."),
			),
			mcp.WithString("since",
				mcp.Description("Show entries since this time (e.g. \"1 hour ago\", \"today\", \"2024-01-15 12:00:00\")."),
			),
		),
		tools.SearchJournal,
	)
}
