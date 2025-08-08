package admin

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/winspan/dnsfw/dns"
)

type Api struct {
    srv *dns.Server
    cfg *dns.Config
}

func BindRoutes(r *chi.Mux, srv *dns.Server, cfg *dns.Config) {
    api := &Api{srv: srv, cfg: cfg}
    r.Get("/api/health", api.health)
    r.Post("/api/reload", api.reload)
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


