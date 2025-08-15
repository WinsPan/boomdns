// BoomDNS Web管理界面 JavaScript应用
class BoomDNSApp {
    constructor() {
        this.config = {
            apiBase: '/api',
            adminToken: this.getAdminToken(),
            refreshInterval: 5000, // 5秒刷新一次
            maxLogs: 200
        };
        
        // 日志管理相关状态
        this.logState = {
            currentPage: 1,
            pageSize: 100,
            totalLogs: 0,
            filteredLogs: [],
            searchTerm: '',
            routeFilter: '',
            timeFilter: '24h',
            sortField: 'time',
            sortOrder: 'desc'
        };
        
        this.searchTimeout = null; // 搜索防抖超时
        this.init();
    }

    init() {
        this.bindEvents();
        this.loadInitialData();
        this.startAutoRefresh();
        this.checkHealth();
    }

    // 获取管理令牌
    getAdminToken() {
        // 从localStorage或URL参数获取token
        const urlParams = new URLSearchParams(window.location.search);
        const token = urlParams.get('token') || localStorage.getItem('boomdns_token') || 'boomdns-secret-token-2024';
        
        // 如果没有令牌，使用默认令牌
        if (!token) {
            localStorage.setItem('boomdns_token', 'boomdns-secret-token-2024');
            return 'boomdns-secret-token-2024';
        }
        
        return token;
    }

    // 绑定事件
    bindEvents() {
        // 安全地绑定事件监听器
        const bindEvent = (id, event, handler) => {
            const element = document.getElementById(id);
            if (element) {
                element.addEventListener(event, handler);
            } else {
                console.warn(`元素 ${id} 不存在，跳过事件绑定`);
            }
        };

        // 刷新按钮
        bindEvent('refresh-btn', 'click', () => {
            this.loadInitialData();
        });

        // 保存规则按钮
        bindEvent('save-rules-btn', 'click', () => {
            this.saveRules();
        });

        // 重载规则按钮
        bindEvent('reload-rules-btn', 'click', () => {
            this.reloadRules();
        });

        // 添加新规则按钮
        bindEvent('add-rule-btn', 'click', () => {
            this.addNewRule();
        });

        // 清空日志按钮
        bindEvent('clear-logs-btn', 'click', () => {
            this.clearLogs();
        });

        // 延迟统计相关事件
        bindEvent('refresh-latency-btn', 'click', () => {
            this.loadLatencyStats();
        });

        bindEvent('reset-latency-btn', 'click', () => {
            this.resetLatencyStats();
        });

        // 导出配置按钮
        bindEvent('export-config-btn', 'click', () => {
            this.exportConfig();
        });

        // 刷新缓存按钮
        bindEvent('refresh-cache-btn', 'click', () => {
            this.loadCacheStats();
        });

        // 清空缓存按钮
        bindEvent('clear-cache-btn', 'click', () => {
            this.clearCache();
        });

        // 立即同步按钮
        bindEvent('sync-now-btn', 'click', () => {
            this.syncNow();
        });

        // 查看同步详情按钮
        bindEvent('refresh-sync-btn', 'click', () => {
            this.loadSyncStatus();
        });

        // 日志控制相关事件
        bindEvent('log-search', 'input', (e) => {
            this.logState.searchTerm = e.target.value;
            this.debounceSearch();
        });

        bindEvent('log-route-filter', 'change', (e) => {
            this.logState.routeFilter = e.target.value;
            this.filterLogs();
        });

        bindEvent('log-time-filter', 'change', (e) => {
            this.logState.timeFilter = e.target.value;
            this.filterLogs();
        });

        bindEvent('log-page-size', 'change', (e) => {
            this.logState.pageSize = parseInt(e.target.value);
            this.logState.currentPage = 1;
            this.loadLogs();
        });

        // 分页控制事件
        bindEvent('log-first-page', 'click', () => {
            this.goToPage(1);
        });

        bindEvent('log-prev-page', 'click', () => {
            this.goToPage(this.logState.currentPage - 1);
        });

        bindEvent('log-next-page', 'click', () => {
            this.goToPage(this.logState.currentPage + 1);
        });

        bindEvent('log-last-page', 'click', () => {
            this.goToPage(this.getTotalPages());
        });

        // 排序事件
        bindEvent('log-time-header', 'click', () => {
            this.setSort('time');
        });

        bindEvent('log-domain-header', 'click', () => {
            this.setSort('name');
        });

        bindEvent('log-route-header', 'click', () => {
            this.setSort('route');
        });

        bindEvent('log-latency-header', 'click', () => {
            this.setSort('latency');
        });

        // 导出日志按钮
        bindEvent('export-logs-btn', 'click', () => {
            this.exportLogs();
        });

        // 清空日志按钮
        bindEvent('clear-logs-btn', 'click', () => {
            this.clearLogs();
        });

        // 通知关闭按钮
        bindEvent('notification-close', 'click', () => {
            this.hideNotification();
        });
        
        // 订阅源管理事件绑定
        bindEvent('add-source-btn', 'click', () => {
            this.showAddSourceModal();
        });
        
        bindEvent('save-source-btn', 'click', () => {
            this.saveSubscriptionSource();
        });
        
        bindEvent('cancel-source-btn', 'click', () => {
            this.hideSubscriptionSourceModal();
        });
    }

