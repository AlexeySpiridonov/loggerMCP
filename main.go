package main

import (
	"bufio"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gopkg.in/yaml.v3"
)

const serverName = "loggerMCP"
const serverVersion = "1.0.0"
const authVerifiedHeader = "X-LoggerMCP-Authorized"

type Config struct {
	AccessKey           string `yaml:"access_key"`
	SyslogPath          string `yaml:"syslog_path"`
	Port                int    `yaml:"port"`
	TLS                 bool   `yaml:"tls"`
	CertFile            string `yaml:"cert_file"`
	KeyFile             string `yaml:"key_file"`
	EncryptionKey       string `yaml:"encryption_key"`
	PublicBaseURL       string `yaml:"public_base_url"`
	ManifestName        string `yaml:"manifest_name"`
	ManifestTitle       string `yaml:"manifest_title"`
	ManifestDescription string `yaml:"manifest_description"`
	ManifestVersion     string `yaml:"manifest_version"`
	ManifestPath        string `yaml:"manifest_path"`
	ManifestRemoteType  string `yaml:"manifest_remote_type"`
	ManifestRemoteURL   string `yaml:"manifest_remote_url"`
	HealthPath          string `yaml:"health_path"`
}

type manifestRemote struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type manifestResponse struct {
	Description string           `json:"description"`
	Name        string           `json:"name"`
	Remotes     []manifestRemote `json:"remotes"`
	Title       string           `json:"title"`
	Version     string           `json:"version"`
}

