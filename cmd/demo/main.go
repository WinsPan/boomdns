package main

import (
	"fmt"
	"log"
	"time"

	"github.com/winspan/boomdns/dns"
)

func main() {
	fmt.Println("=== BoomDNS SQLite 功能演示 ===")

	// 创建测试配置
	config := &dns.Config{}
	config.Persistence.Enabled = true
	config.Persistence.DataDir = "./demo-data"
	config.Persistence.Database.Type = "sqlite"
	config.Persistence.Database.SQLiteFile = "demo.db"
	config.Persistence.AutoSaveInterval = 10
	config.Persistence.MaxLogs = 100
	config.Persistence.MaxCacheEntries = 100

	// 创建 SQLite 管理器
	manager, err := dns.NewSQLiteManager(config)
	if err != nil {
		log.Fatalf("创建 SQLite 管理器失败: %v", err)
	}
	defer manager.Close()

	fmt.Println("✓ SQLite 管理器已创建")

	// 演示缓存操作
	fmt.Println("\n--- 缓存操作演示 ---")

	// 创建测试缓存数据
	cache := make(map[string]*dns.CacheEntry)
	cache["example.com:A"] = &dns.CacheEntry{
		Response: nil, // 简化演示
		ExpireAt: time.Now().Add(time.Hour),
	}
	cache["google.com:A"] = &dns.CacheEntry{
		Response: nil,
		ExpireAt: time.Now().Add(2 * time.Hour),
	}

	// 保存缓存
	if err := manager.SaveCache(cache); err != nil {
		log.Printf("保存缓存失败: %v", err)
	} else {
		fmt.Println("✓ 缓存已保存")
	}

	// 加载缓存
	loadedCache, err := manager.LoadCache()
	if err != nil {
		log.Printf("加载缓存失败: %v", err)
	} else {
		fmt.Printf("✓ 缓存已加载，共 %d 个条目\n", len(loadedCache))
	}

	// 演示日志操作
	fmt.Println("\n--- 日志操作演示 ---")

	// 创建测试日志数据
	logs := []dns.QueryLog{
		{
			Name:    "example.com",
			Route:   "china",
			Latency: 50,
			Time:    time.Now(),
		},
		{
			Name:    "google.com",
			Route:   "intl",
			Latency: 120,
			Time:    time.Now(),
		},
	}

	// 保存日志
	if err := manager.SaveLogs(logs); err != nil {
		log.Printf("保存日志失败: %v", err)
	} else {
		fmt.Println("✓ 日志已保存")
	}

	// 加载日志
	loadedLogs, err := manager.LoadLogs()
	if err != nil {
		log.Printf("加载日志失败: %v", err)
	} else {
		fmt.Printf("✓ 日志已加载，共 %d 条\n", len(loadedLogs))
	}

	// 演示规则操作
	fmt.Println("\n--- 规则操作演示 ---")

	// 创建测试规则数据
	rules := map[string][]string{
		"china": {"baidu.com", "qq.com", "taobao.com"},
		"gfw":   {"google.com", "facebook.com", "twitter.com"},
		"ads":   {"doubleclick.net", "googlesyndication.com"},
	}

	// 保存规则
	if err := manager.SaveRules(rules); err != nil {
		log.Printf("保存规则失败: %v", err)
	} else {
		fmt.Println("✓ 规则已保存")
	}

	// 加载规则
	loadedRules, err := manager.LoadRules()
	if err != nil {
		log.Printf("加载规则失败: %v", err)
	} else {
		fmt.Printf("✓ 规则已加载，共 %d 个类型\n", len(loadedRules))
		for ruleType, domains := range loadedRules {
			fmt.Printf("  - %s: %d 个域名\n", ruleType, len(domains))
		}
	}

	// 演示统计查询
	fmt.Println("\n--- 统计查询演示 ---")

	stats, err := manager.GetQueryStats()
	if err != nil {
		log.Printf("获取统计失败: %v", err)
	} else {
		fmt.Println("✓ 统计信息已获取:")
		for name, value := range stats {
			fmt.Printf("  - %s: %v\n", name, value)
		}
	}

	// 演示数据清理
	fmt.Println("\n--- 数据清理演示 ---")

	if err := manager.CleanupOldData(); err != nil {
		log.Printf("清理数据失败: %v", err)
	} else {
		fmt.Println("✓ 旧数据已清理")
	}

	fmt.Println("\n=== 演示完成 ===")
	fmt.Println("数据库文件位置: ./demo-data/demo.db")
	fmt.Println("可以使用以下命令查看数据库内容:")
	fmt.Println("sqlite3 ./demo-data/demo.db")
	fmt.Println(".tables")
	fmt.Println("SELECT * FROM dns_cache;")
	fmt.Println("SELECT * FROM query_logs;")
	fmt.Println("SELECT * FROM dns_rules;")
}