    // 加载初始数据
    async loadInitialData() {
        try {
            console.log('开始加载初始数据...');
            
            const results = await Promise.allSettled([
                this.loadRules(),
                this.loadLogs(),
                this.loadMetrics(),
                this.loadSyncStatus(),
                this.loadLatencyStats(),
                this.loadRulesDetails(),
                this.loadSubscriptionSources(),
                this.loadSubscriptionStats()
            ]);
            
            // 检查每个任务的结果
            results.forEach((result, index) => {
                const taskNames = ['loadRules', 'loadLogs', 'loadMetrics', 'loadSyncStatus', 'loadLatencyStats', 'loadRulesDetails', 'loadSubscriptionSources', 'loadSubscriptionStats'];
                if (result.status === 'rejected') {
                    console.error(`任务 ${taskNames[index]} 失败:`, result.reason);
                } else {
                    console.log(`任务 ${taskNames[index]} 成功完成`);
                }
            });
            
            // 初始化排序指示器
            this.updateSortIndicators();
            
            console.log('初始数据加载完成');
        } catch (error) {
            console.error('加载数据失败:', error);
            this.showNotification('加载数据失败', 'error');
        }
    }

    // 加载规则
    async loadRules() {
        try {
            const response = await this.apiCall('GET', '/rules');
            if (response.ok) {
                const data = await response.json();
                
                // 填充规则到文本框
                        document.getElementById('china-rules').value = data.china_domains?.join('\n') || '';
        document.getElementById('gfw-rules').value = data.gfw_domains?.join('\n') || '';
        document.getElementById('ad-rules').value = data.ad_domains?.join('\n') || '';
            }
        } catch (error) {
            console.error('加载规则失败:', error);
        }
    }

    // 保存规则
    async saveRules() {
        try {
                    const chinaDomains = document.getElementById('china-rules').value
            .split('\n')
            .map(d => d.trim())
            .filter(d => d && !d.startsWith('#'));

        const gfwDomains = document.getElementById('gfw-rules').value
            .split('\n')
            .map(d => d.trim())
            .filter(d => d && !d.startsWith('#'));

        const adDomains = document.getElementById('ad-rules').value
            .split('\n')
            .map(d => d.trim())
            .filter(d => d && !d.startsWith('#'));

            const response = await this.apiCall('PUT', '/rules', {
                china_domains: chinaDomains,
                gfw_domains: gfwDomains,
                ad_domains: adDomains
            });

            if (response.ok) {
                this.showNotification('规则保存成功', 'success');
                // 自动重载规则
                await this.reloadRules();
            } else {
                throw new Error('保存失败');
            }
        } catch (error) {
            console.error('保存规则失败:', error);
            this.showNotification('保存规则失败', 'error');
        }
    }

    // 重载规则
    async reloadRules() {
        try {
            const response = await this.apiCall('POST', '/reload');
            if (response.ok) {
                this.showNotification('规则重载成功', 'success');
                // 重新加载规则详情
                this.loadRulesDetails();
            } else {
                throw new Error('重载失败');
            }
        } catch (error) {
            console.error('重载规则失败:', error);
            this.showNotification('重载规则失败', 'error');
        }
    }

    // 添加新规则
    async addNewRule() {
        const type = document.getElementById('new-rule-type').value;
        const domain = document.getElementById('new-rule-domain').value.trim();
        
        if (!domain) {
            this.showNotification('请输入域名', 'warning');
            return;
        }
        
        try {
            const response = await this.apiCall('POST', '/rules/add', {
                type: type,
                domain: domain
            });
            
            if (response.ok) {
                const data = await response.json();
                this.showNotification(data.message || '规则添加成功', 'success');
                
                // 清空输入框
                document.getElementById('new-rule-domain').value = '';
                
                // 重新加载规则详情
                this.loadRulesDetails();
                
                // 更新对应的文本框
                this.addRuleToTextarea(type, domain);
            } else {
                throw new Error('添加规则失败');
            }
        } catch (error) {
            console.error('添加规则失败:', error);
            this.showNotification('添加规则失败', 'error');
        }
    }

    // 将新规则添加到对应的文本框
    addRuleToTextarea(type, domain) {
        let textareaId;
        switch (type) {
            case 'china':
                textareaId = 'china-rules';
                break;
            case 'gfw':
                textareaId = 'gfw-rules';
                break;
            case 'ads':
                textareaId = 'ad-rules';
                break;
            default:
                return;
        }
        
        const textarea = document.getElementById(textareaId);
        const currentValue = textarea.value;
        const newValue = currentValue ? currentValue + '\n' + domain : domain;
        textarea.value = newValue;
    }

    // 加载规则详情
    async loadRulesDetails() {
        try {
            const response = await this.apiCall('GET', '/sync/rules');
            if (response.ok) {
                const data = await response.json();
                this.updateRulesDetails(data);
            }
        } catch (error) {
            console.error('加载规则详情失败:', error);
        }
    }

