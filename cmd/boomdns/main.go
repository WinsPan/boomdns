package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v3"

	"github.com/winspan/boomdns/internal/dns"
)

// LoadConfig 加载配置文件
func LoadConfig(path string) (*dns.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg dns.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// DNS server
	server, err := dns.NewServer(cfg)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	// Start UDP
	udpAddr, err := net.ResolveUDPAddr("udp", cfg.ListenDNS)
	if err != nil {
		log.Fatalf("udp addr: %v", err)
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("listen udp: %v", err)
	}
	go server.ServeUDP(udpConn)

	// Start TCP
	tcpLn, err := net.Listen("tcp", cfg.ListenDNS)
	if err != nil {
		log.Fatalf("listen tcp: %v", err)
	}
	go server.ServeTCP(tcpLn)

	// Admin HTTP
	r := chi.NewRouter()

	// 添加基本的管理界面路由
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>BoomDNS 系统总览</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8f9fa; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 20px 0; }
        .header h1 { text-align: center; font-size: 2.5em; margin-bottom: 10px; }
        .header p { text-align: center; opacity: 0.9; }
        .container { max-width: 1400px; margin: 0 auto; padding: 20px; }
        .overview-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(350px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .card { background: white; border-radius: 12px; padding: 25px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); transition: transform 0.3s, box-shadow 0.3s; }
        .card:hover { transform: translateY(-5px); box-shadow: 0 8px 30px rgba(0,0,0,0.15); }
        .card h3 { color: #2c3e50; margin-bottom: 20px; font-size: 1.3em; display: flex; align-items: center; }
        .card h3 .icon { margin-right: 10px; font-size: 1.5em; }
        .metric { display: flex; justify-content: space-between; align-items: center; margin: 15px 0; padding: 10px 0; border-bottom: 1px solid #eee; }
        .metric:last-child { border-bottom: none; }
        .metric .label { color: #6c757d; font-weight: 500; }
        .metric .value { font-weight: 600; font-size: 1.1em; }
        .metric .status { padding: 4px 12px; border-radius: 20px; font-size: 0.9em; font-weight: 500; }
        .status.running { background: #d4edda; color: #155724; }
        .status.stopped { background: #f8d7da; color: #721c24; }
        .status.warning { background: #fff3cd; color: #856404; }
        .progress-bar { width: 100%; height: 8px; background: #e9ecef; border-radius: 4px; overflow: hidden; margin: 10px 0; }
        .progress-fill { height: 100%; background: linear-gradient(90deg, #28a745, #20c997); transition: width 0.3s; }
        .actions { text-align: center; margin-top: 30px; }
        .btn { display: inline-block; padding: 12px 24px; margin: 0 10px; background: #007bff; color: white; text-decoration: none; border-radius: 8px; transition: all 0.3s; font-weight: 500; }
        .btn:hover { background: #0056b3; transform: translateY(-2px); }
        .btn.secondary { background: #6c757d; }
        .btn.secondary:hover { background: #545b62; }
        .btn.success { background: #28a745; }
        .btn.success:hover { background: #1e7e34; }
        .btn.warning { background: #ffc107; color: #212529; }
        .btn.warning:hover { background: #e0a800; }
        .footer { text-align: center; margin-top: 40px; color: #6c757d; }
        .refresh-time { text-align: right; margin-bottom: 20px; color: #6c757d; font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🚀 BoomDNS 系统总览</h1>
        <p>智能 DNS 分流服务 - 实时监控与管理</p>
    </div>
    
    <div class="container">
        <div class="refresh-time">最后更新: <span id="refresh-time">加载中...</span></div>
        
        <div class="overview-grid">
            <!-- 系统资源总览 -->
            <div class="card">
                <h3><span class="icon">💻</span>系统资源总览</h3>
                <div class="metric">
                    <span class="label">CPU 使用率:</span>
                    <span class="value" id="cpu-usage">--</span>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill" id="cpu-bar" style="width: 0%"></div>
                </div>
                <div class="metric">
                    <span class="label">内存使用率:</span>
                    <span class="value" id="memory-usage">--</span>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill" id="memory-bar" style="width: 0%"></div>
                </div>
                <div class="metric">
                    <span class="label">磁盘使用率:</span>
                    <span class="value" id="disk-usage">--</span>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill" id="disk-bar" style="width: 0%"></div>
                </div>
                <div class="metric">
                    <span class="label">网络 I/O:</span>
                    <span class="value" id="network-io">--</span>
                </div>
                <div class="actions">
                    <a href="/system" class="btn secondary">📊 详细监控</a>
                </div>
            </div>
            
            <!-- DNS 解析缓存总览 -->
            <div class="card">
                <h3><span class="icon">🌐</span>DNS 解析缓存总览</h3>
                <div class="metric">
                    <span class="label">DNS 服务状态:</span>
                    <span class="status running" id="dns-status">运行中</span>
                </div>
                <div class="metric">
                    <span class="label">缓存命中率:</span>
                    <span class="value" id="cache-hit-rate">--</span>
                </div>
                <div class="metric">
                    <span class="label">缓存条目数:</span>
                    <span class="value" id="cache-entries">--</span>
                </div>
                <div class="metric">
                    <span class="label">查询总数:</span>
                    <span class="value" id="total-queries">--</span>
                </div>
                <div class="metric">
                    <span class="label">平均响应时间:</span>
                    <span class="value" id="avg-response-time">--</span>
                </div>
                <div class="actions">
                    <a href="/dns" class="btn success">📋 DNS 详情</a>
                </div>
            </div>
            
            <!-- 代理运行情况总览 -->
            <div class="card">
                <h3><span class="icon">🔗</span>代理运行情况总览</h3>
                <div class="metric">
                    <span class="label">代理服务状态:</span>
                    <span class="status" id="proxy-status">未启用</span>
                </div>
                <div class="metric">
                    <span class="label">活跃节点数:</span>
                    <span class="value" id="active-nodes">--</span>
                </div>
                <div class="metric">
                    <span class="label">代理组数量:</span>
                    <span class="value" id="proxy-groups">--</span>
                </div>
                <div class="metric">
                    <span class="label">流量统计:</span>
                    <span class="value" id="traffic-stats">--</span>
                </div>
                <div class="metric">
                    <span class="label">健康检查:</span>
                    <span class="value" id="health-status">--</span>
                </div>
                <div class="actions">
                    <a href="/proxy" class="btn warning">⚙️ 代理管理</a>
                </div>
            </div>
            
            <!-- 规则与订阅总览 -->
            <div class="card">
                <h3><span class="icon">📋</span>规则与订阅总览</h3>
                <div class="metric">
                    <span class="label">总规则数:</span>
                    <span class="value" id="total-rules">--</span>
                </div>
                <div class="metric">
                    <span class="label">中国域名:</span>
                    <span class="value" id="china-rules">--</span>
                </div>
                <div class="metric">
                    <span class="label">GFW 域名:</span>
                    <span class="value" id="gfw-rules">--</span>
                </div>
                <div class="metric">
                    <span class="label">广告域名:</span>
                    <span class="value" id="ads-rules">--</span>
                </div>
                <div class="metric">
                    <span class="label">订阅源:</span>
                    <span class="value" id="subscription-sources">--</span>
                </div>
                <div class="actions">
                    <a href="/rules" class="btn">📝 规则管理</a>
                </div>
            </div>
        </div>
        
        <div class="actions">
            <a href="/api/status" class="btn secondary">📊 API 状态</a>
            <a href="/metrics" class="btn secondary">📈 Prometheus 指标</a>
            <button onclick="refreshData()" class="btn success">🔄 刷新数据</button>
        </div>
        
        <div class="footer">
            <p>BoomDNS v1.0.0 | 智能 DNS 分流服务 | 实时监控与管理</p>
        </div>
    </div>
    
    <script>
        // 加载实时数据
        async function loadMetrics() {
            try {
                const response = await fetch('/api/status');
                const data = await response.json();
                
                if (data.success) {
                    // 更新 DNS 相关数据
                    document.getElementById('cache-entries').textContent = data.data.cache_count || 0;
                    document.getElementById('total-queries').textContent = data.data.logs_count || 0;
                    document.getElementById('total-rules').textContent = data.data.rules_count || 0;
                    
                    // 更新代理状态
                    updateProxyStatus();
                    
                    // 更新系统资源
                    updateSystemResources();
                    
                    // 更新时间
                    document.getElementById('refresh-time').textContent = new Date().toLocaleString('zh-CN');
                }
            } catch (error) {
                console.log('加载指标失败:', error);
            }
        }
        
        // 更新代理状态
        function updateProxyStatus() {
            // 这里可以从 API 获取实际数据
            document.getElementById('active-nodes').textContent = '3';
            document.getElementById('proxy-groups').textContent = '2';
            document.getElementById('traffic-stats').textContent = '1.2 MB';
            document.getElementById('health-status').textContent = '正常';
        }
        
        // 更新系统资源
        function updateSystemResources() {
            // 模拟系统资源数据，实际应该从 API 获取
            const cpuUsage = Math.floor(Math.random() * 30) + 10;
            const memoryUsage = Math.floor(Math.random() * 40) + 20;
            const diskUsage = Math.floor(Math.random() * 20) + 10;
            
            document.getElementById('cpu-usage').textContent = cpuUsage + '%';
            document.getElementById('memory-usage').textContent = memoryUsage + '%';
            document.getElementById('disk-usage').textContent = diskUsage + '%';
            document.getElementById('network-io').textContent = '2.1 MB/s';
            
            document.getElementById('cpu-bar').style.width = cpuUsage + '%';
            document.getElementById('memory-bar').style.width = memoryUsage + '%';
            document.getElementById('disk-bar').style.width = diskUsage + '%';
        }
        
        // 刷新数据
        function refreshData() {
            loadMetrics();
        }
        
        // 页面加载完成后获取数据
        document.addEventListener('DOMContentLoaded', loadMetrics);
        
        // 每30秒刷新一次数据
        setInterval(loadMetrics, 30000);
    </script>
</body>
</html>`
		w.Write([]byte(html))
	})

	// API 状态端点
	r.Get("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// 获取基本状态信息
		status := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"service":     "BoomDNS",
				"version":     "1.0.0",
				"status":      "running",
				"dns_port":    cfg.ListenDNS,
				"http_port":   cfg.ListenHTTP,
				"uptime":      time.Since(time.Now()).String(),
				"cache_count": 0,    // 这里可以从 server 获取实际数据
				"logs_count":  1000, // 这里可以从 server 获取实际数据
				"rules_count": 28,   // 这里可以从 server 获取实际数据
				"timestamp":   time.Now().Unix(),
			},
		}

		json.NewEncoder(w).Encode(status)
	})

	// API 规则端点
	r.Get("/api/rules", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		rules := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"china": []string{"baidu.com", "qq.com", "taobao.com", "alibaba.com", "tencent.com", "jd.com", "163.com", "126.com", "sina.com.cn", "sohu.com"},
				"gfw":   []string{"google.com", "youtube.com", "facebook.com", "twitter.com", "instagram.com", "github.com", "stackoverflow.com", "reddit.com", "wikipedia.org", "netflix.com"},
				"ads":   []string{"doubleclick.net", "googlesyndication.com", "googleadservices.com", "adnxs.com", "facebook.com", "adsystem.com", "adtech.com", "advertising.com", "adtechus.com", "adtech.de"},
			},
		}

		json.NewEncoder(w).Encode(rules)
	})

	// API 缓存端点
	r.Get("/api/cache", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		cache := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"enabled":     true,
				"entries":     0,
				"max_entries": 10000,
				"ttl":         3600,
			},
		}

		json.NewEncoder(w).Encode(cache)
	})

	// DNS 详情页面
	r.Get("/dns", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DNS 详情 - BoomDNS</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8f9fa; }
        .header { background: linear-gradient(135deg, #28a745 0%, #20c997 100%); color: white; padding: 20px 0; }
        .header h1 { text-align: center; font-size: 2.5em; margin-bottom: 10px; }
        .header p { text-align: center; opacity: 0.9; }
        .container { max-width: 1400px; margin: 0 auto; padding: 20px; }
        .back-btn { display: inline-block; padding: 10px 20px; background: rgba(255,255,255,0.2); color: white; text-decoration: none; border-radius: 6px; margin-bottom: 20px; }
        .back-btn:hover { background: rgba(255,255,255,0.3); }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .stat-card { background: white; border-radius: 12px; padding: 20px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); text-align: center; }
        .stat-card .number { font-size: 2.5em; font-weight: bold; color: #28a745; margin-bottom: 10px; }
        .stat-card .label { color: #6c757d; font-size: 1.1em; }
        .section { background: white; border-radius: 12px; padding: 25px; margin-bottom: 30px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); }
        .section h3 { color: #2c3e50; margin-bottom: 20px; font-size: 1.5em; border-bottom: 2px solid #e9ecef; padding-bottom: 10px; }
        .table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        .table th, .table td { padding: 12px; text-align: left; border-bottom: 1px solid #e9ecef; }
        .table th { background: #f8f9fa; font-weight: 600; color: #495057; }
        .table tr:hover { background: #f8f9fa; }
        .status-badge { padding: 4px 12px; border-radius: 20px; font-size: 0.9em; font-weight: 500; }
        .status-success { background: #d4edda; color: #155724; }
        .status-warning { background: #fff3cd; color: #856404; }
        .status-error { background: #f8d7da; color: #721c24; }
        .chart-container { height: 300px; margin: 20px 0; background: #f8f9fa; border-radius: 8px; display: flex; align-items: center; justify-content: center; color: #6c757d; }
        .btn { display: inline-block; padding: 10px 20px; margin: 5px; background: #007bff; color: white; text-decoration: none; border-radius: 6px; transition: all 0.3s; }
        .btn:hover { background: #0056b3; transform: translateY(-2px); }
        .btn.success { background: #28a745; }
        .btn.warning { background: #ffc107; color: #212529; }
        .btn.danger { background: #dc3545; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🌐 DNS 详情</h1>
        <p>DNS 解析、缓存、查询日志详细监控</p>
    </div>
    
    <div class="container">
        <a href="/" class="back-btn">← 返回总览</a>
        
        <!-- 关键指标 -->
        <div class="stats-grid">
            <div class="stat-card">
                <div class="number" id="total-queries">0</div>
                <div class="label">总查询数</div>
            </div>
            <div class="stat-card">
                <div class="number" id="cache-hits">0</div>
                <div class="label">缓存命中</div>
            </div>
            <div class="stat-card">
                <div class="number" id="cache-miss">0</div>
                <div class="label">缓存未命中</div>
            </div>
            <div class="stat-card">
                <div class="number" id="avg-response">0ms</div>
                <div class="label">平均响应时间</div>
            </div>
        </div>
        
        <!-- 缓存状态 -->
        <div class="section">
            <h3>💾 缓存状态</h3>
            <div class="stats-grid">
                <div class="stat-card">
                    <div class="number" id="cache-entries">0</div>
                    <div class="label">缓存条目</div>
                </div>
                <div class="stat-card">
                    <div class="number" id="cache-size">0 MB</div>
                    <div class="label">缓存大小</div>
                </div>
                <div class="stat-card">
                    <div class="number" id="cache-ttl">3600s</div>
                    <div class="label">默认 TTL</div>
                </div>
                <div class="stat-card">
                    <div class="number" id="hit-rate">0%</div>
                    <div class="label">命中率</div>
                </div>
            </div>
            <div class="chart-container">
                📊 缓存命中率趋势图 (需要集成图表库)
            </div>
        </div>
        
        <!-- 上游 DNS 状态 -->
        <div class="section">
            <h3>🔗 上游 DNS 状态</h3>
            <table class="table">
                <thead>
                    <tr>
                        <th>类型</th>
                        <th>服务器</th>
                        <th>状态</th>
                        <th>响应时间</th>
                        <th>成功率</th>
                        <th>操作</th>
                    </tr>
                </thead>
                <tbody id="upstream-tbody">
                    <tr>
                        <td>中国</td>
                        <td>223.5.5.5:53</td>
                        <td><span class="status-badge status-success">正常</span></td>
                        <td>15ms</td>
                        <td>99.8%</td>
                        <td><button class="btn btn-warning">测试</button></td>
                    </tr>
                    <tr>
                        <td>国际</td>
                        <td>8.8.8.8:53</td>
                        <td><span class="status-badge status-success">正常</span></td>
                        <td>45ms</td>
                        <td>98.5%</td>
                        <td><button class="btn btn-warning">测试</button></td>
                    </tr>
                </tbody>
            </table>
        </div>
        
        <!-- 查询日志 -->
        <div class="section">
            <h3>📝 最近查询日志</h3>
            <div style="margin-bottom: 15px;">
                <button class="btn" onclick="refreshLogs()">🔄 刷新</button>
                <button class="btn btn-success" onclick="exportLogs()">📥 导出</button>
                <button class="btn btn-danger" onclick="clearLogs()">🗑️ 清空</button>
            </div>
            <table class="table">
                <thead>
                    <tr>
                        <th>时间</th>
                        <th>域名</th>
                        <th>类型</th>
                        <th>客户端 IP</th>
                        <th>响应时间</th>
                        <th>上游</th>
                        <th>缓存</th>
                    </tr>
                </thead>
                <tbody id="logs-tbody">
                    <!-- 动态加载 -->
                </tbody>
            </table>
        </div>
        
        <!-- 性能图表 -->
        <div class="section">
            <h3>📈 性能监控</h3>
            <div class="chart-container">
                📊 查询响应时间分布图 (需要集成图表库)
            </div>
        </div>
    </div>
    
    <script>
        // 加载 DNS 数据
        async function loadDNSData() {
            try {
                const response = await fetch('/api/status');
                const data = await response.json();
                
                if (data.success) {
                    document.getElementById('total-queries').textContent = data.data.logs_count || 0;
                    document.getElementById('cache-entries').textContent = data.data.cache_count || 0;
                    
                    // 模拟其他数据
                    document.getElementById('cache-hits').textContent = Math.floor(data.data.logs_count * 0.8) || 0;
                    document.getElementById('cache-miss').textContent = Math.floor(data.data.logs_count * 0.2) || 0;
                    document.getElementById('avg-response').textContent = '25ms';
                    document.getElementById('cache-size').textContent = '2.1 MB';
                    document.getElementById('hit-rate').textContent = '80%';
                }
            } catch (error) {
                console.log('加载 DNS 数据失败:', error);
            }
        }
        
        // 刷新日志
        function refreshLogs() {
            loadDNSData();
        }
        
        // 导出日志
        function exportLogs() {
            alert('导出功能开发中...');
        }
        
        // 清空日志
        function clearLogs() {
            if (confirm('确定要清空所有日志吗？')) {
                alert('清空功能开发中...');
            }
        }
        
        // 页面加载完成后获取数据
        document.addEventListener('DOMContentLoaded', loadDNSData);
        
        // 每30秒刷新一次数据
        setInterval(loadDNSData, 30000);
    </script>
</body>
</html>`
		w.Write([]byte(html))
	})

	// 代理管理页面
	r.Get("/proxy", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>代理管理 - BoomDNS</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8f9fa; }
        .header { background: linear-gradient(135deg, #ffc107 0%, #fd7e14 100%); color: white; padding: 20px 0; }
        .header h1 { text-align: center; font-size: 2.5em; margin-bottom: 10px; }
        .header p { text-align: center; opacity: 0.9; }
        .container { max-width: 1400px; margin: 0 auto; padding: 20px; }
        .back-btn { display: inline-block; padding: 10px 20px; background: rgba(255,255,255,0.2); color: white; text-decoration: none; border-radius: 6px; margin-bottom: 20px; }
        .back-btn:hover { background: rgba(255,255,255,0.3); }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .stat-card { background: white; border-radius: 12px; padding: 20px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); text-align: center; }
        .stat-card .number { font-size: 2em; font-weight: bold; color: #fd7e14; margin-bottom: 10px; }
        .stat-card .label { color: #6c757d; font-size: 1em; }
        .section { background: white; border-radius: 12px; padding: 25px; margin-bottom: 30px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); }
        .section h3 { color: #2c3e50; margin-bottom: 20px; font-size: 1.5em; border-bottom: 2px solid #e9ecef; padding-bottom: 10px; }
        .table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        .table th, .table td { padding: 12px; text-align: left; border-bottom: 1px solid #e9ecef; }
        .table th { background: #f8f9fa; font-weight: 600; color: #495057; }
        .table tr:hover { background: #f8f9fa; }
        .status-badge { padding: 4px 12px; border-radius: 20px; font-size: 0.9em; font-weight: 500; }
        .status-success { background: #d4edda; color: #155724; }
        .status-warning { background: #fff3cd; color: #856404; }
        .status-error { background: #f8d7da; color: #721c24; }
        .btn { display: inline-block; padding: 8px 16px; margin: 2px; background: #007bff; color: white; text-decoration: none; border-radius: 6px; transition: all 0.3s; border: none; cursor: pointer; font-size: 0.9em; }
        .btn:hover { background: #0056b3; transform: translateY(-1px); }
        .btn.success { background: #28a745; }
        .btn.warning { background: #ffc107; color: #212529; }
        .btn.danger { background: #dc3545; }
        .btn.small { padding: 6px 12px; font-size: 0.8em; }
        .tabs { display: flex; border-bottom: 2px solid #e9ecef; margin-bottom: 20px; }
        .tab { padding: 12px 24px; background: none; border: none; cursor: pointer; color: #6c757d; font-weight: 500; transition: all 0.3s; }
        .tab.active { color: #fd7e14; border-bottom: 2px solid #fd7e14; }
        .tab:hover { color: #fd7e14; }
        .tab-content { display: none; }
        .tab-content.active { display: block; }
        .form-group { margin-bottom: 15px; }
        .form-group label { display: block; margin-bottom: 5px; font-weight: 500; color: #495057; }
        .form-group input, .form-group select { width: 100%; padding: 8px 12px; border: 1px solid #ced4da; border-radius: 4px; font-size: 14px; }
        .form-group input:focus, .form-group select:focus { outline: none; border-color: #fd7e14; box-shadow: 0 0 0 2px rgba(253, 126, 20, 0.25); }
        .modal { display: none; position: fixed; z-index: 1000; left: 0; top: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); }
        .modal-content { background: white; margin: 5% auto; padding: 20px; border-radius: 12px; width: 90%; max-width: 600px; }
        .modal-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
        .close { font-size: 28px; font-weight: bold; cursor: pointer; color: #aaa; }
        .close:hover { color: #000; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🔗 代理管理</h1>
        <p>代理节点、组、规则配置与管理</p>
    </div>
    
    <div class="container">
        <a href="/" class="back-btn">← 返回总览</a>
        
        <!-- 代理状态概览 -->
        <div class="stats-grid">
            <div class="stat-card">
                <div class="number" id="total-nodes">3</div>
                <div class="label">总节点数</div>
            </div>
            <div class="stat-card">
                <div class="number" id="active-nodes">3</div>
                <div class="label">活跃节点</div>
            </div>
            <div class="stat-card">
                <div class="number" id="total-groups">2</div>
                <div class="label">代理组</div>
            </div>
            <div class="stat-card">
                <div class="number" id="total-rules">4</div>
                <div class="label">代理规则</div>
            </div>
            <div class="stat-card">
                <div class="number" id="total-traffic">1.2 MB</div>
                <div class="label">总流量</div>
            </div>
            <div class="stat-card">
                <div class="number" id="health-rate">100%</div>
                <div class="label">健康率</div>
            </div>
        </div>
        
        <!-- 标签页 -->
        <div class="tabs">
            <button class="tab active" onclick="showTab('nodes')">🖥️ 代理节点</button>
            <button class="tab" onclick="showTab('groups')">👥 代理组</button>
            <button class="tab" onclick="showTab('rules')">📋 代理规则</button>
            <button class="tab" onclick="showTab('traffic')">📊 流量统计</button>
        </div>
        
        <!-- 代理节点 -->
        <div id="nodes" class="tab-content active">
            <div class="section">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
                    <h3>代理节点管理</h3>
                    <button class="btn success" onclick="showAddNodeModal()">➕ 添加节点</button>
                </div>
                <table class="table">
                    <thead>
                        <tr>
                            <th>名称</th>
                            <th>协议</th>
                            <th>地址</th>
                            <th>端口</th>
                            <th>状态</th>
                            <th>延迟</th>
                            <th>权重</th>
                            <th>操作</th>
                        </tr>
                    </thead>
                    <tbody id="nodes-tbody">
                        <tr>
                            <td>Hysteria2-香港</td>
                            <td>hysteria2</td>
                            <td>hk.example.com</td>
                            <td>443</td>
                            <td><span class="status-badge status-success">正常</span></td>
                            <td>45ms</td>
                            <td>100</td>
                            <td>
                                <button class="btn btn-warning btn-small" onclick="testNode(1)">测试</button>
                                <button class="btn btn-small" onclick="editNode(1)">编辑</button>
                                <button class="btn btn-danger btn-small" onclick="deleteNode(1)">删除</button>
                            </td>
                        </tr>
                        <tr>
                            <td>SS-香港</td>
                            <td>ss</td>
                            <td>hk-ss.example.com</td>
                            <td>8388</td>
                            <td><span class="status-badge status-success">正常</span></td>
                            <td>52ms</td>
                            <td>80</td>
                            <td>
                                <button class="btn btn-warning btn-small" onclick="testNode(2)">测试</button>
                                <button class="btn btn-small" onclick="editNode(2)">编辑</button>
                                <button class="btn btn-danger btn-small" onclick="deleteNode(2)">删除</button>
                            </td>
                        </tr>
                        <tr>
                            <td>V2Ray-美国</td>
                            <td>v2ray</td>
                            <td>us.example.com</td>
                            <td>443</td>
                            <td><span class="status-badge status-success">正常</span></td>
                            <td>120ms</td>
                            <td>60</td>
                            <td>
                                <button class="btn btn-warning btn-small" onclick="testNode(3)">测试</button>
                                <button class="btn btn-small" onclick="editNode(3)">编辑</button>
                                <button class="btn btn-danger btn-small" onclick="deleteNode(3)">删除</button>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
        
        <!-- 代理组 -->
        <div id="groups" class="tab-content">
            <div class="section">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
                    <h3>代理组管理</h3>
                    <button class="btn success" onclick="showAddGroupModal()">➕ 添加组</button>
                </div>
                <table class="table">
                    <thead>
                        <tr>
                            <th>名称</th>
                            <th>类型</th>
                            <th>策略</th>
                            <th>测试地址</th>
                            <th>状态</th>
                            <th>操作</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr>
                            <td>自动选择</td>
                            <td>url-test</td>
                            <td>latency</td>
                            <td>http://www.google.com</td>
                            <td><span class="status-badge status-success">正常</span></td>
                            <td>
                                <button class="btn btn-small" onclick="editGroup(1)">编辑</button>
                                <button class="btn btn-danger btn-small" onclick="deleteGroup(1)">删除</button>
                            </td>
                        </tr>
                        <tr>
                            <td>故障转移</td>
                            <td>fallback</td>
                            <td>latency</td>
                            <td>http://www.google.com</td>
                            <td><span class="status-badge status-success">正常</span></td>
                            <td>
                                <button class="btn btn-small" onclick="editGroup(2)">编辑</button>
                                <button class="btn btn-danger btn-small" onclick="deleteGroup(2)">删除</button>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
        
        <!-- 代理规则 -->
        <div id="rules" class="tab-content">
            <div class="section">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
                    <h3>代理规则管理</h3>
                    <button class="btn success" onclick="showAddRuleModal()">➕ 添加规则</button>
                </div>
                <table class="table">
                    <thead>
                        <tr>
                            <th>类型</th>
                            <th>值</th>
                            <th>动作</th>
                            <th>代理组</th>
                            <th>优先级</th>
                            <th>状态</th>
                            <th>操作</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr>
                            <td>domain</td>
                            <td>google.com</td>
                            <td>proxy</td>
                            <td>自动选择</td>
                            <td>100</td>
                            <td><span class="status-badge status-success">启用</span></td>
                            <td>
                                <button class="btn btn-small" onclick="editRule(1)">编辑</button>
                                <button class="btn btn-danger btn-small" onclick="deleteRule(1)">删除</button>
                            </td>
                        </tr>
                        <tr>
                            <td>domain</td>
                            <td>baidu.com</td>
                            <td>direct</td>
                            <td>-</td>
                            <td>200</td>
                            <td><span class="status-badge status-success">启用</span></td>
                            <td>
                                <button class="btn btn-small" onclick="editRule(2)">编辑</button>
                                <button class="btn btn-danger btn-small" onclick="deleteRule(2)">删除</button>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
        
        <!-- 流量统计 -->
        <div id="traffic" class="tab-content">
            <div class="section">
                <h3>流量统计</h3>
                <div class="stats-grid">
                    <div class="stat-card">
                        <div class="number">2.1 GB</div>
                        <div class="label">今日流量</div>
                    </div>
                    <div class="stat-card">
                        <div class="number">15.8 GB</div>
                        <div class="label">本月流量</div>
                    </div>
                    <div class="stat-card">
                        <div class="number">1.2 MB/s</div>
                        <div class="label">当前速度</div>
                    </div>
                    <div class="stat-card">
                        <div class="number">1,234</div>
                        <div class="label">连接数</div>
                    </div>
                </div>
                <div style="height: 300px; background: #f8f9fa; border-radius: 8px; display: flex; align-items: center; justify-content: center; color: #6c757d; margin-top: 20px;">
                    📊 流量趋势图 (需要集成图表库)
                </div>
            </div>
        </div>
    </div>
    
    <!-- 添加节点模态框 -->
    <div id="addNodeModal" class="modal">
        <div class="modal-content">
            <div class="modal-header">
                <h3>添加代理节点</h3>
                <span class="close" onclick="closeModal('addNodeModal')">&times;</span>
            </div>
            <form id="addNodeForm">
                <div class="form-group">
                    <label>节点名称</label>
                    <input type="text" name="name" placeholder="例如: Hysteria2-香港" required>
                </div>
                <div class="form-group">
                    <label>协议类型</label>
                    <select name="protocol" required>
                        <option value="hysteria2">Hysteria2</option>
                        <option value="ss">Shadowsocks</option>
                        <option value="v2ray">V2Ray</option>
                        <option value="trojan">Trojan</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>服务器地址</label>
                    <input type="text" name="address" placeholder="例如: hk.example.com" required>
                </div>
                <div class="form-group">
                    <label>端口</label>
                    <input type="number" name="port" placeholder="例如: 443" required>
                </div>
                <div class="form-group">
                    <label>权重</label>
                    <input type="number" name="weight" value="100" min="1" max="1000">
                </div>
                <div style="text-align: right; margin-top: 20px;">
                    <button type="button" class="btn btn-warning" onclick="closeModal('addNodeModal')">取消</button>
                    <button type="submit" class="btn success">添加</button>
                </div>
            </form>
        </div>
    </div>
    
    <script>
        // 显示标签页
        function showTab(tabName) {
            // 隐藏所有标签页内容
            const tabContents = document.querySelectorAll('.tab-content');
            tabContents.forEach(content => content.classList.remove('active'));
            
            // 移除所有标签页的活跃状态
            const tabs = document.querySelectorAll('.tab');
            tabs.forEach(tab => tab.classList.remove('active'));
            
            // 显示选中的标签页
            document.getElementById(tabName).classList.add('active');
            event.target.classList.add('active');
        }
        
        // 显示模态框
        function showAddNodeModal() {
            document.getElementById('addNodeModal').style.display = 'block';
        }
        
        // 关闭模态框
        function closeModal(modalId) {
            document.getElementById(modalId).style.display = 'none';
        }
        
        // 测试节点
        function testNode(nodeId) {
            alert('测试节点 ' + nodeId + ' 功能开发中...');
        }
        
        // 编辑节点
        function editNode(nodeId) {
            alert('编辑节点 ' + nodeId + ' 功能开发中...');
        }
        
        // 删除节点
        function deleteNode(nodeId) {
            if (confirm('确定要删除这个节点吗？')) {
                alert('删除节点 ' + nodeId + ' 功能开发中...');
            }
        }
        
        // 添加组模态框
        function showAddGroupModal() {
            alert('添加组功能开发中...');
        }
        
        // 编辑组
        function editGroup(groupId) {
            alert('编辑组 ' + groupId + ' 功能开发中...');
        }
        
        // 删除组
        function deleteGroup(groupId) {
            if (confirm('确定要删除这个组吗？')) {
                alert('删除组 ' + groupId + ' 功能开发中...');
            }
        }
        
        // 添加规则模态框
        function showAddRuleModal() {
            alert('添加规则功能开发中...');
        }
        
        // 编辑规则
        function editRule(ruleId) {
            alert('编辑规则 ' + ruleId + ' 功能开发中...');
        }
        
        // 删除规则
        function deleteRule(ruleId) {
            if (confirm('确定要删除这个规则吗？')) {
                alert('删除规则 ' + ruleId + ' 功能开发中...');
            }
        }
        
        // 表单提交
        document.getElementById('addNodeForm').addEventListener('submit', function(e) {
            e.preventDefault();
            alert('添加节点功能开发中...');
            closeModal('addNodeModal');
        });
        
        // 点击模态框外部关闭
        window.onclick = function(event) {
            if (event.target.classList.contains('modal')) {
                event.target.style.display = 'none';
            }
        }
    </script>
</body>
</html>`
		w.Write([]byte(html))
	})

	// 系统监控页面
	r.Get("/system", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>系统监控 - BoomDNS</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8f9fa; }
        .header { background: linear-gradient(135deg, #6f42c1 0%, #e83e8c 100%); color: white; padding: 20px 0; }
        .header h1 { text-align: center; font-size: 2.5em; margin-bottom: 10px; }
        .header p { text-align: center; opacity: 0.9; }
        .container { max-width: 1400px; margin: 0 auto; padding: 20px; }
        .back-btn { display: inline-block; padding: 10px 20px; background: rgba(255,255,255,0.2); color: white; text-decoration: none; border-radius: 6px; margin-bottom: 20px; }
        .back-btn:hover { background: rgba(255,255,255,0.3); }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .stat-card { background: white; border-radius: 12px; padding: 20px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); text-align: center; }
        .stat-card .number { font-size: 2.5em; font-weight: bold; color: #6f42c1; margin-bottom: 10px; }
        .stat-card .label { color: #6c757d; font-size: 1.1em; }
        .section { background: white; border-radius: 12px; padding: 25px; margin-bottom: 30px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); }
        .section h3 { color: #2c3e50; margin-bottom: 20px; font-size: 1.5em; border-bottom: 2px solid #e9ecef; padding-bottom: 10px; }
        .progress-container { margin: 20px 0; }
        .progress-label { display: flex; justify-content: space-between; margin-bottom: 8px; font-weight: 500; }
        .progress-bar { width: 100%; height: 12px; background: #e9ecef; border-radius: 6px; overflow: hidden; }
        .progress-fill { height: 100%; transition: width 0.3s; border-radius: 6px; }
        .progress-cpu { background: linear-gradient(90deg, #28a745, #20c997); }
        .progress-memory { background: linear-gradient(90deg, #007bff, #6610f2); }
        .progress-disk { background: linear-gradient(90deg, #ffc107, #fd7e14); }
        .progress-network { background: linear-gradient(90deg, #e83e8c, #6f42c1); }
        .refresh-btn { position: fixed; bottom: 30px; right: 30px; width: 60px; height: 60px; border-radius: 50%; background: #6f42c1; color: white; border: none; font-size: 24px; cursor: pointer; box-shadow: 0 4px 20px rgba(111, 66, 193, 0.3); transition: all 0.3s; }
        .refresh-btn:hover { transform: scale(1.1); box-shadow: 0 6px 25px rgba(111, 66, 193, 0.4); }
    </style>
</head>
<body>
    <div class="header">
        <h1>💻 系统监控</h1>
        <p>实时系统资源监控与性能分析</p>
    </div>
    
    <div class="container">
        <a href="/" class="back-btn">← 返回总览</a>
        
        <!-- 系统状态概览 -->
        <div class="stats-grid">
            <div class="stat-card">
                <div class="number" id="cpu-usage">--</div>
                <div class="label">CPU 使用率</div>
            </div>
            <div class="stat-card">
                <div class="number" id="memory-usage">--</div>
                <div class="label">内存使用率</div>
            </div>
            <div class="stat-card">
                <div class="number" id="disk-usage">--</div>
                <div class="label">磁盘使用率</div>
            </div>
            <div class="stat-card">
                <div class="number" id="network-io">--</div>
                <div class="label">网络 I/O</div>
            </div>
        </div>
        
        <!-- 系统资源监控 -->
        <div class="section">
            <h3>📊 系统资源监控</h3>
            
            <!-- CPU 监控 -->
            <div class="progress-container">
                <div class="progress-label">
                    <span>CPU 使用率</span>
                    <span id="cpu-text">--</span>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill progress-cpu" id="cpu-bar" style="width: 0%"></div>
                </div>
            </div>
            
            <!-- 内存监控 -->
            <div class="progress-container">
                <div class="progress-label">
                    <span>内存使用率</span>
                    <span id="memory-text">--</span>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill progress-memory" id="memory-bar" style="width: 0%"></div>
                </div>
            </div>
            
            <!-- 磁盘监控 -->
            <div class="progress-container">
                <div class="progress-label">
                    <span>磁盘使用率</span>
                    <span id="disk-text">--</span>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill progress-disk" id="disk-bar" style="width: 0%"></div>
                </div>
            </div>
        </div>
        
        <!-- 系统信息 -->
        <div class="section">
            <h3>ℹ️ 系统信息</h3>
            <div class="stats-grid">
                <div class="stat-card">
                    <div class="number" id="os-info">--</div>
                    <div class="label">操作系统</div>
                </div>
                <div class="stat-card">
                    <div class="number" id="kernel-version">--</div>
                    <div class="label">内核版本</div>
                </div>
                <div class="stat-card">
                    <div class="number" id="hostname">--</div>
                    <div class="label">主机名</div>
                </div>
                <div class="stat-card">
                    <div class="number" id="cpu-cores">--</div>
                    <div class="label">CPU 核心数</div>
                </div>
            </div>
        </div>
    </div>
    
    <!-- 刷新按钮 -->
    <button class="refresh-btn" onclick="refreshSystemData()" title="刷新数据">🔄</button>
    
    <script>
        // 模拟系统数据
        function generateSystemData() {
            const cpuUsage = Math.floor(Math.random() * 40) + 10;
            const memoryUsage = Math.floor(Math.random() * 50) + 20;
            const diskUsage = Math.floor(Math.random() * 30) + 10;
            
            // 更新统计卡片
            document.getElementById('cpu-usage').textContent = cpuUsage + '%';
            document.getElementById('memory-usage').textContent = memoryUsage + '%';
            document.getElementById('disk-usage').textContent = diskUsage + '%';
            document.getElementById('network-io').textContent = (Math.random() * 10 + 1).toFixed(1) + ' MB/s';
            
            // 更新进度条
            document.getElementById('cpu-bar').style.width = cpuUsage + '%';
            document.getElementById('memory-bar').style.width = memoryUsage + '%';
            document.getElementById('disk-bar').style.width = diskUsage + '%';
            
            // 更新进度条文本
            document.getElementById('cpu-text').textContent = cpuUsage + '%';
            document.getElementById('memory-text').textContent = memoryUsage + '%';
            document.getElementById('disk-text').textContent = diskUsage + '%';
            
            // 更新系统信息
            document.getElementById('os-info').textContent = 'macOS 14.6';
            document.getElementById('kernel-version').textContent = 'Darwin 23.6.0';
            document.getElementById('hostname').textContent = 'winspan-mac';
            document.getElementById('cpu-cores').textContent = '8 核';
        }
        
        // 刷新系统数据
        function refreshSystemData() {
            generateSystemData();
            // 添加刷新动画
            const btn = event.target;
            btn.style.transform = 'rotate(360deg)';
            setTimeout(() => btn.style.transform = 'rotate(0deg)', 500);
        }
        
        // 页面加载完成后获取数据
        document.addEventListener('DOMContentLoaded', generateSystemData);
        
        // 每10秒刷新一次数据
        setInterval(generateSystemData, 10000);
    </script>
</body>
</html>`
		w.Write([]byte(html))
	})

	// 规则管理页面
	r.Get("/rules", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>规则管理 - BoomDNS</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8f9fa; }
        .header { background: linear-gradient(135deg, #17a2b8 0%, #20c997 100%); color: white; padding: 20px 0; }
        .header h1 { text-align: center; font-size: 2.5em; margin-bottom: 10px; }
        .header p { text-align: center; opacity: 0.9; }
        .container { max-width: 1400px; margin: 0 auto; padding: 20px; }
        .back-btn { display: inline-block; padding: 10px 20px; background: rgba(255,255,255,0.2); color: white; text-decoration: none; border-radius: 6px; margin-bottom: 20px; }
        .back-btn:hover { background: rgba(255,255,255,0.3); }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .stat-card { background: white; border-radius: 12px; padding: 20px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); text-align: center; }
        .stat-card .number { font-size: 2em; font-weight: bold; color: #17a2b8; margin-bottom: 10px; }
        .stat-card .label { color: #6c757d; font-size: 1em; }
        .section { background: white; border-radius: 12px; padding: 25px; margin-bottom: 30px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); }
        .section h3 { color: #2c3e50; margin-bottom: 20px; font-size: 1.5em; border-bottom: 2px solid #e9ecef; padding-bottom: 10px; }
        .tabs { display: flex; border-bottom: 2px solid #e9ecef; margin-bottom: 20px; }
        .tab { padding: 12px 24px; background: none; border: none; cursor: pointer; color: #6c757d; font-weight: 500; transition: all 0.3s; }
        .tab.active { color: #17a2b8; border-bottom: 2px solid #17a2b8; }
        .tab:hover { color: #17a2b8; }
        .tab-content { display: none; }
        .tab-content.active { display: block; }
        .table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        .table th, .table td { padding: 12px; text-align: left; border-bottom: 1px solid #e9ecef; }
        .table th { background: #f8f9fa; font-weight: 600; color: #495057; }
        .table tr:hover { background: #f8f9fa; }
        .btn { display: inline-block; padding: 8px 16px; margin: 2px; background: #007bff; color: white; text-decoration: none; border-radius: 6px; transition: all 0.3s; border: none; cursor: pointer; font-size: 0.9em; }
        .btn:hover { background: #0056b3; transform: translateY(-1px); }
        .btn.success { background: #28a745; }
        .btn.warning { background: #ffc107; color: #212529; }
        .btn.danger { background: #dc3545; }
        .btn.small { padding: 6px 12px; font-size: 0.8em; }
        .rule-type { padding: 4px 8px; border-radius: 12px; font-size: 0.8em; font-weight: 500; }
        .rule-type.china { background: #d4edda; color: #155724; }
        .rule-type.gfw { background: #f8d7da; color: #721c24; }
        .rule-type.ads { background: #fff3cd; color: #856404; }
        .rule-type.custom { background: #d1ecf1; color: #0c5460; }
    </style>
</head>
<body>
    <div class="header">
        <h1>📋 规则管理</h1>
        <p>DNS 规则、订阅源、分流策略管理</p>
    </div>
    
    <div class="container">
        <a href="/" class="back-btn">← 返回总览</a>
        
        <!-- 规则统计概览 -->
        <div class="stats-grid">
            <div class="stat-card">
                <div class="number" id="total-rules">28</div>
                <div class="label">总规则数</div>
            </div>
            <div class="stat-card">
                <div class="number" id="china-rules">10</div>
                <div class="label">中国域名</div>
            </div>
            <div class="stat-card">
                <div class="number" id="gfw-rules">10</div>
                <div class="label">GFW 域名</div>
            </div>
            <div class="stat-card">
                <div class="number" id="ads-rules">8</div>
                <div class="label">广告域名</div>
            </div>
            <div class="stat-card">
                <div class="number" id="subscription-sources">3</div>
                <div class="label">订阅源</div>
            </div>
            <div class="stat-card">
                <div class="number" id="last-update">2h</div>
                <div class="label">最后更新</div>
            </div>
        </div>
        
        <!-- 标签页 -->
        <div class="tabs">
            <button class="tab active" onclick="showTab('rules')">📋 DNS 规则</button>
            <button class="tab" onclick="showTab('subscriptions')">📡 订阅管理</button>
            <button class="tab" onclick="showTab('import')">📥 导入导出</button>
            <button class="tab" onclick="showTab('settings')">⚙️ 规则设置</button>
        </div>
        
        <!-- DNS 规则 -->
        <div id="rules" class="tab-content active">
            <div class="section">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
                    <h3>DNS 规则管理</h3>
                    <button class="btn success" onclick="showAddRuleModal()">➕ 添加规则</button>
                </div>
                <table class="table">
                    <thead>
                        <tr>
                            <th>类型</th>
                            <th>域名</th>
                            <th>上游</th>
                            <th>TTL</th>
                            <th>状态</th>
                            <th>操作</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr>
                            <td><span class="rule-type china">中国</span></td>
                            <td>baidu.com</td>
                            <td>中国 DNS</td>
                            <td>3600</td>
                            <td><span style="color: #28a745;">✅ 启用</span></td>
                            <td>
                                <button class="btn btn-small" onclick="editRule(1)">编辑</button>
                                <button class="btn btn-danger btn-small" onclick="deleteRule(1)">删除</button>
                            </td>
                        </tr>
                        <tr>
                            <td><span class="rule-type gfw">GFW</span></td>
                            <td>google.com</td>
                            <td>国际 DNS</td>
                            <td>1800</td>
                            <td><span style="color: #28a745;">✅ 启用</span></td>
                            <td>
                                <button class="btn btn-small" onclick="editRule(2)">编辑</button>
                                <button class="btn btn-danger btn-small" onclick="deleteRule(2)">删除</button>
                            </td>
                        </tr>
                        <tr>
                            <td><span class="rule-type ads">广告</span></td>
                            <td>doubleclick.net</td>
                            <td>AdGuard DNS</td>
                            <td>7200</td>
                            <td><span style="color: #28a745;">✅ 启用</span></td>
                            <td>
                                <button class="btn btn-small" onclick="editRule(3)">编辑</button>
                                <button class="btn btn-danger btn-small" onclick="deleteRule(3)">删除</button>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
        
        <!-- 订阅管理 -->
        <div id="subscriptions" class="tab-content">
            <div class="section">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
                    <h3>订阅源管理</h3>
                    <button class="btn success" onclick="showAddSubscriptionModal()">➕ 添加订阅</button>
                </div>
                <table class="table">
                    <thead>
                        <tr>
                            <th>名称</th>
                            <th>URL</th>
                            <th>类型</th>
                            <th>状态</th>
                            <th>最后更新</th>
                            <th>规则数</th>
                            <th>操作</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr>
                            <td>GFW 域名列表</td>
                            <td>https://raw.githubusercontent.com/gfwlist/gfwlist/master/gfwlist.txt</td>
                            <td>gfwlist</td>
                            <td><span style="color: #28a745;">✅ 正常</span></td>
                            <td>2小时前</td>
                            <td>1,234</td>
                            <td>
                                <button class="btn btn-warning btn-small" onclick="testSubscription(1)">测试</button>
                                <button class="btn btn-small" onclick="editSubscription(1)">编辑</button>
                                <button class="btn btn-danger btn-small" onclick="deleteSubscription(1)">删除</button>
                            </td>
                        </tr>
                        <tr>
                            <td>广告域名列表</td>
                            <td>https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts</td>
                            <td>hosts</td>
                            <td><span style="color: #28a745;">✅ 正常</span></td>
                            <td>1小时前</td>
                            <td>567</td>
                            <td>
                                <button class="btn btn-warning btn-small" onclick="testSubscription(2)">测试</button>
                                <button class="btn btn-small" onclick="editSubscription(2)">编辑</button>
                                <button class="btn btn-danger btn-small" onclick="deleteSubscription(2)">删除</button>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
        
        <!-- 导入导出 -->
        <div id="import" class="tab-content">
            <div class="section">
                <h3>📥 导入导出规则</h3>
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 30px;">
                    <div>
                        <h4>📤 导出规则</h4>
                        <p style="color: #6c757d; margin-bottom: 20px;">将当前规则导出为不同格式</p>
                        <button class="btn success" onclick="exportRules('json')">导出为 JSON</button>
                        <button class="btn success" onclick="exportRules('yaml')">导出为 YAML</button>
                        <button class="btn success" onclick="exportRules('hosts')">导出为 Hosts</button>
                    </div>
                    <div>
                        <h4>📥 导入规则</h4>
                        <p style="color: #6c757d; margin-bottom: 20px;">从文件或 URL 导入规则</p>
                        <button class="btn" onclick="showImportModal()">从文件导入</button>
                        <button class="btn" onclick="showImportUrlModal()">从 URL 导入</button>
                    </div>
                </div>
            </div>
        </div>
        
        <!-- 规则设置 -->
        <div id="settings" class="tab-content">
            <div class="section">
                <h3>⚙️ 规则设置</h3>
                <div class="form-group">
                    <label>自动更新间隔</label>
                    <select id="update-interval">
                        <option value="3600">1 小时</option>
                        <option value="7200" selected>2 小时</option>
                        <option value="21600">6 小时</option>
                        <option value="86400">24 小时</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>规则冲突处理</label>
                    <select id="conflict-resolution">
                        <option value="priority">按优先级</option>
                        <option value="last" selected>最后添加</option>
                        <option value="first">首先添加</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>启用规则验证</label>
                    <input type="checkbox" id="rule-validation" checked>
                </div>
                <button class="btn success" onclick="saveSettings()">保存设置</button>
            </div>
        </div>
    </div>
    
    <script>
        // 显示标签页
        function showTab(tabName) {
            const tabContents = document.querySelectorAll('.tab-content');
            tabContents.forEach(content => content.classList.remove('active'));
            
            const tabs = document.querySelectorAll('.tab');
            tabs.forEach(tab => tab.classList.remove('active'));
            
            document.getElementById(tabName).classList.add('active');
            event.target.classList.add('active');
        }
        
        // 编辑规则
        function editRule(ruleId) {
            alert('编辑规则 ' + ruleId + ' 功能开发中...');
        }
        
        // 删除规则
        function deleteRule(ruleId) {
            if (confirm('确定要删除这个规则吗？')) {
                alert('删除规则 ' + ruleId + ' 功能开发中...');
            }
        }
        
        // 测试订阅
        function testSubscription(subId) {
            alert('测试订阅 ' + subId + ' 功能开发中...');
        }
        
        // 编辑订阅
        function editSubscription(subId) {
            alert('编辑订阅 ' + subId + ' 功能开发中...');
        }
        
        // 删除订阅
        function deleteSubscription(subId) {
            if (confirm('确定要删除这个订阅吗？')) {
                alert('删除订阅 ' + subId + ' 功能开发中...');
            }
        }
        
        // 导出规则
        function exportRules(format) {
            alert('导出为 ' + format + ' 格式功能开发中...');
        }
        
        // 显示导入模态框
        function showImportModal() {
            alert('从文件导入功能开发中...');
        }
        
        // 显示 URL 导入模态框
        function showImportUrlModal() {
            alert('从 URL 导入功能开发中...');
        }
        
        // 保存设置
        function saveSettings() {
            alert('设置保存功能开发中...');
        }
    </script>
</body>
</html>`
		w.Write([]byte(html))
	})

	// 日志分析页面
	r.Get("/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>日志分析 - BoomDNS</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8f9fa; }
        .header { background: linear-gradient(135deg, #fd7e14 0%, #e83e8c 100%); color: white; padding: 20px 0; }
        .header h1 { text-align: center; font-size: 2.5em; margin-bottom: 10px; }
        .header p { text-align: center; opacity: 0.9; }
        .container { max-width: 1400px; margin: 0 auto; padding: 20px; }
        .back-btn { display: inline-block; padding: 10px 20px; background: rgba(255,255,255,0.2); color: white; text-decoration: none; border-radius: 6px; margin-bottom: 20px; }
        .back-btn:hover { background: rgba(255,255,255,0.3); }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .stat-card { background: white; border-radius: 12px; padding: 20px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); text-align: center; }
        .stat-card .number { font-size: 2em; font-weight: bold; color: #fd7e14; margin-bottom: 10px; }
        .stat-card .label { color: #6c757d; font-size: 1em; }
        .section { background: white; border-radius: 12px; padding: 25px; margin-bottom: 30px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); }
        .section h3 { color: #2c3e50; margin-bottom: 20px; font-size: 1.5em; border-bottom: 2px solid #e9ecef; padding-bottom: 10px; }
        .filters { background: #f8f9fa; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .filter-row { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; margin-bottom: 15px; }
        .filter-group { display: flex; flex-direction: column; }
        .filter-group label { font-weight: 500; margin-bottom: 5px; color: #495057; }
        .filter-group input, .filter-group select { padding: 8px 12px; border: 1px solid #ced4da; border-radius: 4px; font-size: 14px; }
        .filter-group input:focus, .filter-group select:focus { outline: none; border-color: #fd7e14; box-shadow: 0 0 0 2px rgba(253, 126, 20, 0.25); }
        .table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        .table th, .table td { padding: 12px; text-align: left; border-bottom: 1px solid #e9ecef; }
        .table th { background: #f8f9fa; font-weight: 600; color: #495057; }
        .table tr:hover { background: #f8f9fa; }
        .btn { display: inline-block; padding: 8px 16px; margin: 2px; background: #007bff; color: white; text-decoration: none; border-radius: 6px; transition: all 0.3s; border: none; cursor: pointer; font-size: 0.9em; }
        .btn:hover { background: #0056b3; transform: translateY(-1px); }
        .btn.success { background: #28a745; }
        .btn.warning { background: #ffc107; color: #212529; }
        .btn.danger { background: #dc3545; }
        .btn.small { padding: 6px 12px; font-size: 0.8em; }
        .pagination { display: flex; justify-content: center; align-items: center; margin-top: 20px; gap: 10px; }
        .pagination button { padding: 8px 12px; border: 1px solid #dee2e6; background: white; color: #495057; cursor: pointer; border-radius: 4px; }
        .pagination button:hover { background: #e9ecef; }
        .pagination button.active { background: #fd7e14; color: white; border-color: #fd7e14; }
        .pagination button:disabled { background: #f8f9fa; color: #6c757d; cursor: not-allowed; }
        .chart-container { height: 300px; margin: 20px 0; background: #f8f9fa; border-radius: 8px; display: flex; align-items: center; justify-content: center; color: #6c757d; border: 2px dashed #dee2e6; }
        .log-level { padding: 4px 8px; border-radius: 12px; font-size: 0.8em; font-weight: 500; }
        .log-level.info { background: #d1ecf1; color: #0c5460; }
        .log-level.warning { background: #fff3cd; color: #856404; }
        .log-level.error { background: #f8d7da; color: #721c24; }
        .log-level.debug { background: #e2e3e5; color: #383d41; }
    </style>
</head>
<body>
    <div class="header">
        <h1>📊 日志分析</h1>
        <p>DNS 查询日志分析与统计报告</p>
    </div>
    
    <div class="container">
        <a href="/" class="back-btn">← 返回总览</a>
        
        <!-- 日志统计概览 -->
        <div class="stats-grid">
            <div class="stat-card">
                <div class="number" id="total-logs">1,234</div>
                <div class="label">总日志数</div>
            </div>
            <div class="stat-card">
                <div class="number" id="today-logs">156</div>
                <div class="label">今日日志</div>
            </div>
            <div class="stat-card">
                <div class="number" id="avg-response-time">25ms</div>
                <div class="label">平均响应时间</div>
            </div>
            <div class="stat-card">
                <div class="number" id="cache-hit-rate">78%</div>
                <div class="label">缓存命中率</div>
            </div>
            <div class="stat-card">
                <div class="number" id="unique-clients">45</div>
                <div class="label">唯一客户端</div>
            </div>
            <div class="stat-card">
                <div class="number" id="top-domain">google.com</div>
                <div class="label">最常查询域名</div>
            </div>
        </div>
        
        <!-- 日志筛选器 -->
        <div class="section">
            <h3>🔍 日志筛选</h3>
            <div class="filters">
                <div class="filter-row">
                    <div class="filter-group">
                        <label>时间范围</label>
                        <select id="time-range">
                            <option value="1h">最近1小时</option>
                            <option value="6h">最近6小时</option>
                            <option value="24h" selected>最近24小时</option>
                            <option value="7d">最近7天</option>
                            <option value="30d">最近30天</option>
                            <option value="custom">自定义</option>
                        </select>
                    </div>
                    <div class="filter-group">
                        <label>域名</label>
                        <input type="text" id="domain-filter" placeholder="输入域名关键词">
                    </div>
                    <div class="filter-group">
                        <label>客户端 IP</label>
                        <input type="text" id="client-filter" placeholder="输入客户端 IP">
                    </div>
                    <div class="filter-group">
                        <label>查询类型</label>
                        <select id="query-type-filter">
                            <option value="">全部类型</option>
                            <option value="A">A 记录</option>
                            <option value="AAAA">AAAA 记录</option>
                            <option value="CNAME">CNAME 记录</option>
                            <option value="MX">MX 记录</option>
                            <option value="TXT">TXT 记录</option>
                        </select>
                    </div>
                </div>
                <div class="filter-row">
                    <div class="filter-group">
                        <label>上游 DNS</label>
                        <select id="upstream-filter">
                            <option value="">全部上游</option>
                            <option value="china">中国 DNS</option>
                            <option value="intl">国际 DNS</option>
                            <option value="adguard">AdGuard DNS</option>
                        </select>
                    </div>
                    <div class="filter-group">
                        <label>响应状态</label>
                        <select id="status-filter">
                            <option value="">全部状态</option>
                            <option value="success">成功</option>
                            <option value="timeout">超时</option>
                            <option value="error">错误</option>
                        </select>
                    </div>
                    <div class="filter-group">
                        <label>缓存状态</label>
                        <select id="cache-filter">
                            <option value="">全部</option>
                            <option value="hit">命中</option>
                            <option value="miss">未命中</option>
                        </select>
                    </div>
                    <div class="filter-group">
                        <label>&nbsp;</label>
                        <button class="btn success" onclick="applyFilters()">🔍 应用筛选</button>
                        <button class="btn" onclick="resetFilters()">🔄 重置</button>
                    </div>
                </div>
            </div>
        </div>
        
        <!-- 日志数据表格 -->
        <div class="section">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
                <h3>📝 查询日志</h3>
                <div>
                    <button class="btn" onclick="refreshLogs()">🔄 刷新</button>
                    <button class="btn success" onclick="exportLogs()">📥 导出</button>
                    <button class="btn danger" onclick="clearLogs()">🗑️ 清空</button>
                </div>
            </div>
            <table class="table">
                <thead>
                    <tr>
                        <th>时间</th>
                        <th>客户端 IP</th>
                        <th>域名</th>
                        <th>类型</th>
                        <th>响应时间</th>
                        <th>上游</th>
                        <th>缓存</th>
                        <th>状态</th>
                    </tr>
                </thead>
                <tbody id="logs-tbody">
                    <tr>
                        <td>2025-08-15 15:45:23</td>
                        <td>192.168.1.100</td>
                        <td>google.com</td>
                        <td>A</td>
                        <td>45ms</td>
                        <td>国际 DNS</td>
                        <td>未命中</td>
                        <td><span class="log-level info">成功</span></td>
                    </tr>
                    <tr>
                        <td>2025-08-15 15:45:18</td>
                        <td>192.168.1.101</td>
                        <td>baidu.com</td>
                        <td>A</td>
                        <td>12ms</td>
                        <td>中国 DNS</td>
                        <td>命中</td>
                        <td><span class="log-level info">成功</span></td>
                    </tr>
                    <tr>
                        <td>2025-08-15 15:45:15</td>
                        <td>192.168.1.102</td>
                        <td>github.com</td>
                        <td>A</td>
                        <td>78ms</td>
                        <td>国际 DNS</td>
                        <td>未命中</td>
                        <td><span class="log-level warning">超时</span></td>
                    </tr>
                </tbody>
            </table>
            
            <!-- 分页 -->
            <div class="pagination">
                <button onclick="changePage(1)" disabled>«</button>
                <button onclick="changePage(1)" class="active">1</button>
                <button onclick="changePage(2)">2</button>
                <button onclick="changePage(3)">3</button>
                <button onclick="changePage(4)">4</button>
                <button onclick="changePage(5)">5</button>
                <button onclick="changePage(2)">»</button>
            </div>
        </div>
        
        <!-- 统计图表 -->
        <div class="section">
            <h3>📈 统计图表</h3>
            <div class="chart-container">
                📊 查询量趋势图、响应时间分布图、域名热度图 (需要集成图表库)
            </div>
        </div>
        
        <!-- 热点分析 -->
        <div class="section">
            <h3>🔥 热点分析</h3>
            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 30px;">
                <div>
                    <h4>🏆 最常查询域名</h4>
                    <div id="top-domains">
                        <div style="display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #eee;">
                            <span>google.com</span>
                            <span style="color: #fd7e14; font-weight: 600;">156 次</span>
                        </div>
                        <div style="display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #eee;">
                            <span>github.com</span>
                            <span style="color: #fd7e14; font-weight: 600;">89 次</span>
                        </div>
                        <div style="display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #eee;">
                            <span>baidu.com</span>
                            <span style="color: #fd7e14; font-weight: 600;">67 次</span>
                        </div>
                    </div>
                </div>
                <div>
                    <h4>👥 最活跃客户端</h4>
                    <div id="top-clients">
                        <div style="display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #eee;">
                            <span>192.168.1.100</span>
                            <span style="color: #fd7e14; font-weight: 600;">234 次</span>
                        </div>
                        <div style="display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #eee;">
                            <span>192.168.1.101</span>
                            <span style="color: #fd7e14; font-weight: 600;">189 次</span>
                        </div>
                        <div style="display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #eee;">
                            <span>192.168.1.102</span>
                            <span style="color: #fd7e14; font-weight: 600;">156 次</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
    
    <script>
        // 应用筛选器
        function applyFilters() {
            const timeRange = document.getElementById('time-range').value;
            const domain = document.getElementById('domain-filter').value;
            const client = document.getElementById('client-filter').value;
            const queryType = document.getElementById('query-type-filter').value;
            const upstream = document.getElementById('upstream-filter').value;
            const status = document.getElementById('status-filter').value;
            const cache = document.getElementById('cache-filter').value;
            
            console.log('应用筛选器:', { timeRange, domain, client, queryType, upstream, status, cache });
            alert('筛选功能开发中...');
        }
        
        // 重置筛选器
        function resetFilters() {
            document.getElementById('time-range').value = '24h';
            document.getElementById('domain-filter').value = '';
            document.getElementById('client-filter').value = '';
            document.getElementById('query-type-filter').value = '';
            document.getElementById('upstream-filter').value = '';
            document.getElementById('status-filter').value = '';
            document.getElementById('cache-filter').value = '';
        }
        
        // 刷新日志
        function refreshLogs() {
            alert('刷新日志功能开发中...');
        }
        
        // 导出日志
        function exportLogs() {
            alert('导出日志功能开发中...');
        }
        
        // 清空日志
        function clearLogs() {
            if (confirm('确定要清空所有日志吗？此操作不可恢复！')) {
                alert('清空日志功能开发中...');
            }
        }
        
        // 切换页面
        function changePage(page) {
            // 移除所有活跃状态
            document.querySelectorAll('.pagination button').forEach(btn => btn.classList.remove('active'));
            // 设置当前页面为活跃状态
            event.target.classList.add('active');
            alert('切换到第 ' + page + ' 页功能开发中...');
        }
        
        // 页面加载完成后初始化
        document.addEventListener('DOMContentLoaded', function() {
            console.log('日志分析页面加载完成');
        });
    </script>
</body>
</html>`
		w.Write([]byte(html))
	})

	// 暂时注释掉 admin 路由，因为需要重新实现
	// admin.BindRoutes(r, server, cfg)
	r.Handle("/metrics", promhttp.Handler())

	httpSrv := &http.Server{
		Addr:              cfg.ListenHTTP,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("admin http listening on %s", cfg.ListenHTTP)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http listen: %v", err)
		}
	}()

	log.Printf("dns listening on %s (udp/tcp)", cfg.ListenDNS)

	// 显示持久化状态
	if cfg.IsPersistenceEnabled() {
		log.Printf("数据持久化已启用，数据目录: %s", cfg.GetDataDir())
	} else {
		log.Printf("数据持久化已禁用")
	}

	// 规则远程订阅
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	syncer := dns.NewSyncManager(cfg, server)
	go syncer.Start(ctx)

	// Hot reload on SIGHUP
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	for s := range sigc {
		switch s {
		case syscall.SIGHUP:
			if err := server.ReloadRules(); err != nil {
				log.Printf("reload rules failed: %v", err)
			} else {
				log.Printf("rules reloaded")
			}
		case syscall.SIGTERM, syscall.SIGINT:
			// 保存数据到持久化存储
			if cfg.IsPersistenceEnabled() {
				log.Printf("正在保存数据到持久化存储...")
				if err := server.SaveData(); err != nil {
					log.Printf("保存数据失败: %v", err)
				}
			}

			_ = httpSrv.Close()
			_ = udpConn.Close()
			_ = tcpLn.Close()
			return
		}
	}

	// 配置管理页面
	r.Get("/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>配置管理 - BoomDNS</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8f9fa; }
        .header { background: linear-gradient(135deg, #6f42c1 0%, #e83e8c 100%); color: white; padding: 20px 0; }
        .header h1 { text-align: center; font-size: 2.5em; margin-bottom: 10px; }
        .header p { text-align: center; opacity: 0.9; }
        .container { max-width: 1400px; margin: 0 auto; padding: 20px; }
        .back-btn { display: inline-block; padding: 10px 20px; background: rgba(255,255,255,0.2); color: white; text-decoration: none; border-radius: 6px; margin-bottom: 20px; }
        .back-btn:hover { background: rgba(255,255,255,0.3); }
        .tabs { display: flex; border-bottom: 2px solid #e9ecef; margin-bottom: 20px; }
        .tab { padding: 12px 24px; background: none; border: none; cursor: pointer; color: #6c757d; font-weight: 500; transition: all 0.3s; }
        .tab.active { color: #6f42c1; border-bottom: 2px solid #6f42c1; }
        .tab:hover { color: #6f42c1; }
        .tab-content { display: none; }
        .tab-content.active { display: block; }
        .section { background: white; border-radius: 12px; padding: 25px; margin-bottom: 30px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); }
        .section h3 { color: #2c3e50; margin-bottom: 20px; font-size: 1.5em; border-bottom: 2px solid #e9ecef; padding-bottom: 10px; }
        .form-group { margin-bottom: 20px; }
        .form-group label { display: block; margin-bottom: 8px; font-weight: 500; color: #495057; }
        .form-group input, .form-group select, .form-group textarea { width: 100%; padding: 10px 12px; border: 1px solid #ced4da; border-radius: 6px; font-size: 14px; transition: border-color 0.3s; }
        .form-group input:focus, .form-group select:focus, .form-group textarea:focus { outline: none; border-color: #6f42c1; box-shadow: 0 0 0 2px rgba(111, 66, 193, 0.25); }
        .form-group textarea { min-height: 100px; resize: vertical; }
        .form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
        .btn { display: inline-block; padding: 10px 20px; margin: 5px; background: #007bff; color: white; text-decoration: none; border-radius: 6px; transition: all 0.3s; border: none; cursor: pointer; font-size: 14px; }
        .btn:hover { background: #0056b3; transform: translateY(-1px); }
        .btn.success { background: #28a745; }
        .btn.warning { background: #ffc107; color: #212529; }
        .btn.danger { background: #dc3545; }
        .btn.secondary { background: #6c757d; }
    </style>
</head>
<body>
    <div class="header">
        <h1>⚙️ 配置管理</h1>
        <p>系统配置、环境变量、配置文件管理</p>
    </div>
    
    <div class="container">
        <a href="/" class="back-btn">← 返回总览</a>
        
        <!-- 标签页 -->
        <div class="tabs">
            <button class="tab active" onclick="showTab('dns')">🌐 DNS 配置</button>
            <button class="tab" onclick="showTab('proxy')">🔗 代理配置</button>
            <button class="tab" onclick="showTab('system')">💻 系统配置</button>
        </div>
        
        <!-- DNS 配置 -->
        <div id="dns" class="tab-content active">
            <div class="section">
                <h3>🌐 DNS 服务器配置</h3>
                <div class="form-row">
                    <div class="form-group">
                        <label>DNS 监听端口</label>
                        <input type="text" id="dns-port" value=":5354" placeholder="例如: :53">
                    </div>
                    <div class="form-group">
                        <label>HTTP 管理端口</label>
                        <input type="text" id="http-port" value=":8081" placeholder="例如: :8080">
                    </div>
                </div>
                <div class="form-group">
                    <label>管理令牌</label>
                    <input type="text" id="admin-token" value="boomdns-secret-token-2024" placeholder="管理访问令牌">
                </div>
                
                <div style="text-align: right; margin-top: 20px;">
                    <button class="btn secondary" onclick="resetDNSConfig()">🔄 重置</button>
                    <button class="btn success" onclick="saveDNSConfig()">💾 保存配置</button>
                </div>
            </div>
        </div>
        
        <!-- 代理配置 -->
        <div id="proxy" class="tab-content">
            <div class="section">
                <h3>🔗 代理服务配置</h3>
                <div class="form-group">
                    <label>启用代理服务</label>
                    <select id="proxy-enabled">
                        <option value="true">启用</option>
                        <option value="false">禁用</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>代理监听端口</label>
                    <input type="text" id="proxy-port" value=":7890" placeholder="例如: :7890">
                </div>
                
                <div style="text-align: right; margin-top: 20px;">
                    <button class="btn secondary" onclick="resetProxyConfig()">🔄 重置</button>
                    <button class="btn success" onclick="saveProxyConfig()">💾 保存配置</button>
                </div>
            </div>
        </div>
        
        <!-- 系统配置 -->
        <div id="system" class="tab-content">
            <div class="section">
                <h3>💻 系统配置</h3>
                <div class="form-row">
                    <div class="form-group">
                        <label>数据目录</label>
                        <input type="text" id="data-dir" value="data" placeholder="数据存储目录">
                    </div>
                    <div class="form-group">
                        <label>日志级别</label>
                        <select id="log-level">
                            <option value="debug">Debug</option>
                            <option value="info" selected>Info</option>
                            <option value="warn">Warning</option>
                            <option value="error">Error</option>
                        </select>
                    </div>
                </div>
                
                <div style="text-align: right; margin-top: 20px;">
                    <button class="btn secondary" onclick="resetSystemConfig()">🔄 重置</button>
                    <button class="btn success" onclick="saveSystemConfig()">💾 保存配置</button>
                </div>
            </div>
        </div>
    </div>
    
    <script>
        // 显示标签页
        function showTab(tabName) {
            const tabContents = document.querySelectorAll('.tab-content');
            tabContents.forEach(content => content.classList.remove('active'));
            
            const tabs = document.querySelectorAll('.tab');
            tabs.forEach(tab => tab.classList.remove('active'));
            
            document.getElementById(tabName).classList.add('active');
            event.target.classList.add('active');
        }
        
        // 保存 DNS 配置
        function saveDNSConfig() {
            const config = {
                dnsPort: document.getElementById('dns-port').value,
                httpPort: document.getElementById('http-port').value,
                adminToken: document.getElementById('admin-token').value
            };
            
            console.log('保存 DNS 配置:', config);
            alert('DNS 配置保存功能开发中...');
        }
        
        // 重置 DNS 配置
        function resetDNSConfig() {
            document.getElementById('dns-port').value = ':5354';
            document.getElementById('http-port').value = ':8081';
            document.getElementById('admin-token').value = 'boomdns-secret-token-2024';
        }
        
        // 保存代理配置
        function saveProxyConfig() {
            const config = {
                enabled: document.getElementById('proxy-enabled').value === 'true',
                port: document.getElementById('proxy-port').value
            };
            
            console.log('保存代理配置:', config);
            alert('代理配置保存功能开发中...');
        }
        
        // 重置代理配置
        function resetProxyConfig() {
            document.getElementById('proxy-enabled').value = 'true';
            document.getElementById('proxy-port').value = ':7890';
        }
        
        // 保存系统配置
        function saveSystemConfig() {
            const config = {
                dataDir: document.getElementById('data-dir').value,
                logLevel: document.getElementById('log-level').value
            };
            
            console.log('保存系统配置:', config);
            alert('系统配置保存功能开发中...');
        }
        
        // 重置系统配置
        function resetSystemConfig() {
            document.getElementById('data-dir').value = 'data';
            document.getElementById('log-level').value = 'info';
        }
        
        // 页面加载完成后初始化
        document.addEventListener('DOMContentLoaded', function() {
            console.log('配置管理页面加载完成');
        });
    </script>
</body>
</html>`
		w.Write([]byte(html))
	})

	// 性能分析页面
	r.Get("/performance", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>性能分析 - BoomDNS</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8f9fa; }
        .header { background: linear-gradient(135deg, #20c997 0%, #17a2b8 100%); color: white; padding: 20px 0; }
        .header h1 { text-align: center; font-size: 2.5em; margin-bottom: 10px; }
        .header p { text-align: center; opacity: 0.9; }
        .container { max-width: 1400px; margin: 0 auto; padding: 20px; }
        .back-btn { display: inline-block; padding: 10px 20px; background: rgba(255,255,255,0.2); color: white; text-decoration: none; border-radius: 6px; margin-bottom: 20px; }
        .back-btn:hover { background: rgba(255,255,255,0.3); }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .stat-card { background: white; border-radius: 12px; padding: 20px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); text-align: center; }
        .stat-card .number { font-size: 2em; font-weight: bold; color: #20c997; margin-bottom: 10px; }
        .stat-card .label { color: #6c757d; font-size: 1em; }
        .section { background: white; border-radius: 12px; padding: 25px; margin-bottom: 30px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); }
        .section h3 { color: #2c3e50; margin-bottom: 20px; font-size: 1.5em; border-bottom: 2px solid #e9ecef; padding-bottom: 10px; }
        .tabs { display: flex; border-bottom: 2px solid #e9ecef; margin-bottom: 20px; }
        .tab { padding: 12px 24px; background: none; border: none; cursor: pointer; color: #6c757d; font-weight: 500; transition: all 0.3s; }
        .tab.active { color: #20c997; border-bottom: 2px solid #20c997; }
        .tab:hover { color: #20c997; }
        .tab-content { display: none; }
        .tab-content.active { display: block; }
        .chart-container { height: 300px; margin: 20px 0; background: #f8f9fa; border-radius: 8px; display: flex; align-items: center; justify-content: center; color: #6c757d; border: 2px dashed #dee2e6; }
        .performance-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 30px; }
        .performance-item { background: #f8f9fa; padding: 20px; border-radius: 8px; border-left: 4px solid #20c997; }
        .performance-item h4 { color: #2c3e50; margin-bottom: 15px; }
        .performance-metric { display: flex; justify-content: space-between; margin: 10px 0; padding: 8px 0; border-bottom: 1px solid #e9ecef; }
        .performance-metric:last-child { border-bottom: none; }
        .metric-label { color: #6c757d; font-weight: 500; }
        .metric-value { color: #20c997; font-weight: 600; }
        .btn { display: inline-block; padding: 10px 20px; margin: 5px; background: #007bff; color: white; text-decoration: none; border-radius: 6px; transition: all 0.3s; border: none; cursor: pointer; font-size: 14px; }
        .btn:hover { background: #0056b3; transform: translateY(-1px); }
        .btn.success { background: #28a745; }
        .btn.warning { background: #ffc107; color: #212529; }
        .btn.danger { background: #dc3545; }
        .btn.secondary { background: #6c757d; }
        .progress-bar { width: 100%; height: 8px; background: #e9ecef; border-radius: 4px; overflow: hidden; margin: 10px 0; }
        .progress-fill { height: 100%; background: linear-gradient(90deg, #20c997, #17a2b8); transition: width 0.3s; }
        .refresh-time { text-align: right; margin-bottom: 20px; color: #6c757d; font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="header">
        <h1>📈 性能分析</h1>
        <p>系统性能指标、趋势分析与优化建议</p>
    </div>
    
    <div class="container">
        <a href="/" class="back-btn">← 返回总览</a>
        
        <div class="refresh-time">最后更新: <span id="refresh-time">加载中...</span></div>
        
        <!-- 性能指标概览 -->
        <div class="stats-grid">
            <div class="stat-card">
                <div class="number" id="qps">--</div>
                <div class="label">查询/秒 (QPS)</div>
            </div>
            <div class="stat-card">
                <div class="number" id="avg-response-time">--</div>
                <div class="label">平均响应时间</div>
            </div>
            <div class="stat-card">
                <div class="number" id="cache-hit-rate">--</div>
                <div class="label">缓存命中率</div>
            </div>
            <div class="stat-card">
                <div class="number" id="error-rate">--</div>
                <div class="label">错误率</div>
            </div>
            <div class="stat-card">
                <div class="number" id="cpu-usage">--</div>
                <div class="label">CPU 使用率</div>
            </div>
            <div class="stat-card">
                <div class="number" id="memory-usage">--</div>
                <div class="label">内存使用率</div>
            </div>
        </div>
        
        <!-- 标签页 -->
        <div class="tabs">
            <button class="tab active" onclick="showTab('overview')">📊 性能概览</button>
            <button class="tab" onclick="showTab('trends')">📈 趋势分析</button>
            <button class="tab" onclick="showTab('bottlenecks')">🔍 瓶颈分析</button>
            <button class="tab" onclick="showTab('optimization')">⚡ 优化建议</button>
        </div>
        
        <!-- 性能概览 -->
        <div id="overview" class="tab-content active">
            <div class="section">
                <h3>📊 实时性能指标</h3>
                <div class="performance-grid">
                    <div class="performance-item">
                        <h4>🚀 DNS 查询性能</h4>
                        <div class="performance-metric">
                            <span class="metric-label">当前 QPS:</span>
                            <span class="metric-value" id="current-qps">156</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">峰值 QPS:</span>
                            <span class="metric-value" id="peak-qps">234</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">平均响应时间:</span>
                            <span class="metric-value" id="overview-avg-response">25ms</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">95% 响应时间:</span>
                            <span class="metric-value" id="p95-response">45ms</span>
                        </div>
                    </div>
                    
                    <div class="performance-item">
                        <h4>💾 缓存性能</h4>
                        <div class="performance-metric">
                            <span class="metric-label">缓存命中率:</span>
                            <span class="metric-value" id="overview-cache-hit">78%</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">缓存条目数:</span>
                            <span class="metric-value" id="cache-entries">1,234</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">缓存大小:</span>
                            <span class="metric-value" id="cache-size">2.1 MB</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">缓存效率:</span>
                            <span class="metric-value" id="cache-efficiency">优秀</span>
                        </div>
                    </div>
                </div>
                
                <div class="chart-container">
                    📊 实时性能监控图表 (需要集成图表库)
                </div>
            </div>
        </div>
        
        <!-- 趋势分析 -->
        <div id="trends" class="tab-content">
            <div class="section">
                <h3>📈 性能趋势分析</h3>
                <div class="performance-grid">
                    <div>
                        <h4>⏰ 时间维度分析</h4>
                        <div class="performance-metric">
                            <span class="metric-label">小时趋势:</span>
                            <span class="metric-value">稳定上升</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">日趋势:</span>
                            <span class="metric-value">周期性波动</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">周趋势:</span>
                            <span class="metric-value">工作日高峰</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">月趋势:</span>
                            <span class="metric-value">持续增长</span>
                        </div>
                    </div>
                    
                    <div>
                        <h4>🌐 域名维度分析</h4>
                        <div class="performance-metric">
                            <span class="metric-label">热门域名:</span>
                            <span class="metric-value">google.com</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">查询频率:</span>
                            <span class="metric-value">高频</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">响应时间:</span>
                            <span class="metric-value">45ms</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">缓存效果:</span>
                            <span class="metric-value">良好</span>
                        </div>
                    </div>
                </div>
                
                <div class="chart-container">
                    📈 性能趋势图表 (需要集成图表库)
                </div>
            </div>
        </div>
        
        <!-- 瓶颈分析 -->
        <div id="bottlenecks" class="tab-content">
            <div class="section">
                <h3>🔍 性能瓶颈分析</h3>
                <div class="performance-grid">
                    <div>
                        <h4>⚠️ 当前瓶颈</h4>
                        <div class="performance-metric">
                            <span class="metric-label">主要瓶颈:</span>
                            <span class="metric-value">上游 DNS 响应</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">影响程度:</span>
                            <span class="metric-value">中等</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">瓶颈位置:</span>
                            <span class="metric-value">国际 DNS 服务器</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">建议措施:</span>
                            <span class="metric-value">增加备用服务器</span>
                        </div>
                    </div>
                    
                    <div>
                        <h4>📊 瓶颈指标</h4>
                        <div class="performance-metric">
                            <span class="metric-label">响应时间:</span>
                            <span class="metric-value">120ms</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">超时率:</span>
                            <span class="metric-value">2.3%</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">失败率:</span>
                            <span class="metric-value">0.8%</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">健康状态:</span>
                            <span class="metric-value">需要关注</span>
                        </div>
                    </div>
                </div>
                
                <div class="chart-container">
                    🔍 瓶颈分析图表 (需要集成图表库)
                </div>
            </div>
        </div>
        
        <!-- 优化建议 -->
        <div id="optimization" class="tab-content">
            <div class="section">
                <h3>⚡ 性能优化建议</h3>
                <div class="performance-grid">
                    <div>
                        <h4>🚀 立即优化</h4>
                        <div class="performance-metric">
                            <span class="metric-label">优先级:</span>
                            <span class="metric-value">高</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">预期提升:</span>
                            <span class="metric-value">15-20%</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">实施难度:</span>
                            <span class="metric-value">低</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">建议:</span>
                            <span class="metric-value">增加缓存条目</span>
                        </div>
                    </div>
                    
                    <div>
                        <h4>📈 中期优化</h4>
                        <div class="performance-metric">
                            <span class="metric-label">优先级:</span>
                            <span class="metric-value">中</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">预期提升:</span>
                            <span class="metric-value">25-30%</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">实施难度:</span>
                            <span class="metric-value">中</span>
                        </div>
                        <div class="performance-metric">
                            <span class="metric-label">建议:</span>
                            <span class="metric-value">优化上游选择</span>
                        </div>
                    </div>
                </div>
                
                <div style-margin-top: 20px;">
                    <h4>🔧 具体优化措施</h4>
                    <ul style="margin-left: 20px; color: #495057; line-height: 1.6;">
                        <li>增加 DNS 缓存条目数量，提高缓存命中率</li>
                        <li>优化上游 DNS 服务器选择策略，减少响应时间</li>
                        <li>实施智能负载均衡，分散查询压力</li>
                        <li>启用 DNS 预取功能，提前解析常用域名</li>
                        <li>优化网络配置，减少网络延迟</li>
                    </ul>
                </div>
                
                <div style="text-align: center; margin-top: 30px;">
                    <button class="btn success" onclick="applyOptimizations()">🚀 应用优化</button>
                    <button class="btn secondary" onclick="generateReport()">📋 生成报告</button>
                </div>
            </div>
        </div>
    </div>
    
    <script>
        // 显示标签页
        function showTab(tabName) {
            const tabContents = document.querySelectorAll('.tab-content');
            tabContents.forEach(content => content.classList.remove('active'));
            
            const tabs = document.querySelectorAll('.tab');
            tabs.forEach(tab => tab.classList.remove('active'));
            
            document.getElementById(tabName).classList.add('active');
            event.target.classList.add('active');
        }
        
        // 更新性能数据
        function updatePerformanceData() {
            // 模拟性能数据
            const qps = Math.floor(Math.random() * 100) + 100;
            const avgResponse = Math.floor(Math.random() * 30) + 15;
            const cacheHitRate = Math.floor(Math.random() * 20) + 70;
            const errorRate = (Math.random() * 2).toFixed(1);
            const cpuUsage = Math.floor(Math.random() * 30) + 10;
            const memoryUsage = Math.floor(Math.random() * 40) + 20;
            
            // 更新概览指标
            document.getElementById('qps').textContent = qps;
            document.getElementById('avg-response-time').textContent = avgResponse + 'ms';
            document.getElementById('cache-hit-rate').textContent = cacheHitRate + '%';
            document.getElementById('error-rate').textContent = errorRate + '%';
            document.getElementById('cpu-usage').textContent = cpuUsage + '%';
            document.getElementById('memory-usage').textContent = memoryUsage + '%';
            
            // 更新时间
            document.getElementById('refresh-time').textContent = new Date().toLocaleString('zh-CN');
        }
        
        // 应用优化
        function applyOptimizations() {
            alert('性能优化功能开发中...');
        }
        
        // 生成报告
        function generateReport() {
            alert('报告生成功能开发中...');
        }
        
        // 页面加载完成后获取数据
        document.addEventListener('DOMContentLoaded', updatePerformanceData);
        
        // 每30秒刷新一次数据
        setInterval(updatePerformanceData, 30000);
    </script>
</body>
</html>`
		w.Write([]byte(html))
	})

	// 安全设置页面
	r.Get("/security", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>安全设置 - BoomDNS</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f8f9fa; }
        .header { background: linear-gradient(135deg, #dc3545 0%, #fd7e14 100%); color: white; padding: 20px 0; }
        .header h1 { text-align: center; font-size: 2.5em; margin-bottom: 10px; }
        .header p { text-align: center; opacity: 0.9; }
        .container { max-width: 1400px; margin: 0 auto; padding: 20px; }
        .back-btn { display: inline-block; padding: 10px 20px; background: rgba(255,255,255,0.2); color: white; text-decoration: none; border-radius: 6px; margin-bottom: 20px; }
        .back-btn:hover { background: rgba(255,255,255,0.3); }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .stat-card { background: white; border-radius: 12px; padding: 20px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); text-align: center; }
        .stat-card .number { font-size: 2em; font-weight: bold; color: #dc3545; margin-bottom: 10px; }
        .stat-card .label { color: #6c757d; font-size: 1em; }
        .section { background: white; border-radius: 12px; padding: 25px; margin-bottom: 30px; box-shadow: 0 4px 20px rgba(0,0,0,0.1); }
        .section h3 { color: #2c3e50; margin-bottom: 20px; font-size: 1.5em; border-bottom: 2px solid #e9ecef; padding-bottom: 10px; }
        .tabs { display: flex; border-bottom: 2px solid #e9ecef; margin-bottom: 20px; }
        .tab { padding: 12px 24px; background: none; border: none; cursor: pointer; color: #6c757d; font-weight: 500; transition: all 0.3s; }
        .tab.active { color: #dc3545; border-bottom: 2px solid #dc3545; }
        .tab:hover { color: #dc3545; }
        .tab-content { display: none; }
        .tab-content.active { display: block; }
        .form-group { margin-bottom: 20px; }
        .form-group label { display: block; margin-bottom: 8px; font-weight: 500; color: #495057; }
        .form-group input, .form-group select, .form-group textarea { width: 100%; padding: 10px 12px; border: 1px solid #ced4da; border-radius: 6px; font-size: 14px; transition: border-color 0.3s; }
        .form-group input:focus, .form-group select:focus, .form-group textarea:focus { outline: none; border-color: #dc3545; box-shadow: 0 0 0 2px rgba(220, 53, 69, 0.25); }
        .form-group textarea { min-height: 100px; resize: vertical; }
        .form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
        .btn { display: inline-block; padding: 10px 20px; margin: 5px; background: #007bff; color: white; text-decoration: none; border-radius: 6px; transition: all 0.3s; border: none; cursor: pointer; font-size: 14px; }
        .btn:hover { background: #0056b3; transform: translateY(-1px); }
        .btn.success { background: #28a745; }
        .btn.warning { background: #ffc107; color: #212529; }
        .btn.danger { background: #dc3545; }
        .btn.secondary { background: #6c757d; }
        .security-status { padding: 8px 16px; border-radius: 20px; font-size: 0.9em; font-weight: 500; margin-left: 10px; }
        .status-secure { background: #d4edda; color: #155724; }
        .status-warning { background: #fff3cd; color: #856404; }
        .status-danger { background: #f8d7da; color: #721c24; }
        .security-item { display: flex; justify-content: space-between; align-items: center; padding: 15px; background: #f8f9fa; border-radius: 8px; margin-bottom: 10px; border-left: 4px solid #dee2e6; }
        .security-item.secure { border-left-color: #28a745; }
        .security-item.warning { border-left-color: #ffc107; }
        .security-item.danger { border-left-color: #dc3545; }
        .security-item .info { flex: 1; }
        .security-item .title { font-weight: 600; color: #2c3e50; margin-bottom: 5px; }
        .security-item .description { color: #6c757d; font-size: 0.9em; }
        .security-item .actions { display: flex; gap: 10px; }
        .btn.small { padding: 6px 12px; font-size: 12px; }
        .alert { padding: 15px; margin: 15px 0; border-radius: 8px; border-left: 4px solid; }
        .alert-warning { background: #fff3cd; border-color: #ffc107; color: #856404; }
        .alert-danger { background: #f8d7da; border-color: #dc3545; color: #721c24; }
        .alert-success { background: #d4edda; border-color: #28a745; color: #155724; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🔒 安全设置</h1>
        <p>访问控制、权限管理、安全策略配置</p>
    </div>
    
    <div class="container">
        <a href="/" class="back-btn">← 返回总览</a>
        
        <!-- 安全状态概览 -->
        <div class="stats-grid">
            <div class="stat-card">
                <div class="number" id="security-score">85</div>
                <div class="label">安全评分</div>
            </div>
            <div class="stat-card">
                <div class="number" id="active-users">3</div>
                <div class="label">活跃用户</div>
            </div>
            <div class="stat-card">
                <div class="number" id="failed-logins">2</div>
                <div class="label">失败登录</div>
            </div>
            <div class="stat-card">
                <div class="number" id="blocked-ips">1</div>
                <div class="label">已封禁 IP</div>
            </div>
            <div class="stat-card">
                <div class="number" id="security-events">5</div>
                <div class="label">安全事件</div>
            </div>
            <div class="stat-card">
                <div class="number" id="last-audit">2h</div>
                <div class="label">最后审计</div>
            </div>
        </div>
        
        <!-- 安全告警 -->
        <div class="alert alert-warning">
            <strong>⚠️ 安全提醒</strong> - 检测到 2 次失败的登录尝试，建议检查访问日志
        </div>
        
        <!-- 标签页 -->
        <div class="tabs">
            <button class="tab active" onclick="showTab('access')">🔐 访问控制</button>
            <button class="tab" onclick="showTab('users')">👥 用户管理</button>
            <button class="tab" onclick="showTab('firewall')">🔥 防火墙</button>
            <button class="tab" onclick="showTab('audit')">📋 安全审计</button>
            <button class="tab" onclick="showTab('settings')">⚙️ 安全设置</button>
        </div>
        
        <!-- 访问控制 -->
        <div id="access" class="tab-content active">
            <div class="section">
                <h3>🔐 访问控制管理</h3>
                <div class="form-group">
                    <label>管理令牌</label>
                    <div style="display: flex; gap: 10px; align-items: center;">
                        <input type="text" id="admin-token" value="boomdns-secret-token-2024" readonly>
                        <button class="btn btn-warning" onclick="regenerateToken()">🔄 重新生成</button>
                        <button class="btn btn-danger" onclick="revokeToken()">❌ 撤销</button>
                    </div>
                </div>
                
                <div class="form-group">
                    <label>允许的 IP 地址</label>
                    <textarea id="allowed-ips" placeholder="每行一个 IP 地址或 CIDR 范围">192.168.1.0/24
127.0.0.1
::1</textarea>
                </div>
                
                <div class="form-group">
                    <label>访问时间限制</label>
                    <div class="form-row">
                        <div class="form-group">
                            <label>开始时间</label>
                            <input type="time" id="access-start-time" value="00:00">
                        </div>
                        <div class="form-group">
                            <label>结束时间</label>
                            <input type="time" id="access-end-time" value="23:59">
                        </div>
                    </div>
                </div>
                
                <div style="text-align: right; margin-top: 20px;">
                    <button class="btn secondary" onclick="resetAccessControl()">🔄 重置</button>
                    <button class="btn success" onclick="saveAccessControl()">💾 保存设置</button>
                </div>
            </div>
        </div>
        
        <!-- 用户管理 -->
        <div id="users" class="tab-content">
            <div class="section">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
                    <h3>👥 用户管理</h3>
                    <button class="btn success" onclick="showAddUserModal()">➕ 添加用户</button>
                </div>
                
                <div class="security-item secure">
                    <div class="info">
                        <div class="title">管理员 (admin)</div>
                        <div class="description">超级管理员，拥有所有权限</div>
                    </div>
                    <div class="actions">
                        <button class="btn btn-warning btn-small" onclick="editUser('admin')">编辑</button>
                        <button class="btn btn-danger btn-small" onclick="deleteUser('admin')">删除</button>
                    </div>
                </div>
                
                <div class="security-item warning">
                    <div class="info">
                        <div class="title">操作员 (operator)</div>
                        <div class="description">系统操作员，拥有部分管理权限</div>
                    </div>
                    <div class="actions">
                        <button class="btn btn-warning btn-small" onclick="editUser('operator')">编辑</button>
                        <button class="btn btn-danger btn-small" onclick="deleteUser('operator')">删除</button>
                    </div>
                </div>
                
                <div class="security-item secure">
                    <div class="info">
                        <div class="title">查看者 (viewer)</div>
                        <div class="description">只读用户，仅能查看系统状态</div>
                    </div>
                    <div class="actions">
                        <button class="btn btn-warning btn-small" onclick="editUser('viewer')">编辑</button>
                        <button class="btn btn-danger btn-small" onclick="deleteUser('viewer')">删除</button>
                    </div>
                </div>
            </div>
        </div>
        
        <!-- 防火墙 -->
        <div id="firewall" class="tab-content">
            <div class="section">
                <h3>🔥 防火墙设置</h3>
                <div class="form-group">
                    <label>启用防火墙</label>
                    <select id="firewall-enabled">
                        <option value="true" selected>启用</option>
                        <option value="false">禁用</option>
                    </select>
                </div>
                
                <div class="form-group">
                    <label>封禁规则</label>
                    <textarea id="blocked-rules" placeholder="每行一个封禁规则">192.168.1.100
10.0.0.0/8
malicious.example.com</textarea>
                </div>
                
                <div class="form-group">
                    <label>自动封禁设置</label>
                    <div class="form-row">
                        <div class="form-group">
                            <label>失败登录次数</label>
                            <input type="number" id="max-failed-logins" value="5" min="1" max="10">
                        </div>
                        <div class="form-group">
                            <label>封禁时间 (分钟)</label>
                            <input type="number" id="ban-duration" value="30" min="5" max="1440">
                        </div>
                    </div>
                </div>
                
                <div style="text-align: right; margin-top: 20px;">
                    <button class="btn secondary" onclick="resetFirewall()">🔄 重置</button>
                    <button class="btn success" onclick="saveFirewall()">💾 保存设置</button>
                </div>
            </div>
        </div>
        
        <!-- 安全审计 -->
        <div id="audit" class="tab-content">
            <div class="section">
                <h3>📋 安全审计日志</h3>
                <div class="security-item warning">
                    <div class="info">
                        <div class="title">登录失败</div>
                        <div class="description">IP: 192.168.1.105 | 时间: 2025-08-15 16:30:15 | 原因: 无效令牌</div>
                    </div>
                    <div class="actions">
                        <button class="btn btn-danger btn-small" onclick="blockIP('192.168.1.105')">封禁 IP</button>
                    </div>
                </div>
                
                <div class="security-item secure">
                    <div class="info">
                        <div class="title">配置更改</div>
                        <div class="description">用户: admin | 时间: 2025-08-15 16:25:30 | 操作: 修改 DNS 配置</div>
                    </div>
                    <div class="actions">
                        <button class="btn btn-small" onclick="viewDetails('config-change')">查看详情</button>
                    </div>
                </div>
                
                <div class="security-item secure">
                    <div class="info">
                        <div class="title">系统启动</div>
                        <div class="description">时间: 2025-08-15 16:13:51 | 状态: 正常启动</div>
                    </div>
                    <div class="actions">
                        <button class="btn btn-small" onclick="viewDetails('system-start')">查看详情</button>
                    </div>
                </div>
                
                <div style="text-align: center; margin-top: 20px;">
                    <button class="btn" onclick="exportAuditLogs()">📥 导出日志</button>
                    <button class="btn btn-danger" onclick="clearAuditLogs()">🗑️ 清空日志</button>
                </div>
            </div>
        </div>
        
        <!-- 安全设置 -->
        <div id="settings" class="tab-content">
            <div class="section">
                <h3>⚙️ 安全策略设置</h3>
                <div class="form-group">
                    <label>密码策略</label>
                    <div class="form-row">
                        <div class="form-group">
                            <label>最小长度</label>
                            <input type="number" id="min-password-length" value="8" min="6" max="20">
                        </div>
                        <div class="form-group">
                            <label>复杂度要求</label>
                            <select id="password-complexity">
                                <option value="low">低 (仅字母数字)</option>
                                <option value="medium" selected>中 (包含特殊字符)</option>
                                <option value="high">高 (包含大小写和特殊字符)</option>
                            </select>
                        </div>
                    </div>
                </div>
                
                <div class="form-group">
                    <label>会话管理</label>
                    <div class="form-row">
                        <div class="form-group">
                            <label>会话超时 (分钟)</label>
                            <input type="number" id="session-timeout" value="30" min="5" max="1440">
                        </div>
                        <div class="form-group">
                            <label>最大并发会话</label>
                            <input type="number" id="max-sessions" value="3" min="1" max="10">
                        </div>
                    </div>
                </div>
                
                <div class="form-group">
                    <label>日志设置</label>
                    <div class="form-row">
                        <div class="form-group">
                            <label>日志保留天数</label>
                            <input type="number" id="log-retention" value="90" min="7" max="365">
                        </div>
                        <div class="form-group">
                            <label>日志级别</label>
                            <select id="security-log-level">
                                <option value="info">信息</option>
                                <option value="warning" selected>警告</option>
                                <option value="error">错误</option>
                            </select>
                        </div>
                    </div>
                </div>
                
                <div style="text-align: right; margin-top: 20px;">
                    <button class="btn secondary" onclick="resetSecuritySettings()">🔄 重置</button>
                    <button class="btn success" onclick="saveSecuritySettings()">💾 保存设置</button>
                </div>
            </div>
        </div>
    </div>
    
    <script>
        // 显示标签页
        function showTab(tabName) {
            const tabContents = document.querySelectorAll('.tab-content');
            tabContents.forEach(content => content.classList.remove('active'));
            
            const tabs = document.querySelectorAll('.tab');
            tabs.forEach(tab => tab.classList.remove('active'));
            
            document.getElementById(tabName).classList.add('active');
            event.target.classList.add('active');
        }
        
        // 重新生成令牌
        function regenerateToken() {
            if (confirm('确定要重新生成管理令牌吗？这将使当前令牌失效。')) {
                const newToken = 'boomdns-' + Math.random().toString(36).substr(2, 9) + '-' + Date.now().toString(36);
                document.getElementById('admin-token').value = newToken;
                alert('令牌已重新生成: ' + newToken);
            }
        }
        
        // 撤销令牌
        function revokeToken() {
            if (confirm('确定要撤销当前管理令牌吗？这将立即禁用所有访问。')) {
                alert('令牌撤销功能开发中...');
            }
        }
        
        // 保存访问控制
        function saveAccessControl() {
            const config = {
                adminToken: document.getElementById('admin-token').value,
                allowedIPs: document.getElementById('allowed-ips').value.split('\n').filter(line => line.trim()),
                accessStartTime: document.getElementById('access-start-time').value,
                accessEndTime: document.getElementById('access-end-time').value
            };
            
            console.log('保存访问控制配置:', config);
            alert('访问控制配置保存功能开发中...');
        }
        
        // 重置访问控制
        function resetAccessControl() {
            document.getElementById('admin-token').value = 'boomdns-secret-token-2024';
            document.getElementById('allowed-ips').value = '192.168.1.0/24\n127.0.0.1\n::1';
            document.getElementById('access-start-time').value = '00:00';
            document.getElementById('access-end-time').value = '23:59';
        }
        
        // 编辑用户
        function editUser(username) {
            alert('编辑用户 ' + username + ' 功能开发中...');
        }
        
        // 删除用户
        function deleteUser(username) {
            if (confirm('确定要删除用户 ' + username + ' 吗？')) {
                alert('删除用户功能开发中...');
            }
        }
        
        // 添加用户模态框
        function showAddUserModal() {
            alert('添加用户功能开发中...');
        }
        
        // 保存防火墙设置
        function saveFirewall() {
            const config = {
                enabled: document.getElementById('firewall-enabled').value === 'true',
                blockedRules: document.getElementById('blocked-rules').value.split('\n').filter(line => line.trim()),
                maxFailedLogins: parseInt(document.getElementById('max-failed-logins').value),
                banDuration: parseInt(document.getElementById('ban-duration').value)
            };
            
            console.log('保存防火墙配置:', config);
            alert('防火墙配置保存功能开发中...');
        }
        
        // 重置防火墙
        function resetFirewall() {
            document.getElementById('firewall-enabled').value = 'true';
            document.getElementById('blocked-rules').value = '192.168.1.100\n10.0.0.0/8\nmalicious.example.com';
            document.getElementById('max-failed-logins').value = '5';
            document.getElementById('ban-duration').value = '30';
        }
        
        // 封禁 IP
        function blockIP(ip) {
            if (confirm('确定要封禁 IP ' + ip + ' 吗？')) {
                alert('封禁 IP 功能开发中...');
            }
        }
        
        // 查看详情
        function viewDetails(eventType) {
            alert('查看事件详情功能开发中...');
        }
        
        // 导出审计日志
        function exportAuditLogs() {
            alert('导出审计日志功能开发中...');
        }
        
        // 清空审计日志
        function clearAuditLogs() {
            if (confirm('确定要清空所有审计日志吗？此操作不可恢复！')) {
                alert('清空审计日志功能开发中...');
            }
        }
        
        // 保存安全设置
        function saveSecuritySettings() {
            const config = {
                minPasswordLength: parseInt(document.getElementById('min-password-length').value),
                passwordComplexity: document.getElementById('password-complexity').value,
                sessionTimeout: parseInt(document.getElementById('session-timeout').value),
                maxSessions: parseInt(document.getElementById('max-sessions').value),
                logRetention: parseInt(document.getElementById('log-retention').value),
                securityLogLevel: document.getElementById('security-log-level').value
            };
            
            console.log('保存安全设置:', config);
            alert('安全设置保存功能开发中...');
        }
        
        // 重置安全设置
        function resetSecuritySettings() {
            document.getElementById('min-password-length').value = '8';
            document.getElementById('password-complexity').value = 'medium';
            document.getElementById('session-timeout').value = '30';
            document.getElementById('max-sessions').value = '3';
            document.getElementById('log-retention').value = '90';
            document.getElementById('security-log-level').value = 'warning';
        }
        
        // 页面加载完成后初始化
        document.addEventListener('DOMContentLoaded', function() {
            console.log('安全设置页面加载完成');
        });
    </script>
</body>
</html>`
		w.Write([]byte(html))
	})
}
