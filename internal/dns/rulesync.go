package dns

// 规则远程订阅与转换：
// - 支持从 Loyalsoldier geosite/geoip、gfwlist 拉取
// - 将域名规则转成 Config 的 ChinaDomains/GfwDomains/AdDomains
// - 提供 SyncRules(ctx) 定时任务（留待接线）

// 说明：
// 原型先占位接口，实际实现需：
// 1) 使用 http 客户端下载发布资源
// 2) 解析 geosite.dat（可引入第三方库或改用提供的纯文本派生清单）
// 3) gfwlist base64 解码并提取域名
// 4) 合并/去重/最小化后写入内存并触发 ReloadRules()


