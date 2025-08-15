#!/usr/bin/env bash

echo "🔍 BoomDNS 数据功能验证"
echo "========================"

# 检查服务状态
echo "1. 检查服务状态..."
if ps aux | grep boomdns | grep -v grep > /dev/null; then
    echo "   ✅ BoomDNS 服务正在运行"
else
    echo "   ❌ BoomDNS 服务未运行"
    exit 1
fi

# 检查数据库文件
echo "2. 检查数据库文件..."
if [ -f "data/boomdns.db" ]; then
    echo "   ✅ SQLite 数据库文件存在"
    db_size=$(ls -lh data/boomdns.db | awk '{print $5}')
    echo "   📊 数据库大小: $db_size"
else
    echo "   ❌ SQLite 数据库文件不存在"
fi

# 检查数据库表
echo "3. 检查数据库表结构..."
tables=$(sqlite3 data/boomdns.db ".tables" 2>/dev/null)
if [ $? -eq 0 ]; then
    echo "   ✅ 数据库表结构正常"
    echo "   📋 表列表: $tables"
else
    echo "   ❌ 无法访问数据库"
fi

# 检查缓存数据
echo "4. 检查缓存数据..."
cache_count=$(sqlite3 data/boomdns.db "SELECT COUNT(*) FROM dns_cache;" 2>/dev/null)
if [ $? -eq 0 ]; then
    echo "   ✅ 缓存表正常"
    echo "   📊 缓存条目数: $cache_count"
    
    if [ "$cache_count" -gt 0 ]; then
        echo "   📋 缓存内容:"
        sqlite3 data/boomdns.db "SELECT key, expire_at FROM dns_cache;" 2>/dev/null | while read line; do
            echo "      - $line"
        done
    fi
else
    echo "   ❌ 无法查询缓存数据"
fi

# 检查查询日志
echo "5. 检查查询日志..."
logs_count=$(sqlite3 data/boomdns.db "SELECT COUNT(*) FROM query_logs;" 2>/dev/null)
if [ $? -eq 0 ]; then
    echo "   ✅ 查询日志表正常"
    echo "   📊 日志条目数: $logs_count"
    
    if [ "$logs_count" -gt 0 ]; then
        echo "   📋 最近查询:"
        sqlite3 data/boomdns.db "SELECT name, route, latency FROM query_logs ORDER BY timestamp DESC LIMIT 5;" 2>/dev/null | while read line; do
            echo "      - $line"
        done
    fi
else
    echo "   ❌ 无法查询日志数据"
fi

# 检查统计信息
echo "6. 检查统计信息..."
stats_count=$(sqlite3 data/boomdns.db "SELECT COUNT(*) FROM stats;" 2>/dev/null)
if [ $? -eq 0 ]; then
    echo "   ✅ 统计信息表正常"
    echo "   📊 统计条目数: $stats_count"
else
    echo "   ❌ 无法查询统计信息"
fi

# 测试 DNS 查询
echo "7. 测试 DNS 查询功能..."
echo "   查询 google.com:"
dig @127.0.0.1 google.com +short 2>/dev/null | head -2

echo "   查询 baidu.com:"
dig @127.0.0.1 baidu.com +short 2>/dev/null | head -2

# 检查 HTTP 管理界面
echo "8. 检查 HTTP 管理界面..."
if curl -s http://127.0.0.1:8080/ > /dev/null; then
    echo "   ✅ HTTP 管理界面可访问"
    echo "   🌐 访问地址: http://127.0.0.1:8080"
else
    echo "   ❌ HTTP 管理界面无法访问"
fi

echo ""
echo "🎉 数据功能验证完成！"
echo "📊 当前状态:"
echo "   - DNS 服务: 运行中"
echo "   - 数据库: SQLite (${cache_count} 缓存, ${logs_count} 日志)"
echo "   - Web 界面: http://127.0.0.1:8080"
