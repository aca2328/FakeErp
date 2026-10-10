package main

// Lab endpoints used to demonstrate Avi Load Balancer features.

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

var (
	hostname, _ = os.Hostname()
	healthy     atomic.Bool
)

func init() { healthy.Store(true) }

// queryInt returns the integer query parameter name, clamped to [min, max].
func queryInt(r *http.Request, name string, def, min, max int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil {
		return def
	}
	return clamp(v, min, max)
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// infoHandler reports which backend served the request and what the load
// balancer passed along (client IP, X-Forwarded-* headers, ...).
func infoHandler(code string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serverAddr := ""
		if a, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr); ok {
			serverAddr = a.String()
		}
		clientIP, _, _ := net.SplitHostPort(r.RemoteAddr)
		headers := map[string]string{}
		for k := range r.Header {
			headers[k] = r.Header.Get(k)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]any{
			"site":              code,
			"server_hostname":   hostname,
			"server_address":    serverAddr,
			"client_ip":         clientIP,
			"x_forwarded_for":   r.Header.Get("X-Forwarded-For"),
			"x_forwarded_proto": r.Header.Get("X-Forwarded-Proto"),
			"host":              r.Host,
			"method":            r.Method,
			"uri":               r.RequestURI,
			"proto":             r.Proto,
			"tls":               r.TLS != nil,
			"healthy":           healthy.Load(),
			"time":              time.Now().Format(time.RFC3339Nano),
			"headers":           headers,
		})
	}
}

// healthHandler answers health monitors; /health/fail and /health/ok flip it.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !healthy.Load() {
		http.Error(w, "DOWN "+hostname, http.StatusServiceUnavailable)
		return
	}
	fmt.Fprintln(w, "OK", hostname)
}

func setHealth(v bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		healthy.Store(v)
		healthHandler(w, r)
	}
}

// slowHandler delays the response: /slow?ms=2000 (max 60s).
func slowHandler(w http.ResponseWriter, r *http.Request) {
	ms := queryInt(r, "ms", 1000, 0, 60000)
	select {
	case <-time.After(time.Duration(ms) * time.Millisecond):
		fmt.Fprintf(w, "%s answered after %d ms\n", hostname, ms)
	case <-r.Context().Done():
	}
}

// errorHandler returns the requested status: /error?code=503.
func errorHandler(w http.ResponseWriter, r *http.Request) {
	code := queryInt(r, "code", 500, 400, 599)
	http.Error(w, fmt.Sprintf("%d %s from %s", code, http.StatusText(code), hostname), code)
}

// cpuHandler burns CPU on one core: /cpu?ms=500 (max 10s).
func cpuHandler(w http.ResponseWriter, r *http.Request) {
	ms := queryInt(r, "ms", 500, 0, 10000)
	deadline := time.Now().Add(time.Duration(ms) * time.Millisecond)
	n := 0
	for time.Now().Before(deadline) && r.Context().Err() == nil {
		n++
	}
	fmt.Fprintf(w, "%s burned %d ms of CPU (%d iterations)\n", hostname, ms, n)
}

// Deliberately "vulnerable-looking" pages to exercise the Avi WAF. Input is
// only reflected (HTML-escaped) and never executed; no real database exists.
var wafTmpl = template.Must(template.New("waf").Parse(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>ERP - Recherche</title></head>
<body style="font-family: Arial, sans-serif">
<h2>Recherche clients</h2>
<form action="/search"><input name="q" value="{{.Q}}" size="40"> <button>Rechercher</button></form>
{{if .Q}}<p>Requête SQL : <code>SELECT * FROM customers WHERE name = '{{.Q}}'</code></p>
<p>Résultat pour : {{.Q}} — 0 client trouvé (servi par {{.Host}})</p>{{end}}
<h2>Connexion</h2>
<form action="/login" method="post">
<input name="user" placeholder="utilisateur"> <input name="password" type="password" placeholder="mot de passe"> <button>Connexion</button>
</form>
{{if .Login}}<p>Identifiants invalides pour l'utilisateur : {{.Login}}</p>{{end}}
<p><a href="/">Accueil</a></p>
</body></html>`))

func wafHandler(w http.ResponseWriter, r *http.Request) {
	data := struct{ Q, Login, Host string }{Q: r.FormValue("q"), Host: hostname}
	if r.URL.Path == "/login" && r.Method == http.MethodPost {
		data.Login = r.FormValue("user")
		w.WriteHeader(http.StatusUnauthorized)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	wafTmpl.Execute(w, data)
}
