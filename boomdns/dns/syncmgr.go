package dns

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

type SyncManager struct {
	cfg    *Config
	server *Server
	httpc  *http.Client
}

func NewSyncManager(cfg *Config, server *Server) *SyncManager {
	return &SyncManager{
		cfg:    cfg,
		server: server,
		httpc:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (m *SyncManager) Start(ctx context.Context) {
	// determine interval
	iv := 6 * time.Hour
	if d, err := time.ParseDuration(strings.TrimSpace(m.cfg.RuleSync.RefreshInterval)); err == nil && d > 0 {
		iv = d
	}
	ticker := time.NewTicker(iv)
	defer ticker.Stop()
	// initial sync
	_ = m.SyncNow(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = m.SyncNow(ctx)
		}
	}
}

func (m *SyncManager) SyncNow(ctx context.Context) error {
	var chinaSet = map[string]struct{}{}
	var gfwSet = map[string]struct{}{}
	var adSet = map[string]struct{}{}

	// china lists (dnsmasq format or plain domains)
	for _, u := range m.cfg.RuleSync.ChinaListURLs {
		if strings.TrimSpace(u) == "" {
			continue
		}
		b, err := m.fetch(ctx, u)
		if err != nil {
			continue
		}
		for d := range parseDomainsFromChinaList(string(b)) {
			chinaSet[d] = struct{}{}
		}
	}

	// gfwlist (base64-encoded rules)
	for _, u := range m.cfg.RuleSync.GFWListURLs {
		if strings.TrimSpace(u) == "" {
			continue
		}
		b64, err := m.fetch(ctx, u)
		if err != nil {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(string(b64))
		if err != nil {
			continue
		}
		for d := range parseDomainsFromGFWList(string(raw)) {
			gfwSet[d] = struct{}{}
		}
	}

	// ad lists (hosts/address or plain domains)
	for _, u := range m.cfg.RuleSync.AdListURLs {
		if strings.TrimSpace(u) == "" {
			continue
		}
		b, err := m.fetch(ctx, u)
		if err != nil {
			continue
		}
		for d := range parseDomainsGeneric(string(b)) {
			adSet[d] = struct{}{}
		}
	}

	// convert to sorted slices
	china := setToSlice(chinaSet)
	gfw := setToSlice(gfwSet)
	ads := setToSlice(adSet)

	if len(china) == 0 && len(gfw) == 0 && len(ads) == 0 {
		return errors.New("rule sync got empty sets")
	}
	// update server rules atomically
	m.server.SetRules(china, gfw, ads)
	return nil
}

func (m *SyncManager) fetch(ctx context.Context, url string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := m.httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, errors.New(resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func setToSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for d := range set {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

// parsers

var reDomain = regexp.MustCompile(`(?i)([a-z0-9][a-z0-9-]*\.)+[a-z]{2,}`)

func normalizeDomain(d string) (string, bool) {
	d = strings.ToLower(strings.TrimSpace(d))
	if d == "" {
		return "", false
	}
	// strip leading dots and scheme patterns
	d = strings.TrimPrefix(d, ".")
	d = strings.TrimPrefix(d, "||")
	d = strings.TrimPrefix(d, "|")
	d = strings.TrimPrefix(d, "@")
	if !reDomain.MatchString(d) {
		m := reDomain.FindString(d)
		if m == "" {
			return "", false
		}
		d = m
	}
	return "." + d, true
}

func parseDomainsFromChinaList(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// dnsmasq: server=/example.com/114.114.114.114
		if strings.HasPrefix(line, "server=/") {
			seg := strings.TrimPrefix(line, "server=/")
			idx := strings.Index(seg, "/")
			if idx > 0 {
				seg = seg[:idx]
			}
			if d, ok := normalizeDomain(seg); ok {
				out[d] = struct{}{}
			}
			continue
		}
		if d, ok := normalizeDomain(line); ok {
			out[d] = struct{}{}
		}
	}
	return out
}

func parseDomainsFromGFWList(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "@@") {
			continue
		}
		line = strings.TrimPrefix(line, "||")
		line = strings.TrimPrefix(line, "|")
		line = strings.TrimPrefix(line, ".")
		if d, ok := normalizeDomain(line); ok {
			out[d] = struct{}{}
		}
	}
	return out
}

func parseDomainsGeneric(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// hosts: 0.0.0.0 domain
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			if d, ok := normalizeDomain(fields[1]); ok {
				out[d] = struct{}{}
			}
			continue
		}
		// address=/domain/
		if strings.HasPrefix(line, "address=/") {
			seg := strings.TrimPrefix(line, "address=/")
			idx := strings.Index(seg, "/")
			if idx > 0 {
				seg = seg[:idx]
			}
			if d, ok := normalizeDomain(seg); ok {
				out[d] = struct{}{}
			}
			continue
		}
		if d, ok := normalizeDomain(line); ok {
			out[d] = struct{}{}
		}
	}
	return out
}