type healthResponse struct {
	Status            string `json:"status"`
	Version           string `json:"version"`
	Time              string `json:"time"`
	AuthRequired      bool   `json:"auth_required"`
	TLS               bool   `json:"tls"`
	LogFile           string `json:"log_file"`
	LogFileAccessible bool   `json:"log_file_accessible"`
	ManifestURL       string `json:"manifest_url"`
	RemoteURL         string `json:"remote_url"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Port == 0 {
		cfg.Port = 7777
	}
	if cfg.SyslogPath == "" {
		cfg.SyslogPath = "/var/log/syslog"
	}
	if cfg.CertFile == "" {
		cfg.CertFile = "cert.pem"
	}
	if cfg.KeyFile == "" {
		cfg.KeyFile = "key.pem"
	}
	if cfg.ManifestName == "" {
		cfg.ManifestName = "logger.local/mcp"
	}
	if cfg.ManifestTitle == "" {
		cfg.ManifestTitle = serverName
	}
	if cfg.ManifestDescription == "" {
		cfg.ManifestDescription = "Remote MCP server for Ubuntu syslog search workflows."
	}
	if cfg.ManifestVersion == "" {
		cfg.ManifestVersion = serverVersion
	}
	if cfg.ManifestPath == "" {
		cfg.ManifestPath = "/manifest"
	}
	if cfg.ManifestRemoteType == "" {
		cfg.ManifestRemoteType = "sse"
	}
	if cfg.HealthPath == "" {
		cfg.HealthPath = "/health"
	}
	return &cfg, nil
}

func serverScheme(cfg *Config) string {
	if cfg.TLS {
		return "https"
	}
	return "http"
}

func baseURL(cfg *Config) string {
	if cfg.PublicBaseURL != "" {
		return strings.TrimRight(cfg.PublicBaseURL, "/")
	}
	return fmt.Sprintf("%s://localhost:%d", serverScheme(cfg), cfg.Port)
}

func manifestRemoteURL(cfg *Config) string {
	if cfg.ManifestRemoteURL != "" {
		return cfg.ManifestRemoteURL
	}
	return baseURL(cfg) + "/sse"
}

func isAccessKeyValid(cfg *Config, accessKey string) bool {
	if cfg.AccessKey == "" {
		return true
	}
	return accessKey == cfg.AccessKey
}

func extractAccessKeyFromHeaders(headers http.Header) string {
	if headers == nil {
		return ""
	}

	authorization := strings.TrimSpace(headers.Get("Authorization"))
	if authorization != "" {
		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	return strings.TrimSpace(headers.Get("X-Access-Key"))
}

func extractAccessKeyFromRequest(r *http.Request) string {
	if accessKey := extractAccessKeyFromHeaders(r.Header); accessKey != "" {
		return accessKey
	}
	return strings.TrimSpace(r.URL.Query().Get("access_key"))
}

func accessKeyMiddleware(cfg *Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAccessKeyValid(cfg, extractAccessKeyFromRequest(r)) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		authorizedRequest := r.Clone(r.Context())
		authorizedRequest.Header = r.Header.Clone()
		authorizedRequest.Header.Set(authVerifiedHeader, "true")
		next.ServeHTTP(w, authorizedRequest)
	})
}

func isAuthorizedToolRequest(cfg *Config, request mcp.CallToolRequest, args map[string]any) bool {
	if cfg.AccessKey == "" {
		return true
	}

	if request.Header.Get(authVerifiedHeader) == "true" {
		return true
	}

	if isAccessKeyValid(cfg, extractAccessKeyFromHeaders(request.Header)) {
		return true
	}

	toolAccessKey, _ := args["access_key"].(string)
	return isAccessKeyValid(cfg, toolAccessKey)
}

func buildManifest(cfg *Config) manifestResponse {
	return manifestResponse{
		Description: cfg.ManifestDescription,
		Name:        cfg.ManifestName,
		Remotes: []manifestRemote{{
			Type: cfg.ManifestRemoteType,
			URL:  manifestRemoteURL(cfg),
		}},
		Title:   cfg.ManifestTitle,
		Version: cfg.ManifestVersion,
	}
}

func manifestHandler(cfg *Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(buildManifest(cfg)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func healthHandler(cfg *Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		logFileAccessible := false
		if file, err := os.Open(cfg.SyslogPath); err == nil {
			logFileAccessible = true
			file.Close()
		}

		status := "ok"
		statusCode := http.StatusOK
		if !logFileAccessible {
			status = "degraded"
			statusCode = http.StatusServiceUnavailable
		}

		response := healthResponse{
			Status:            status,
			Version:           serverVersion,
			Time:              time.Now().UTC().Format(time.RFC3339),
			AuthRequired:      cfg.AccessKey != "",
			TLS:               cfg.TLS,
			LogFile:           cfg.SyslogPath,
			LogFileAccessible: logFileAccessible,
			ManifestURL:       baseURL(cfg) + cfg.ManifestPath,
			RemoteURL:         manifestRemoteURL(cfg),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// ensureTLSCert generates a self-signed certificate if files don't exist.
func ensureTLSCert(certFile, keyFile string) error {
	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			return nil // both files already exist
		}
	}

	log.Println("Generating self-signed TLS certificate...")

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("key generation: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("serial number generation: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"loggerMCP"},
			CommonName:   "loggerMCP",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour), // 10 years
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("0.0.0.0")},
		DNSNames:              []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("certificate creation: %w", err)
	}

	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("writing cert: %w", err)
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return err
	}

	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("writing key: %w", err)
	}
	defer keyOut.Close()
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return err
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return err
	}

	log.Printf("Certificate saved: %s, %s", certFile, keyFile)
	return nil
}

// parseSyslogTime parses a timestamp from a syslog line.
// Format: "Apr 15 10:30:00"
func parseSyslogTime(line string) (time.Time, bool) {
	if len(line) < 15 {
		return time.Time{}, false
	}
	timeStr := line[:15]
	t, err := time.Parse("Jan  2 15:04:05", timeStr)
	if err != nil {
		t, err = time.Parse("Jan 2 15:04:05", timeStr)
		if err != nil {
			return time.Time{}, false
		}
	}
	now := time.Now()
	t = t.AddDate(now.Year(), 0, 0)
	return t, true
}

// matchWildcard checks if text matches a pattern with * (wildcard) support.
// Example: "error*disk" matches "error on disk", "error: disk full", etc.
func matchWildcard(pattern, text string) bool {
	pattern = strings.ToLower(pattern)
	text = strings.ToLower(text)

	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return strings.Contains(text, pattern)
	}

	pos := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		idx := strings.Index(text[pos:], part)
		if idx < 0 {
			return false
		}
		if i == 0 && !strings.HasPrefix(pattern, "*") && idx != 0 {
			return false
		}
		pos += idx + len(part)
	}
	if !strings.HasSuffix(pattern, "*") && pos != len(text) {
		return false
	}
	return true
}

// parseInputDate parses a date from user input.
func parseInputDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported date format: %s (use 2006-01-02 or 2006-01-02T15:04:05)", s)
}

// encryptAESGCM encrypts text using AES-256-GCM.
// The key is hashed via SHA-256 to produce exactly 32 bytes.
// Returns base64(nonce + ciphertext).
func encryptAESGCM(plaintext, key string) (string, error) {
	keyHash := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func main() {
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	s := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
	)

	readLogsTool := mcp.NewTool("read_logs",
		mcp.WithDescription("Read and search syslog entries with date filtering, pattern matching, and pagination"),
		mcp.WithString("access_key",
			mcp.Description("Optional legacy access key. Not required when transport auth is used."),
		),
		mcp.WithString("start_date",
			mcp.Description("Start date filter (format: 2006-01-02 or 2006-01-02T15:04:05)"),
		),
		mcp.WithString("end_date",
			mcp.Description("End date filter (format: 2006-01-02 or 2006-01-02T15:04:05)"),
		),
		mcp.WithString("pattern",
			mcp.Description("Substring filter with * (wildcard) support. Example: 'error*disk'"),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number (default: 1)"),
		),
		mcp.WithNumber("page_size",
			mcp.Description("Entries per page (default: 100, max: 1000)"),
		),
		mcp.WithBoolean("encrypt",
			mcp.Description("Encrypt response with AES-256-GCM (key from config)"),
		),
	)

	s.AddTool(readLogsTool, readLogsHandler(cfg))

	sseServer := server.NewSSEServer(s,
		server.WithBaseURL(baseURL(cfg)),
		server.WithAppendQueryToMessageEndpoint(),
	)
	mux := http.NewServeMux()
	mux.Handle(cfg.ManifestPath, manifestHandler(cfg))
	mux.Handle(cfg.HealthPath, healthHandler(cfg))
	mux.Handle("/", accessKeyMiddleware(cfg, sseServer))

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if cfg.TLS {
		if err := ensureTLSCert(cfg.CertFile, cfg.KeyFile); err != nil {
			log.Fatalf("TLS error: %v", err)
		}

		tlsCert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			log.Fatalf("Failed to load certificate: %v", err)
		}

		httpServer.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
			MinVersion:   tls.VersionTLS12,
		}

		log.Printf("loggerMCP started on https://0.0.0.0:%d (TLS)", cfg.Port)
		log.Printf("Log file: %s", cfg.SyslogPath)
		if cfg.AccessKey != "" {
			log.Printf("Access key auth: enabled")
		}
		log.Printf("Manifest endpoint: %s", baseURL(cfg)+cfg.ManifestPath)
		log.Printf("Health endpoint: %s", baseURL(cfg)+cfg.HealthPath)
		if err := httpServer.ListenAndServeTLS("", ""); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		log.Printf("loggerMCP started on http://0.0.0.0:%d", cfg.Port)
		log.Printf("Log file: %s", cfg.SyslogPath)
		if cfg.AccessKey != "" {
			log.Printf("Access key auth: enabled")
		}
		log.Printf("Manifest endpoint: %s", baseURL(cfg)+cfg.ManifestPath)
		log.Printf("Health endpoint: %s", baseURL(cfg)+cfg.HealthPath)
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}

func readLogsHandler(cfg *Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()

		// Verify access key
		if !isAuthorizedToolRequest(cfg, request, args) {
			return mcp.NewToolResultError("unauthorized: invalid access key"), nil
		}

		// Pagination parameters
		page := 1
		pageSize := 100
		if p, ok := args["page"].(float64); ok && p > 0 {
			page = int(p)
		}
		if ps, ok := args["page_size"].(float64); ok && ps > 0 {
			pageSize = int(ps)
			if pageSize > 1000 {
				pageSize = 1000
			}
		}

		// Parse dates
		var startDate, endDate time.Time
		var hasStart, hasEnd bool

		if sd, ok := args["start_date"].(string); ok && sd != "" {
			t, err := parseInputDate(sd)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid start_date: %v", err)), nil
			}
			startDate = t
			hasStart = true
		}
		if ed, ok := args["end_date"].(string); ok && ed != "" {
			t, err := parseInputDate(ed)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid end_date: %v", err)), nil
			}
			endDate = t
			hasEnd = true
		}

		pattern, _ := args["pattern"].(string)

		// Read and filter log file
		file, err := os.Open(cfg.SyslogPath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to open log file: %v", err)), nil
		}
		defer file.Close()

		var filtered []string
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			// Date filter
			if hasStart || hasEnd {
				logTime, ok := parseSyslogTime(line)
				if ok {
					if hasStart && logTime.Before(startDate) {
						continue
					}
					if hasEnd && logTime.After(endDate) {
						continue
					}
				}
			}

			// Pattern filter
			if pattern != "" && !matchWildcard(pattern, line) {
				continue
			}

			filtered = append(filtered, line)
		}

		if err := scanner.Err(); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("error reading log file: %v", err)), nil
		}

		// Pagination
		total := len(filtered)
		totalPages := (total + pageSize - 1) / pageSize
		if totalPages == 0 {
			totalPages = 1
		}
		if page > totalPages {
			page = totalPages
		}

		start := (page - 1) * pageSize
		end := start + pageSize
		if end > total {
			end = total
		}

		var result strings.Builder
		result.WriteString(fmt.Sprintf("Total: %d entries | Page %d/%d (size: %d)\n", total, page, totalPages, pageSize))
		result.WriteString("---\n")

		if total > 0 {
			for i := start; i < end; i++ {
				result.WriteString(filtered[i])
				result.WriteString("\n")
			}
		} else {
			result.WriteString("No entries found.\n")
		}

		text := result.String()

		// Encrypt if requested and key is configured
		wantEncrypt, _ := args["encrypt"].(bool)
		if wantEncrypt {
			if cfg.EncryptionKey == "" {
				return mcp.NewToolResultError("encryption_key is not set in server config"), nil
			}
			encrypted, err := encryptAESGCM(text, cfg.EncryptionKey)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("encryption error: %v", err)), nil
			}
			return mcp.NewToolResultText("ENC:" + encrypted), nil
		}

		return mcp.NewToolResultText(text), nil
	}
}