    // 更新规则详情显示
    updateRulesDetails(data) {
        // 更新中国域名
        const chinaDomains = data.china_domains || [];
        document.getElementById('china-rules-count').textContent = chinaDomains.length;
        document.getElementById('china-rules-list').innerHTML = chinaDomains
            .slice(0, 5) // 只显示前5个
            .map(domain => `<div>${domain}</div>`)
            .join('');
        
        // 更新GFW域名
        const gfwDomains = data.gfw_domains || [];
        document.getElementById('gfw-rules-count').textContent = gfwDomains.length;
        document.getElementById('gfw-rules-list').innerHTML = gfwDomains
            .slice(0, 5) // 只显示前5个
            .map(domain => `<div>${domain}</div>`)
            .join('');
        
        // 更新广告域名
        const adDomains = data.ad_domains || [];
        document.getElementById('ads-rules-count').textContent = adDomains.length;
        document.getElementById('ads-rules-list').innerHTML = adDomains
            .slice(0, 5) // 只显示前5个
            .map(domain => `<div>${domain}</div>`)
            .join('');
        
        // 如果规则数量超过5个，显示更多提示
        if (chinaDomains.length > 5) {
            document.getElementById('china-rules-list').innerHTML += '<div class="text-green-600">... 还有更多</div>';
        }
        if (gfwDomains.length > 5) {
            document.getElementById('gfw-rules-list').innerHTML += '<div class="text-blue-600">... 还有更多</div>';
        }
        if (adDomains.length > 5) {
            document.getElementById('ads-rules-list').innerHTML += '<div class="text-red-600">... 还有更多</div>';
        }
    }

    // 加载日志
    async loadLogs() {
        try {
            const response = await this.apiCall('GET', `/logs?limit=1000`); // 获取更多日志用于过滤
            if (response.ok) {
                const data = await response.json();
                this.logState.totalLogs = data.items?.length || 0;
                this.logState.filteredLogs = data.items || [];
                this.filterLogs();
            }
        } catch (error) {
            console.error('加载日志失败:', error);
        }
    }

    // 渲染日志
    renderLogs(logs) {
        const tbody = document.getElementById('log-tbody');
        tbody.innerHTML = '';

        if (logs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" class="px-6 py-4 text-center text-gray-500">暂无日志</td></tr>';
            return;
        }

        logs.forEach(log => {
            const row = document.createElement('tr');
            row.className = 'hover:bg-gray-50';
            
            const time = new Date(log.time).toLocaleString('zh-CN');
            const routeClass = this.getRouteClass(log.route);
            
            // 获取延迟信息（如果有的话）
            const latency = log.latency ? `${log.latency}ms` : '-';
            const latencyClass = this.getLatencyClass(log.latency);
            
            row.innerHTML = `
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">${time}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900 font-mono">${log.name}</td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <span class="inline-flex px-2 py-1 text-xs font-semibold rounded-full ${routeClass}">
                        ${this.getRouteLabel(log.route)}
                    </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    <span class="${latencyClass}">${latency}</span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    <button class="text-blue-600 hover:text-blue-800" onclick="window.BoomDNSUtils.copyToClipboard('${log.name}')">
                        <i class="ti ti-copy"></i>
                    </button>
                </td>
            `;
            
            tbody.appendChild(row);
        });
    }

    // 过滤日志
    filterLogs() {
        let filtered = [...this.logState.filteredLogs];

        // 时间过滤
        if (this.logState.timeFilter !== 'all') {
            const now = new Date();
            let cutoffTime;
            
            switch (this.logState.timeFilter) {
                case '1h':
                    cutoffTime = new Date(now.getTime() - 60 * 60 * 1000);
                    break;
                case '6h':
                    cutoffTime = new Date(now.getTime() - 6 * 60 * 60 * 1000);
                    break;
                case '24h':
                    cutoffTime = new Date(now.getTime() - 24 * 60 * 60 * 1000);
                    break;
                case '7d':
                    cutoffTime = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
                    break;
            }
            
            if (cutoffTime) {
                filtered = filtered.filter(log => new Date(log.time) >= cutoffTime);
            }
        }

        // 路由过滤
        if (this.logState.routeFilter) {
            filtered = filtered.filter(log => log.route === this.logState.routeFilter);
        }

        // 搜索过滤
        if (this.logState.searchTerm) {
            const term = this.logState.searchTerm.toLowerCase();
            filtered = filtered.filter(log => 
                log.name.toLowerCase().includes(term) || 
                this.getRouteLabel(log.route).toLowerCase().includes(term) ||
                (log.latency && log.latency.toString().includes(term))
            );
        }

        // 排序
        this.sortLogs(filtered);

        // 更新分页
        this.logState.currentPage = 1;
        this.updatePagination(filtered.length);
        this.renderCurrentPage(filtered);
    }

    // 排序日志
    sortLogs(logs) {
        logs.sort((a, b) => {
            let aVal, bVal;
            
            switch (this.logState.sortField) {
                case 'time':
                    aVal = new Date(a.time).getTime();
                    bVal = new Date(b.time).getTime();
                    break;
                case 'name':
                    aVal = a.name.toLowerCase();
                    bVal = b.name.toLowerCase();
                    break;
                case 'route':
                    aVal = a.route.toLowerCase();
                    bVal = b.route.toLowerCase();
                    break;
                case 'latency':
                    aVal = a.latency || 0;
                    bVal = b.latency || 0;
                    break;
                default:
                    return 0;
            }
            
            if (this.logState.sortOrder === 'asc') {
                return aVal > bVal ? 1 : -1;
            } else {
                return aVal < bVal ? 1 : -1;
            }
        });
    }

