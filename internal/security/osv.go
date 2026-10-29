package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/domain"
)

const defaultOSVEndpoint = "https://api.osv.dev/v1/query"

// Compiled once at package initialization per golang-design-patterns.
var semverExtractRegex = regexp.MustCompile(`(?i)(?:^|[^0-9.])(?:go|v)?(\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?)\b`)

// ExtractSemver extracts the first clean semantic version from raw --version output.
func ExtractSemver(raw string) string {
	matches := semverExtractRegex.FindStringSubmatch(raw)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// OSVOption configures an OSVClient.
type OSVOption func(*OSVClient)

// WithOSVEndpoint overrides the OSV query URL (useful for testing and enterprise mirrors).
func WithOSVEndpoint(endpoint string) OSVOption {
	return func(c *OSVClient) {
		if endpoint != "" {
			c.endpoint = endpoint
		}
	}
}

// WithOSVCacheDir enables local disk caching of OSV responses with a configurable TTL.
func WithOSVCacheDir(dir string, ttl time.Duration) OSVOption {
	return func(c *OSVClient) {
		c.cacheDir = dir
		if ttl > 0 {
			c.cacheTTL = ttl
		}
	}
}

// OSVClient queries the Open Source Vulnerabilities database with tuned pooling and local cache fallback.
type OSVClient struct {
	httpClient *http.Client
	endpoint   string
	cacheDir   string
	cacheTTL   time.Duration
}

// NewOSVClient creates an OSVClient with timeout and functional options.
func NewOSVClient(timeout time.Duration, opts ...OSVOption) *OSVClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        16,
		MaxIdleConnsPerHost: 8,
		IdleConnTimeout:     60 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	client := &OSVClient{
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		endpoint: defaultOSVEndpoint,
		cacheTTL: 6 * time.Hour,
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

type osvQueryRequest struct {
	Version string     `json:"version,omitempty"`
	Package osvPackage `json:"package"`
}

type osvPackage struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem,omitempty"`
}

type osvVulnEntry struct {
	ID               string `json:"id"`
	Summary          string `json:"summary"`
	Details          string `json:"details"`
	DatabaseSpecific struct {
		Severity string `json:"severity"`
	} `json:"database_specific"`
	Severity []struct {
		Type  string `json:"type"`
		Score string `json:"score"`
	} `json:"severity"`
}

type osvQueryResponse struct {
	Vulns []osvVulnEntry `json:"vulns"`
}

type osvCacheEnvelope struct {
	CachedAt time.Time             `json:"cached_at"`
	Findings []domain.AuditFinding `json:"findings"`
}

// Query checks OSV for vulnerabilities affecting toolName at rawVersion.
func (c *OSVClient) Query(ctx context.Context, toolName, rawVersion string) ([]domain.AuditFinding, error) {
	return c.QueryWithEcosystem(ctx, toolName, "", rawVersion)
}

// QueryWithEcosystem checks OSV with an explicit package ecosystem when known (e.g. "Go", "crates.io", "PyPI", "npm").
func (c *OSVClient) QueryWithEcosystem(
	ctx context.Context,
	toolName string,
	ecosystem string,
	rawVersion string,
) ([]domain.AuditFinding, error) {
	version := ExtractSemver(rawVersion)
	if version == "" {
		return []domain.AuditFinding{}, nil
	}

	cacheKey := ComputeSHA256([]byte(toolName + "|" + ecosystem + "|" + version))
	if cached, ok := c.readCache(cacheKey, false); ok {
		return cached, nil
	}

	reqBody := osvQueryRequest{
		Version: version,
		Package: osvPackage{
			Name:      toolName,
			Ecosystem: ecosystem,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling osv request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("building osv request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if stale, ok := c.readCache(cacheKey, true); ok {
			return stale, nil
		}
		return []domain.AuditFinding{}, fmt.Errorf("osv query failed for %s@%s: %w", toolName, version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if stale, ok := c.readCache(cacheKey, true); ok {
			return stale, nil
		}
		return []domain.AuditFinding{}, fmt.Errorf("osv returned HTTP %d for %s@%s", resp.StatusCode, toolName, version)
	}

	var res osvQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decoding osv response: %w", err)
	}

	now := time.Now().UTC()
	findings := make([]domain.AuditFinding, 0, len(res.Vulns))
	for _, v := range res.Vulns {
		summary := v.Summary
		if summary == "" && len(v.Details) > 0 {
			summary = v.Details
			if len(summary) > 140 {
				summary = summary[:139] + "…"
			}
		}
		findings = append(findings, domain.AuditFinding{
			ID:              v.ID,
			ResourceID:      toolName,
			ToolName:        toolName,
			DetectedVersion: version,
			Severity:        classifyOSVSeverity(v),
			Summary:         summary,
			AdvisoryURL:     fmt.Sprintf("https://osv.dev/vulnerability/%s", v.ID),
			DiscoveredAt:    now,
		})
	}

	c.writeCache(cacheKey, findings)
	return findings, nil
}

func classifyOSVSeverity(v osvVulnEntry) domain.Severity {
	switch strings.ToUpper(strings.TrimSpace(v.DatabaseSpecific.Severity)) {
	case "CRITICAL":
		return domain.SeverityCritical
	case "HIGH":
		return domain.SeverityHigh
	case "MODERATE", "MEDIUM":
		return domain.SeverityMedium
	case "LOW":
		return domain.SeverityLow
	}
	return domain.SeverityHigh
}

func (c *OSVClient) readCache(key string, allowExpired bool) ([]domain.AuditFinding, bool) {
	if c.cacheDir == "" {
		return nil, false
	}
	raw, err := os.ReadFile(filepath.Join(c.cacheDir, key+".json"))
	if err != nil {
		return nil, false
	}
	var env osvCacheEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, false
	}
	if !allowExpired && time.Since(env.CachedAt) > c.cacheTTL {
		return nil, false
	}
	if env.Findings == nil {
		return []domain.AuditFinding{}, true
	}
	return env.Findings, true
}

func (c *OSVClient) writeCache(key string, findings []domain.AuditFinding) {
	if c.cacheDir == "" {
		return
	}
	if err := os.MkdirAll(c.cacheDir, 0o700); err != nil {
		return
	}
	raw, err := json.Marshal(osvCacheEnvelope{
		CachedAt: time.Now().UTC(),
		Findings: findings,
	})
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(c.cacheDir, key+".json"), raw, 0o600)
}
