package main

// Access logging and Prometheus metrics (text format, no extra dependency).

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var (
	accessLog  = os.Getenv("ACCESS_LOG") != "0"
	wsActive   atomic.Int64
	wsMessages atomic.Uint64

	metricsMu sync.Mutex
	reqCount  = map[[2]string]uint64{} // {route, status code}
	reqSecs   = map[string]float64{}   // route -> total seconds
	reqTotal  = map[string]uint64{}    // route -> requests
)

type recorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *recorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Hijack is required for the WebSocket upgrade.
func (r *recorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	r.status = http.StatusSwitchingProtocols
	return http.NewResponseController(r.ResponseWriter).Hijack()
}

func (r *recorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &recorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		dur := time.Since(start)
		if rec.status == 0 {
			rec.status = http.StatusOK
		}

		route := r.Pattern // set by ServeMux; keeps label cardinality low
		if route == "" {
			route = "unmatched"
		}
		metricsMu.Lock()
		reqCount[[2]string{route, fmt.Sprint(rec.status)}]++
		reqSecs[route] += dur.Seconds()
		reqTotal[route]++
		metricsMu.Unlock()

		if accessLog {
			xff := r.Header.Get("X-Forwarded-For")
			if xff == "" {
				xff = "-"
			}
			log.Printf("%s xff=%s %q %q %s %d %d %.3fms %q",
				r.RemoteAddr, xff, r.Host, r.Method+" "+r.RequestURI, r.Proto, rec.status, rec.bytes,
				float64(dur.Microseconds())/1000, r.UserAgent())
		}
	})
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	labels := fmt.Sprintf(`site=%q,server=%q`, siteCode, hostname)

	metricsMu.Lock()
	defer metricsMu.Unlock()
	keys := make([][2]string, 0, len(reqCount))
	for k := range reqCount {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i][0]+keys[i][1] < keys[j][0]+keys[j][1] })
	routes := make([]string, 0, len(reqTotal))
	for k := range reqTotal {
		routes = append(routes, k)
	}
	sort.Strings(routes)

	fmt.Fprintln(w, "# TYPE erp_http_requests_total counter")
	for _, k := range keys {
		fmt.Fprintf(w, "erp_http_requests_total{%s,route=%q,code=%q} %d\n", labels, k[0], k[1], reqCount[k])
	}
	fmt.Fprintln(w, "# TYPE erp_http_request_duration_seconds summary")
	for _, rt := range routes {
		fmt.Fprintf(w, "erp_http_request_duration_seconds_sum{%s,route=%q} %g\n", labels, rt, reqSecs[rt])
		fmt.Fprintf(w, "erp_http_request_duration_seconds_count{%s,route=%q} %d\n", labels, rt, reqTotal[rt])
	}
	fmt.Fprintf(w, "# TYPE erp_websocket_connections gauge\nerp_websocket_connections{%s} %d\n", labels, wsActive.Load())
	fmt.Fprintf(w, "# TYPE erp_websocket_messages_total counter\nerp_websocket_messages_total{%s} %d\n", labels, wsMessages.Load())
	up := 0
	if healthy.Load() {
		up = 1
	}
	fmt.Fprintf(w, "# TYPE erp_healthy gauge\nerp_healthy{%s} %d\n", labels, up)
}
