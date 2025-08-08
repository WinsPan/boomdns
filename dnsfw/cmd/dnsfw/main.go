package main

import (
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

	"github.com/winspan/dnsfw/admin"
	"github.com/winspan/dnsfw/dns"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := dns.LoadConfig(configPath)
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
	admin.BindRoutes(r, server, cfg)
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
			_ = httpSrv.Close()
			_ = udpConn.Close()
			_ = tcpLn.Close()
			return
		}
	}
}
