package dns

import (
    "os"
    "gopkg.in/yaml.v3"
)

type Upstream struct {
    Address   string `yaml:"address"`
    ViaSocks5 string `yaml:"socks5"`
}

type Config struct {
    ListenDNS  string     `yaml:"listen_dns"`
    ListenHTTP string     `yaml:"listen_http"`
    AdminToken string     `yaml:"admin_token"`

    ChinaUpstreams []Upstream `yaml:"china_upstreams"`
    IntlUpstreams  []Upstream `yaml:"intl_upstreams"`
    AdguardAddr    string     `yaml:"adguard_addr"`

    ChinaDomains []string `yaml:"china_domains"`
    GfwDomains   []string `yaml:"gfw_domains"`
    AdDomains    []string `yaml:"ad_domains"`

    // 规则订阅源与刷新周期（可选）
    RuleSync struct {
        RefreshInterval string   `yaml:"refresh_interval"` // 例如 "6h"
        GFWListURLs     []string `yaml:"gfwlist_urls"`
        ChinaListURLs   []string `yaml:"china_list_urls"`  // felixonmars accelerated-domains.china.conf
        AdListURLs      []string `yaml:"adlist_urls"`      // hosts 或域名列表
    } `yaml:"rule_sync"`
}

func LoadConfig(path string) (*Config, error) {
    b, err := os.ReadFile(path)
    if err != nil { return nil, err }
    var c Config
    if err := yaml.Unmarshal(b, &c); err != nil { return nil, err }
    if c.ListenDNS == "" { c.ListenDNS = ":53" }
    if c.ListenHTTP == "" { c.ListenHTTP = ":8080" }
    return &c, nil
}


