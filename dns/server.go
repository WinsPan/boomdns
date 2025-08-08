package dns

import (
    "context"
    "errors"
    "log"
    "net"
    "strings"
    "sync"
    "time"

    mdns "github.com/miekg/dns"
    "github.com/prometheus/client_golang/prometheus"
)

type Server struct {
	cfg *Config
	mu  sync.RWMutex
	// 最近查询日志（环形缓冲）
	logs []QueryLog

    // 预编译的规则列表（已标准化为小写、带或不带前导点的一致形式）
    compiledChina []string
    compiledGfw   []string
    compiledAds   []string
}

func NewServer(cfg *Config) (*Server, error) {
    srv := &Server{cfg: cfg}
    _ = srv.ReloadRules()
    return srv, nil
}

func (s *Server) ReloadRules() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.compiledChina = normalizeSuffixes(s.cfg.ChinaDomains)
    s.compiledGfw = normalizeSuffixes(s.cfg.GfwDomains)
    s.compiledAds = normalizeSuffixes(s.cfg.AdDomains)
    return nil
}

// SetRules 原子更新规则（由 SyncManager 或 API 调用）
func (s *Server) SetRules(china, gfw, ads []string) {
    s.mu.Lock()
    s.cfg.ChinaDomains = china
    s.cfg.GfwDomains = gfw
    s.cfg.AdDomains = ads
    s.mu.Unlock()
    _ = s.ReloadRules()
}

func (s *Server) ServeUDP(conn *net.UDPConn) {
	srv := &mdns.Server{Handler: mdns.HandlerFunc(s.handle), PacketConn: conn}
	if err := srv.ActivateAndServe(); err != nil {
		log.Printf("udp serve err: %v", err)
	}
}

func (s *Server) ServeTCP(ln net.Listener) {
	srv := &mdns.Server{Handler: mdns.HandlerFunc(s.handle), Listener: ln}
	if err := srv.ActivateAndServe(); err != nil {
		log.Printf("tcp serve err: %v", err)
	}
}

func (s *Server) handle(w mdns.ResponseWriter, r *mdns.Msg) {
	if len(r.Question) == 0 {
		_ = w.WriteMsg(new(mdns.Msg))
		return
	}
	q := r.Question[0]
	name := strings.TrimSuffix(strings.ToLower(q.Name), ".")

	// 分流：广告 -> adguard；gfw -> intl；china -> china；其他：先 china 失败再 intl
    var upstreams []Upstream
	decision := ""
    if s.match(name, s.compiledAds) && s.cfg.AdguardAddr != "" {
		upstreams = []Upstream{{Address: s.cfg.AdguardAddr}}
		decision = "adguard"
    } else if s.match(name, s.compiledGfw) {
		upstreams = s.cfg.IntlUpstreams
		decision = "intl"
    } else if s.match(name, s.compiledChina) {
		upstreams = s.cfg.ChinaUpstreams
		decision = "china"
	} else {
		// fallback：china -> intl
        if resp, err := s.forward(context.Background(), r, s.cfg.ChinaUpstreams, "china"); err == nil && hasAnswer(resp) {
			s.addLog(name, "china")
            queryCounter.WithLabelValues("china").Inc()
			_ = w.WriteMsg(resp)
			return
		}
		upstreams = s.cfg.IntlUpstreams
		decision = "intl"
	}

    resp, err := s.forward(context.Background(), r, upstreams, decision)
	if err != nil {
		s.writeServFail(w, r)
		return
	}
    s.addLog(name, decision)
    queryCounter.WithLabelValues(decision).Inc()
	_ = w.WriteMsg(resp)
}

func hasAnswer(m *mdns.Msg) bool { return m != nil && (len(m.Answer) > 0 || len(m.Ns) > 0) }

func (s *Server) writeServFail(w mdns.ResponseWriter, req *mdns.Msg) {
	m := new(mdns.Msg)
	m.SetRcode(req, mdns.RcodeServerFailure)
	_ = w.WriteMsg(m)
}

func (s *Server) match(name string, suffixes []string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sfx := range suffixes {
        sfx = strings.ToLower(strings.TrimSpace(sfx))
        if sfx == "" {
			continue
		}
		if strings.HasSuffix(name, strings.TrimPrefix(sfx, ".")) {
			return true
		}
	}
	return false
}

func (s *Server) forward(ctx context.Context, req *mdns.Msg, ups []Upstream, target string) (*mdns.Msg, error) {
	var lastErr error
	for _, up := range ups {
        c := &mdns.Client{Net: netProto(up.Address), Timeout: 3 * time.Second}
        start := time.Now()
		// 这里未直接支持 socks5，建议使用 mihomo 暴露本地 DNS 端口，或在系统层做 socks5 透明转发
		resp, _, err := c.Exchange(req, up.Address)
        upstreamLatency.WithLabelValues(target).Observe(time.Since(start).Seconds())
		if err == nil && resp != nil {
			return resp, nil
		}
        upstreamFailures.WithLabelValues(target).Inc()
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no upstream")
	}
	return nil, lastErr
}

func netProto(addr string) string {
	if strings.HasPrefix(addr, "tls://") || strings.HasPrefix(addr, "https://") {
		return "tcp"
	}
	return "udp"
}

type QueryLog struct {
	Time  time.Time `json:"time"`
	Name  string    `json:"name"`
	Route string    `json:"route"`
}

func (s *Server) addLog(name, route string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	const max = 1000
	s.logs = append(s.logs, QueryLog{Time: time.Now(), Name: name, Route: route})
	if len(s.logs) > max {
		s.logs = s.logs[len(s.logs)-max:]
	}
}

func (s *Server) GetLogs(limit int) []QueryLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.logs) {
		limit = len(s.logs)
	}
	out := make([]QueryLog, limit)
	copy(out, s.logs[len(s.logs)-limit:])
	return out
}

// normalizeSuffixes 将规则统一为小写去空白的后缀匹配形式
func normalizeSuffixes(in []string) []string {
    out := make([]string, 0, len(in))
    for _, s := range in {
        s = strings.ToLower(strings.TrimSpace(s))
        if s == "" {
            continue
        }
        // 与 normalizeDomain 一致，确保有前导点，便于一致匹配
        if !strings.HasPrefix(s, ".") {
            if d, ok := normalizeDomain(s); ok {
                s = d
            } else {
                continue
            }
        }
        out = append(out, s)
    }
    return out
}

var (
    queryCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "boomdns_queries_total",
            Help: "Total DNS queries by route decision",
        },
        []string{"route"},
    )
    upstreamLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "boomdns_upstream_request_duration_seconds",
            Help:    "DNS upstream request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"target"},
    )
    upstreamFailures = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "boomdns_upstream_failures_total",
            Help: "Total DNS upstream request failures",
        },
        []string{"target"},
    )
)

func init() {
    prometheus.MustRegister(queryCounter, upstreamLatency, upstreamFailures)
}
