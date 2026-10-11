package main

import (
	"bytes"
	_ "embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var (
	//go:embed erp.html
	erpTmpl string
	//go:embed wsock.html
	wsockHTML []byte
	//go:embed favicon.ico
	favicon []byte
)

type site struct {
	Name  string
	Color string
}

// Per-site branding; selected with the SITE env var (default US).
var sites = map[string]site{
	"US": {"USA", "#0e9fc4"},
	"FR": {"FRANCE", "#c4810e"},
	"DE": {"GERMANY", "#750ec4"},
}

var upgrader = websocket.Upgrader{
	// Allowing connections from any origin
	CheckOrigin: func(r *http.Request) bool { return true },
}

// echoHandler echoes every message back. With ?hello=1 it first sends
// "HELLO <hostname>" so the client knows which backend holds the socket.
func echoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}
	defer conn.Close()
	wsActive.Add(1)
	defer wsActive.Add(-1)

	if r.URL.Query().Get("hello") != "" {
		if err := conn.WriteMessage(websocket.TextMessage, []byte("HELLO "+hostname)); err != nil {
			return
		}
	}

	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Println("Failed to read message:", err)
			}
			return
		}
		wsMessages.Add(1)
		if err := conn.WriteMessage(mt, message); err != nil {
			log.Println("Failed to write message:", err)
			return
		}
	}
}

func serveBytes(contentType string, body []byte) http.HandlerFunc {
	modTime := time.Now()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		http.ServeContent(w, r, "", modTime, bytes.NewReader(body))
	}
}

var (
	siteCode string
	siteInfo site
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "loadgen" {
		loadgen(os.Args[2:])
		return
	}

	siteCode = strings.ToUpper(os.Getenv("SITE"))
	s, ok := sites[siteCode]
	if !ok {
		siteCode, s = "US", sites["US"]
	}
	siteInfo = s
	var buf bytes.Buffer
	if err := template.Must(template.New("erp").Parse(erpTmpl)).Execute(&buf, s); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/{$}", serveBytes("text/html; charset=utf-8", buf.Bytes()))
	mux.HandleFunc("/wsock.html", serveBytes("text/html; charset=utf-8", wsockHTML))
	mux.HandleFunc("/favicon.ico", serveBytes("image/x-icon", favicon))
	mux.HandleFunc("/sock", echoHandler)

	mux.HandleFunc("/info", infoHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/health/fail", setHealth(false))
	mux.HandleFunc("/health/ok", setHealth(true))
	mux.HandleFunc("/slow", slowHandler)
	mux.HandleFunc("/error", errorHandler)
	mux.HandleFunc("/cpu", cpuHandler)
	mux.HandleFunc("/search", wafHandler)
	mux.HandleFunc("/login", wafHandler)
	mux.HandleFunc("/session", sessionHandler)
	mux.HandleFunc("/metrics", metricsHandler)
	for _, m := range modules {
		mux.HandleFunc("/"+m.Path, moduleHandler(m))
		mux.HandleFunc("/api/"+m.Path, moduleAPIHandler(m))
	}

	handler := observe(mux)
	go serveTLS(handler)
	log.Printf("Starting ERP site %s on :8080 (HTTP) and :8443 (HTTPS/HTTP2)", siteCode)
	srv := &http.Server{Addr: ":8080", Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(srv.ListenAndServe())
}
