#!/usr/bin/env bash

echo "🚀 BoomDNS 简单功能测试"
echo "=========================="

# 测试 DNS 服务
echo "1. 测试 DNS 服务..."
if dig @127.0.0.1 google.com +short >/dev/null 2>&1; then
    echo "   ✅ DNS 服务正常"
else
    echo "   ❌ DNS 服务异常"
fi

# 测试 HTTP 管理接口
echo "2. 测试 HTTP 管理接口..."
if curl -s http://127.0.0.1:8080/ >/dev/null; then
    echo "   ✅ HTTP 管理接口正常"
else
    echo "   ❌ HTTP 管理接口异常"
fi

# 测试智能路由
echo "3. 测试智能路由..."
echo "   中国域名 (baidu.com):"
dig @127.0.0.1 baidu.com +short | head -2

echo "   国际域名 (github.com):"
dig @127.0.0.1 github.com +short | head -2

echo "   广告域名 (doubleclick.net):"
dig @127.0.0.1 doubleclick.net +short | head -2

# 测试缓存功能
echo "4. 测试缓存功能..."
echo "   第一次查询 (缓存未命中):"
time dig @127.0.0.1 google.com +short >/dev/null 2>&1

echo "   第二次查询 (缓存命中):"
time dig @127.0.0.1 google.com +short >/dev/null 2>&1

echo "5. 测试完成！"
echo "   管理界面: http://127.0.0.1:8080"
echo "   DNS 服务: 127.0.0.1:53"
