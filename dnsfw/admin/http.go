package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/winspan/dnsfw/dns"
)

type Api struct {
	srv   *dns.Server
	cfg   *dns.Config
	token string
}

func BindRoutes(r *chi.Mux, srv *dns.Server, cfg *dns.Config) {
	api := &Api{srv: srv, cfg: cfg, token: cfg.AdminToken}
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, middleware.Timeout(10*time.Second))
	r.Get("/api/health", api.health)
	r.Group(func(pr chi.Router) {
		pr.Use(api.auth)
		pr.Post("/api/reload", api.reload)
		pr.Get("/api/rules", api.getRules)
		pr.Put("/api/rules", api.putRules)
		pr.Get("/api/logs", api.getLogs)
	})
}

func (a *Api) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") || a.token == "" || strings.TrimPrefix(h, "Bearer ") != a.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *Api) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (a *Api) reload(w http.ResponseWriter, r *http.Request) {
	if err := a.srv.ReloadRules(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *Api) getRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	data := map[string]any{
		"china_domains": a.cfg.ChinaDomains,
		"gfw_domains":   a.cfg.GfwDomains,
		"ad_domains":    a.cfg.AdDomains,
	}
	_ = json.NewEncoder(w).Encode(data)
}

func (a *Api) putRules(w http.ResponseWriter, r *http.Request) {
	var body struct {
		China []string `json:"china_domains"`
		Gfw   []string `json:"gfw_domains"`
		Ads   []string `json:"ad_domains"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.China != nil {
		a.cfg.ChinaDomains = body.China
	}
	if body.Gfw != nil {
		a.cfg.GfwDomains = body.Gfw
	}
	if body.Ads != nil {
		a.cfg.AdDomains = body.Ads
	}
	if err := a.srv.ReloadRules(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *Api) getLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	items := a.srv.GetLogs(200)
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
}