    // 设置排序
    setSort(field) {
        if (this.logState.sortField === field) {
            this.logState.sortOrder = this.logState.sortOrder === 'asc' ? 'desc' : 'asc';
        } else {
            this.logState.sortField = field;
            this.logState.sortOrder = 'desc';
        }
        
        this.filterLogs();
        this.updateSortIndicators();
    }

    // 更新排序指示器
    updateSortIndicators() {
        const headers = ['log-time-header', 'log-domain-header', 'log-route-header', 'log-latency-header'];
        const fields = ['time', 'name', 'route', 'latency'];
        
        headers.forEach((headerId, index) => {
            const header = document.getElementById(headerId);
            if (!header) {
                console.warn(`排序头部元素 ${headerId} 不存在，跳过排序指示器更新`);
                return;
            }
            
            const icon = header.querySelector('i');
            if (!icon) {
                console.warn(`排序图标元素在 ${headerId} 中不存在，跳过排序指示器更新`);
                return;
            }
            
            if (this.logState.sortField === fields[index]) {
                icon.className = this.logState.sortOrder === 'asc' ? 
                    'ti ti-arrow-up text-blue-600' : 'ti ti-arrow-down text-blue-600';
            } else {
                icon.className = 'ti ti-arrows-sort text-gray-400';
            }
        });
    }

    // 渲染当前页
    renderCurrentPage(filteredLogs) {
        const startIndex = (this.logState.currentPage - 1) * this.logState.pageSize;
        const endIndex = startIndex + this.logState.pageSize;
        const pageLogs = filteredLogs.slice(startIndex, endIndex);
        
        this.renderLogs(pageLogs);
        this.updateLogStats(filteredLogs.length, pageLogs.length);
    }

    // 更新分页信息
    updatePagination(totalFiltered) {
        const totalPages = this.getTotalPages(totalFiltered);
        
        const elements = {
            'log-first-page': this.logState.currentPage === 1,
            'log-prev-page': this.logState.currentPage === 1,
            'log-next-page': this.logState.currentPage >= totalPages,
            'log-last-page': this.logState.currentPage >= totalPages
        };
        
        // 安全地更新分页按钮状态
        Object.entries(elements).forEach(([id, disabled]) => {
            const element = document.getElementById(id);
            if (element) {
                element.disabled = disabled;
            } else {
                console.warn(`分页按钮元素 ${id} 不存在`);
            }
        });
        
        // 更新页码信息
        const pageInfoElement = document.getElementById('log-page-info');
        if (pageInfoElement) {
            pageInfoElement.textContent = `${this.logState.currentPage} / ${totalPages}`;
        } else {
            console.warn('页码信息元素 log-page-info 不存在');
        }
    }

    // 获取总页数
    getTotalPages(totalFiltered = null) {
        const total = totalFiltered !== null ? totalFiltered : this.logState.totalLogs;
        return Math.ceil(total / this.logState.pageSize);
    }

    // 跳转到指定页
    goToPage(page) {
        const totalPages = this.getTotalPages();
        if (page < 1 || page > totalPages) return;
        
        this.logState.currentPage = page;
        this.filterLogs();
    }

    // 更新日志统计信息
    updateLogStats(totalFiltered, currentPageSize) {
        const elements = {
            'log-total': this.logState.totalLogs,
            'log-current-page': this.logState.currentPage,
            'log-total-pages': this.getTotalPages(totalFiltered),
            'log-showing': currentPageSize
        };
        
        // 安全地更新元素
        Object.entries(elements).forEach(([id, value]) => {
            const element = document.getElementById(id);
            if (element) {
                element.textContent = value;
            } else {
                console.warn(`日志统计元素 ${id} 不存在`);
            }
        });
    }

    // 导出日志
    exportLogs() {
        try {
            const logs = this.logState.filteredLogs;
            if (logs.length === 0) {
                this.showNotification('没有日志可导出', 'warning');
                return;
            }
            
            const csvContent = this.convertLogsToCSV(logs);
            const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `boomdns-logs-${new Date().toISOString().split('T')[0]}.csv`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
            
            this.showNotification('日志导出成功', 'success');
        } catch (error) {
            console.error('导出日志失败:', error);
            this.showNotification('导出日志失败', 'error');
        }
    }

    // 转换日志为CSV格式
    convertLogsToCSV(logs) {
        const headers = ['时间', '域名', '路由', '延迟(ms)'];
        const rows = logs.map(log => [
            new Date(log.time).toLocaleString('zh-CN'),
            log.name,
            this.getRouteLabel(log.route),
            log.latency || '-'
        ]);
        
        return [headers, ...rows]
            .map(row => row.map(cell => `"${cell}"`).join(','))
            .join('\n');
    }

    // 防抖搜索
    debounceSearch() {
        if (this.searchTimeout) {
            clearTimeout(this.searchTimeout);
        }
        
        this.searchTimeout = setTimeout(() => {
            this.filterLogs();
        }, 300); // 300ms延迟
    }

    // 获取路由样式类
    getRouteClass(route) {
        switch (route) {
            case 'china':
                return 'bg-green-100 text-green-800';
            case 'intl':
                return 'bg-blue-100 text-blue-800';
            case 'adguard':
                return 'bg-red-100 text-red-800';
            case 'cache':
                return 'bg-yellow-100 text-yellow-800';
            default:
                return 'bg-gray-100 text-gray-800';
        }
    }

    // 获取延迟样式类
    getLatencyClass(latency) {
        if (!latency) return 'text-gray-500';
        
        if (latency < 50) {
            return 'text-green-600 font-semibold'; // 优秀
        } else if (latency < 100) {
            return 'text-blue-600 font-semibold'; // 良好
        } else if (latency < 200) {
            return 'text-orange-600 font-semibold'; // 一般
        } else {
            return 'text-red-600 font-semibold'; // 较差
        }
    }

    // 获取路由标签
    getRouteLabel(route) {
        const labels = {
            'china': '中国',
            'intl': '国际',
            'adguard': '广告拦截'
        };
        return labels[route] || route;
    }

    // 加载指标数据
    async loadMetrics() {
        try {
            const response = await this.apiCall('GET', '/metrics');
            if (response.ok) {
                const data = await response.json();
                this.updateMetrics(data);
            } else {
                throw new Error('获取指标失败');
            }
        } catch (error) {
            console.error('加载指标失败:', error);
            // 如果API失败，显示0
            this.updateMetrics({
                total_queries: 0,
                china_queries: 0,
                intl_queries: 0,
                adguard_queries: 0
            });
        }
    }

    // 更新指标显示
    updateMetrics(metrics) {
        const elements = {
            'total-queries': metrics.total_queries || 0,
            'china-queries': metrics.china_queries || 0,
            'intl-queries': metrics.intl_queries || 0,
            'ad-queries': metrics.adguard_queries || 0,
            'cache-queries': metrics.cache_queries || 0
        };
        
        // 安全地更新元素
        Object.entries(elements).forEach(([id, value]) => {
            const element = document.getElementById(id);
            if (element) {
                element.textContent = value;
            } else {
                console.warn(`元素 ${id} 不存在`);
            }
        });
        
        // 更新缓存统计
        if (metrics.cache_stats) {
            this.updateCacheStats(metrics.cache_stats);
        }
        
        // 更新延迟统计
        if (metrics.latency_stats) {
            this.updateLatencyStats(metrics.latency_stats);
        }
    }

    // 更新延迟统计显示
    updateLatencyStats(latencyStats) {
        // 更新延迟范围显示
        const minLatency = latencyStats.min_latency_ms || 0;
        const maxLatency = latencyStats.max_latency_ms || 0;
        const avgLatency = latencyStats.avg_latency_ms || 0;
        
        const elements = {
            'avg-latency': avgLatency ? `${avgLatency}ms` : '-',
            'latency-range': minLatency && maxLatency ? `${minLatency}-${maxLatency}ms` : '-',
            'total-queries-latency': latencyStats.total_queries || 0,
            'min-latency': minLatency ? `${minLatency}ms` : '-',
            'max-latency': maxLatency ? `${maxLatency}ms` : '-',
            'total-latency': latencyStats.total_latency_ms ? `${latencyStats.total_latency_ms}ms` : '-'
        };
        
        // 安全地更新元素
        Object.entries(elements).forEach(([id, value]) => {
            const element = document.getElementById(id);
            if (element) {
                element.textContent = value;
            } else {
                console.warn(`延迟统计元素 ${id} 不存在`);
            }
        });
        
        // 更新路由延迟统计表格
        this.updateRouteLatencyTable(latencyStats.route_stats || {});
    }

    // 更新缓存统计显示
    updateCacheStats(cacheStats) {
        const elements = {
            'cache-hits': cacheStats.hits || 0,
            'cache-misses': cacheStats.misses || 0,
            'cache-hit-rate': (cacheStats.hit_rate || 0).toFixed(1),
            'cache-entries': cacheStats.entries || 0,
            'cache-hit-rate-display': (cacheStats.hit_rate || 0).toFixed(1)
        };
        
        // 安全地更新元素
        Object.entries(elements).forEach(([id, value]) => {
            const element = document.getElementById(id);
            if (element) {
                element.textContent = value;
            } else {
                console.warn(`缓存统计元素 ${id} 不存在`);
            }
        });
    }

    // 加载缓存统计
    async loadCacheStats() {
        try {
            const response = await this.apiCall('GET', '/cache/stats');
            if (response.ok) {
                const data = await response.json();
                this.updateCacheStats(data);
                this.showNotification('缓存统计已更新', 'success');
            } else {
                throw new Error('获取缓存统计失败');
            }
        } catch (error) {
            console.error('加载缓存统计失败:', error);
            this.showNotification('加载缓存统计失败', 'error');
        }
    }

    // 清空缓存
    async clearCache() {
        if (!confirm('确定要清空所有缓存吗？此操作不可恢复。')) {
            return;
        }

        try {
            const response = await this.apiCall('DELETE', '/cache');
            if (response.ok) {
                this.showNotification('缓存已清空', 'success');
                // 重新加载缓存统计
                this.loadCacheStats();
            } else {
                throw new Error('清空缓存失败');
            }
        } catch (error) {
            console.error('清空缓存失败:', error);
            this.showNotification('清空缓存失败', 'error');
        }
    }

    // 立即同步规则
    async syncNow() {
        if (!confirm('确定要立即同步规则吗？这可能需要一些时间。')) {
            return;
        }

        try {
            const response = await this.apiCall('POST', '/sync/now');
            if (response.ok) {
                this.showNotification('规则同步已触发', 'success');
                // 重新加载同步状态
                this.loadSyncStatus();
            } else {
                throw new Error('触发同步失败');
            }
        } catch (error) {
            console.error('触发同步失败:', error);
            this.showNotification('触发同步失败', 'error');
        }
    }

    // 加载同步状态
    async loadSyncStatus() {
        try {
            const response = await this.apiCall('GET', '/sync/status');
            if (response.ok) {
                const data = await response.json();
                this.updateSyncStatus(data);
                this.showNotification('同步状态已更新', 'success');
            } else {
                throw new Error('获取同步状态失败');
            }
        } catch (error) {
            console.error('加载同步状态失败:', error);
            this.showNotification('加载同步状态失败', 'error');
        }
    }

    // 更新同步状态显示
    updateSyncStatus(syncData) {
        // 更新状态显示
        document.getElementById('sync-status').textContent = syncData.status || 'unknown';
        document.getElementById('last-sync').textContent = this.formatTime(syncData.last_sync);
        document.getElementById('next-sync').textContent = this.formatTime(syncData.next_sync);
        document.getElementById('sync-success-rate').textContent = (syncData.success_rate || 0).toFixed(1);
        
        // 设置状态颜色
        const statusEl = document.getElementById('sync-status');
        if (syncData.status === 'running') {
            statusEl.className = 'text-2xl font-bold text-green-600';
        } else if (syncData.status === 'error') {
            statusEl.className = 'text-2xl font-bold text-red-600';
        } else {
            statusEl.className = 'text-2xl font-bold text-gray-600';
        }
    }

    // 格式化时间显示
    formatTime(timestamp) {
        if (!timestamp) return '-';
        
        try {
            const date = new Date(timestamp);
            if (isNaN(date.getTime())) return '-';
            
            const now = new Date();
            const diff = now - date;
            
            if (diff < 60000) { // 1分钟内
                return '刚刚';
            } else if (diff < 3600000) { // 1小时内
                const minutes = Math.floor(diff / 60000);
                return `${minutes}分钟前`;
            } else if (diff < 86400000) { // 1天内
                const hours = Math.floor(diff / 3600000);
                return `${hours}小时前`;
            } else {
                return date.toLocaleDateString('zh-CN');
            }
        } catch (error) {
            return '-';
        }
    }

    // 清空日志
    async clearLogs() {
        if (!confirm('确定要清空所有日志吗？此操作不可恢复。')) {
            return;
        }
        
        try {
            const response = await this.apiCall('DELETE', '/logs');
            if (response.ok) {
                this.showNotification('日志清空成功', 'success');
                this.loadLogs(); // 重新加载
            } else {
                throw new Error('清空日志失败');
            }
        } catch (error) {
            console.error('清空日志失败:', error);
            this.showNotification('清空日志失败', 'error');
        }
    }

    // 加载延迟统计
    async loadLatencyStats() {
        try {
            const response = await this.apiCall('GET', '/latency/stats');
            if (response.ok) {
                const data = await response.json();
                this.updateLatencyStats(data);
            } else {
                throw new Error('获取延迟统计失败');
            }
        } catch (error) {
            console.error('加载延迟统计失败:', error);
            this.showNotification('加载延迟统计失败', 'error');
        }
    }

    // 更新路由延迟统计表格
    updateRouteLatencyTable(routeStats) {
        const tbody = document.getElementById('route-latency-tbody');
        tbody.innerHTML = '';
        
        if (Object.keys(routeStats).length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="px-4 py-2 text-center text-gray-500">暂无数据</td></tr>';
            return;
        }
        
        Object.entries(routeStats).forEach(([route, stats]) => {
            const row = document.createElement('tr');
            row.className = 'hover:bg-gray-50';
            
            const routeLabel = this.getRouteLabel(route);
            const routeClass = this.getRouteClass(route);
            
            row.innerHTML = `
                <td class="px-4 py-2 whitespace-nowrap">
                    <span class="inline-flex px-2 py-1 text-xs font-semibold rounded-full ${routeClass}">
                        ${routeLabel}
                    </span>
                </td>
                <td class="px-4 py-2 whitespace-nowrap text-sm text-gray-900">${stats.count}</td>
                <td class="px-4 py-2 whitespace-nowrap text-sm text-gray-900">${stats.avg_latency_ms}ms</td>
                <td class="px-4 py-2 whitespace-nowrap text-sm text-gray-900">${stats.min_latency_ms}ms</td>
                <td class="px-4 py-2 whitespace-nowrap text-sm text-gray-900">${stats.max_latency_ms}ms</td>
                <td class="px-4 py-2 whitespace-nowrap text-sm text-gray-500">${this.formatTime(stats.last_updated)}</td>
            `;
            
            tbody.appendChild(row);
        });
    }

    // 重置延迟统计
    async resetLatencyStats() {
        if (!confirm('确定要重置所有延迟统计吗？此操作不可恢复。')) {
            return;
        }
        
        try {
            // 这里可以调用重置API，暂时显示提示
            this.showNotification('延迟统计重置功能正在开发中', 'info');
        } catch (error) {
            console.error('重置延迟统计失败:', error);
            this.showNotification('重置延迟统计失败', 'error');
        }
    }

    // 导出配置
    exportConfig() {
        try {
            const config = {
                china_domains: document.getElementById('china-rules').value.split('\n').filter(d => d.trim()),
                gfw_domains: document.getElementById('gfw-rules').value.split('\n').filter(d => d.trim()),
                ad_domains: document.getElementById('ad-rules').value.split('\n').filter(d => d.trim())
            };

            const blob = new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `boomdns-rules-${new Date().toISOString().split('T')[0]}.json`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);

            this.showNotification('配置导出成功', 'success');
        } catch (error) {
            console.error('导出配置失败:', error);
            this.showNotification('导出配置失败', 'error');
        }
    }

    // 检查健康状态
    async checkHealth() {
        try {
            const response = await this.apiCall('GET', '/health');
            if (response.ok) {
                this.updateStatus('online');
            } else {
                this.updateStatus('offline');
            }
        } catch (error) {
            this.updateStatus('offline');
        }
    }

    // 更新状态显示
    updateStatus(status) {
        const indicator = document.getElementById('status-indicator');
        const statusText = document.getElementById('status-text');
        
        if (status === 'online') {
            indicator.querySelector('span').className = 'w-2 h-2 bg-green-400 rounded-full animate-pulse';
            statusText.textContent = '在线';
            statusText.className = 'status-online';
        } else {
            indicator.querySelector('span').className = 'w-2 h-2 bg-red-400 rounded-full';
            statusText.textContent = '离线';
            statusText.className = 'status-offline';
        }
    }

    // 显示通知
    showNotification(message, type = 'info') {
        const notification = document.getElementById('notification');
        const icon = document.getElementById('notification-icon');
        const messageEl = document.getElementById('notification-message');

        // 设置图标和颜色
        const icons = {
            'success': 'ti ti-check text-green-400',
            'error': 'ti ti-x text-red-400',
            'warning': 'ti ti-alert-triangle text-yellow-400',
            'info': 'ti ti-info-circle text-blue-400'
        };

        icon.className = icons[type] || icons.info;
        messageEl.textContent = message;

        // 显示通知
        notification.classList.remove('translate-x-full');
        notification.classList.add('translate-x-0');

        // 5秒后自动隐藏
        setTimeout(() => {
            this.hideNotification();
        }, 5000);
    }

    // 隐藏通知
    hideNotification() {
        const notification = document.getElementById('notification');
        notification.classList.remove('translate-x-0');
        notification.classList.add('translate-x-full');
    }

    // API调用
    async apiCall(method, endpoint, data = null) {
        const url = `${this.config.apiBase}${endpoint}`;
        const options = {
            method,
            headers: {
                'Content-Type': 'application/json'
            }
        };

        if (this.config.adminToken) {
            options.headers['Authorization'] = `Bearer ${this.config.adminToken}`;
        }

        if (data) {
            options.body = JSON.stringify(data);
        }

        return fetch(url, options);
    }

    // 开始自动刷新
    startAutoRefresh() {
        setInterval(() => {
            this.loadMetrics();
            this.loadLatencyStats();
            // 只在第一页时自动刷新日志，避免影响用户浏览
            if (this.logState.currentPage === 1) {
                this.loadLogs();
            }
        }, this.config.refreshInterval);
    }
    
    // ==================== 订阅源管理方法 ====================
    
    // 加载订阅源列表
    async loadSubscriptionSources() {
        try {
            const response = await this.apiCall('GET', '/subscriptions/sources');
            if (response.ok) {
                const data = await response.json();
                this.renderSubscriptionSources(data.data || []);
            }
        } catch (error) {
            console.error('加载订阅源失败:', error);
        }
    }
    
    // 加载订阅源统计
    async loadSubscriptionStats() {
        try {
            const response = await this.apiCall('GET', '/subscriptions/stats');
            if (response.ok) {
                const data = await response.json();
                this.updateSubscriptionStats(data);
            }
        } catch (error) {
            console.error('加载订阅源统计失败:', error);
        }
    }
    
    // 渲染订阅源列表
    renderSubscriptionSources(sources) {
        const container = document.getElementById('subscription-sources-list');
        if (!container) return;
        
        if (sources.length === 0) {
            container.innerHTML = '<div class="text-center text-gray-500 py-8">暂无订阅源</div>';
            return;
        }
        
        const html = sources.map(source => `
            <div class="bg-gray-50 p-4 rounded-lg border border-gray-200">
                <div class="flex justify-between items-start">
                    <div class="flex-1">
                        <div class="flex items-center space-x-2 mb-2">
                            <span class="px-2 py-1 text-xs font-medium rounded-full ${
                                source.enabled ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                            }">
                                ${source.enabled ? '已启用' : '已禁用'}
                            </span>
                            <span class="px-2 py-1 text-xs font-medium rounded-full bg-blue-100 text-blue-800">
                                ${source.category}
                            </span>
                            <span class="px-2 py-1 text-xs font-medium rounded-full bg-purple-100 text-purple-800">
                                ${source.format}
                            </span>
                        </div>
                        <h5 class="font-medium text-gray-900">${source.name}</h5>
                        <p class="text-sm text-gray-600 mt-1">${source.url}</p>
                        <div class="text-xs text-gray-500 mt-2">
                            最后更新: ${source.last_update ? new Date(source.last_update * 1000).toLocaleString() : '从未更新'}
                        </div>
                    </div>
                    <div class="flex space-x-2 ml-4">
                        <button class="text-blue-600 hover:text-blue-800 p-1" onclick="window.app.editSubscriptionSource(${source.id})">
                            <i class="ti ti-edit"></i>
                        </button>
                        <button class="text-green-600 hover:text-green-800 p-1" onclick="window.app.testSubscriptionSource(${source.id})">
                            <i class="ti ti-refresh"></i>
                        </button>
                        <button class="text-red-600 hover:text-red-800 p-1" onclick="window.app.deleteSubscriptionSource(${source.id})">
                            <i class="ti ti-trash"></i>
                        </button>
                    </div>
                </div>
            </div>
        `).join('');
        
        container.innerHTML = html;
    }
    
    // 更新订阅源统计
    updateSubscriptionStats(stats) {
        const totalSources = document.getElementById('total-sources');
        const enabledSources = document.getElementById('enabled-sources');
        const totalRules = document.getElementById('total-rules');
        
        if (totalSources) totalSources.textContent = stats.total_sources || 0;
        if (enabledSources) enabledSources.textContent = stats.enabled_sources || 0;
        if (totalRules) totalRules.textContent = stats.total_rules || 0;
    }
    
    // 显示添加订阅源模态框
    showAddSourceModal() {
        const modal = document.getElementById('subscription-source-modal');
        const form = document.getElementById('subscription-source-form');
        const title = document.getElementById('modal-title');
        
        // 重置表单
        form.reset();
        document.getElementById('source-id').value = '';
        title.textContent = '添加订阅源';
        
        // 显示模态框
        modal.classList.remove('hidden');
    }
    
    // 隐藏订阅源模态框
    hideSubscriptionSourceModal() {
        const modal = document.getElementById('subscription-source-modal');
        modal.classList.add('hidden');
    }
    
    // 保存订阅源
    async saveSubscriptionSource() {
        try {
            const form = document.getElementById('subscription-source-form');
            const formData = new FormData(form);
            
            const sourceData = {
                name: document.getElementById('source-name').value,
                category: document.getElementById('source-category').value,
                url: document.getElementById('source-url').value,
                format: document.getElementById('source-format').value,
                enabled: document.getElementById('source-enabled').checked
            };
            
            // 验证必填字段
            if (!sourceData.name || !sourceData.category || !sourceData.url || !sourceData.format) {
                this.showNotification('请填写所有必填字段', 'warning');
                return;
            }
            
            const sourceId = document.getElementById('source-id').value;
            const method = sourceId ? 'PUT' : 'POST';
            const endpoint = sourceId ? `/subscriptions/sources/${sourceId}` : '/subscriptions/sources';
            
            const response = await this.apiCall(method, endpoint, sourceData);
            if (response.ok) {
                const data = await response.json();
                this.showNotification(data.message || '订阅源保存成功', 'success');
                this.hideSubscriptionSourceModal();
                await this.loadSubscriptionSources();
                await this.loadSubscriptionStats();
            } else {
                throw new Error('保存失败');
            }
        } catch (error) {
            console.error('保存订阅源失败:', error);
            this.showNotification('保存订阅源失败', 'error');
        }
    }
    
    // 编辑订阅源
    editSubscriptionSource(id) {
        // 这里需要先获取订阅源详情，然后填充到表单中
        // 暂时显示编辑模态框
        this.showAddSourceModal();
        document.getElementById('modal-title').textContent = '编辑订阅源';
        document.getElementById('source-id').value = id;
    }
    
    // 测试订阅源
    async testSubscriptionSource(id) {
        try {
            const response = await this.apiCall('POST', `/subscriptions/sources/${id}/test`);
            if (response.ok) {
                const data = await response.json();
                this.showNotification(data.message || '订阅源测试成功', 'success');
            } else {
                throw new Error('测试失败');
            }
        } catch (error) {
            console.error('测试订阅源失败:', error);
            this.showNotification('测试订阅源失败', 'error');
        }
    }
    
    // 删除订阅源
    async deleteSubscriptionSource(id) {
        if (!confirm('确定要删除这个订阅源吗？')) {
            return;
        }
        
        try {
            const response = await this.apiCall('DELETE', `/subscriptions/sources/${id}`);
            if (response.ok) {
                const data = await response.json();
                this.showNotification(data.message || '订阅源删除成功', 'success');
                await this.loadSubscriptionSources();
                await this.loadSubscriptionStats();
            } else {
                throw new Error('删除失败');
            }
        } catch (error) {
            console.error('删除订阅源失败:', error);
            this.showNotification('删除订阅源失败', 'error');
        }
    }
}

// 页面加载完成后初始化应用
document.addEventListener('DOMContentLoaded', () => {
    window.app = new BoomDNSApp();
});

// 全局工具函数
window.BoomDNSUtils = {
    // 复制到剪贴板
    copyToClipboard: function(text) {
        if (navigator.clipboard) {
            navigator.clipboard.writeText(text).then(() => {
                // 可以添加一个临时的成功提示
                console.log('已复制到剪贴板:', text);
            }).catch(err => {
                console.error('复制失败:', err);
            });
        } else {
            // 降级方案
            const textArea = document.createElement('textarea');
            textArea.value = text;
            document.body.appendChild(textArea);
            textArea.select();
            try {
                document.execCommand('copy');
                console.log('已复制到剪贴板:', text);
            } catch (err) {
                console.error('复制失败:', err);
            }
            document.body.removeChild(textArea);
        }
    }
};
